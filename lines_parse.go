package ubl

import (
	"math"
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/cef"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

// lineSource pairs a parsed line with the document line(s) it came from.
type lineSource struct {
	doc      *InvoiceLine
	children []*InvoiceLine
}

// Single-rate groups fold into breakdowns; anything else is read flat, in document order.
func (ui *Invoice) goblAddLines(out *bill.Invoice, o *options, flat map[string]bool) ([]lineSource, error) {
	items := ui.InvoiceLines
	if len(ui.CreditNoteLines) > 0 {
		items = ui.CreditNoteLines
	}

	// Build tax category map from TaxTotal
	taxCategoryMap := ui.buildTaxCategoryMap()

	// Only EXTENDED-CTC-FR gives billing references this meaning.
	extended := o.context.Is(ContextPeppolFranceExtended)
	docs := make([]*InvoiceLine, len(items))
	ids := make(map[string]bool)
	for i := range items {
		docs[i] = &items[i]
		docs[i].hierarchy = nil
		if extended {
			docs[i].hierarchy = goblLineHierarchy(docs[i], strings.TrimSpace(ui.ID))
		}
		if id := goblLineID(docs[i]); id != "" {
			ids[id] = true
		}
	}
	children := make(map[string][]*InvoiceLine)
	for _, it := range docs {
		if p := goblParentLineID(it); p != "" && ids[p] {
			children[p] = append(children[p], it)
		}
	}

	// Each child folds into one top-level parent at most.
	groups := make(map[*InvoiceLine][]*InvoiceLine)
	folded := make(map[*InvoiceLine]bool)
	for _, it := range docs {
		if p := goblParentLineID(it); p != "" && ids[p] {
			continue
		}
		id := goblLineID(it)
		kids := children[id]
		if len(kids) == 0 || flat[id] || goblAnyFolded(folded, kids) || !goblCanFold(it, kids, taxCategoryMap) {
			continue
		}
		groups[it] = kids
		for _, k := range kids {
			folded[k] = true
		}
	}

	out.Lines = make([]*bill.Line, 0, len(items))
	srcs := make([]lineSource, 0, len(items))
	for _, it := range docs {
		if folded[it] {
			continue
		}
		var l *bill.Line
		var err error
		kids, ok := groups[it]
		if ok {
			l, err = goblNewGroupLine(it, kids, taxCategoryMap, o)
		} else {
			l, err = goblConvertLine(it, taxCategoryMap, o)
		}
		if err != nil {
			return nil, err
		}
		if l == nil {
			continue
		}
		out.Lines = append(out.Lines, l)
		srcs = append(srcs, lineSource{doc: it, children: kids})
	}

	return srcs, nil
}

func goblAnyFolded(folded map[*InvoiceLine]bool, lines []*InvoiceLine) bool {
	for _, l := range lines {
		if folded[l] {
			return true
		}
	}
	return false
}

func goblLineID(it *InvoiceLine) string {
	return strings.TrimSpace(it.ID)
}

// Only the reference naming the invoice itself (BT-1) carries the hierarchy.
func goblLineHierarchy(it *InvoiceLine, self string) *LineBillingReference {
	for _, br := range it.BillingReference {
		if br != nil && br.InvoiceDocumentReference != nil && strings.TrimSpace(br.InvoiceDocumentReference.ID.Value) == self {
			return br
		}
	}
	return nil
}

func goblParentLineID(it *InvoiceLine) string {
	br := it.hierarchy
	if br == nil || br.BillingReferenceLine == nil {
		return ""
	}
	return strings.TrimSpace(br.BillingReferenceLine.ID.Value)
}

func goblLineStatus(it *InvoiceLine) string {
	br := it.hierarchy
	if br == nil || br.InvoiceDocumentReference == nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(br.InvoiceDocumentReference.DocumentStatusCode))
}

// Only DETAIL and untyped lines count towards the totals.
func goblLineIsSummed(it *InvoiceLine) bool {
	switch goblLineStatus(it) {
	case lineStatusGroup, lineStatusInformation:
		return false
	}
	return true
}

// goblCanFold checks what a breakdown can express: one tax, per-unit quantities, matching amounts.
func goblCanFold(parent *InvoiceLine, kids []*InvoiceLine, taxCategoryMap map[string]*taxCategoryInfo) bool {
	group := goblLineStatus(parent) == lineStatusGroup
	if group && len(parent.AllowanceCharge) > 0 {
		return false
	}
	if !group && parent.Price == nil {
		return false
	}
	if !group && !goblLineIsSummed(parent) {
		return false
	}
	qty, ok := goblLineQuantity(parent)
	if !ok {
		return false
	}

	taxKey := ""
	detail := 0
	sum := num.AmountZero
	for _, k := range kids {
		q, ok := goblLineQuantity(k)
		if !ok || !goblDividesBy(q, qty) {
			return false
		}
		switch goblLineStatus(k) {
		case lineStatusGroup:
			return false
		case lineStatusInformation:
			continue
		}
		if !group || k.Price == nil {
			return false
		}
		detail++
		key := goblLineTaxKey(k, taxCategoryMap)
		if key == "" || (taxKey != "" && key != taxKey) {
			return false
		}
		taxKey = key

		for _, ac := range k.AllowanceCharge {
			for _, a := range []*Amount{&ac.Amount, ac.BaseAmount} {
				if a == nil {
					continue
				}
				if v, ok := goblDeclaredAmount(*a); ok && !goblDividesBy(v, qty) {
					return false
				}
			}
		}
		amount, ok := goblDeclaredAmount(k.LineExtensionAmount)
		if !ok {
			return false
		}
		sum = sum.MatchPrecision(amount).Add(amount)
	}
	if group && detail == 0 {
		return false
	}
	if declared, ok := goblDeclaredAmount(parent.LineExtensionAmount); ok && group && !declared.Equals(sum) {
		return false
	}
	return true
}

func goblDividesBy(a, qty num.Amount) bool {
	return a.Divide(qty).Multiply(qty).Equals(a)
}

func goblLineQuantity(it *InvoiceLine) (num.Amount, bool) {
	iq := it.InvoicedQuantity
	if it.CreditedQuantity != nil {
		iq = it.CreditedQuantity
	}
	if iq == nil || strings.TrimSpace(iq.Value) == "" {
		return num.MakeAmount(1, 0), true
	}
	q, err := num.AmountFromString(normalizeNumericString(iq.Value))
	if err != nil || q.IsZero() {
		return q, false
	}
	return q, true
}

func goblLineTaxKey(it *InvoiceLine, taxCategoryMap map[string]*taxCategoryInfo) string {
	if it.Item == nil || it.Item.ClassifiedTaxCategory == nil || it.Item.ClassifiedTaxCategory.TaxScheme == nil {
		return ""
	}
	ctc := it.Item.ClassifiedTaxCategory
	cat := ""
	if ctc.ID != nil {
		cat = ctc.ID.Value
	}
	key := buildTaxCategoryKey(ctc.TaxScheme.ID.Value, cat, ctc.Percent)
	exemption := ""
	if info, ok := taxCategoryMap[key]; ok {
		exemption = info.exemptionReasonCode
	}
	return key + ":" + normalizeTaxPercent(ctc.Percent) + ":" + exemption
}

// The inverse of newGroupLines.
func goblNewGroupLine(parent *InvoiceLine, kids []*InvoiceLine, taxCategoryMap map[string]*taxCategoryInfo, o *options) (*bill.Line, error) {
	l, err := goblConvertLine(parent, taxCategoryMap, o)
	if err != nil {
		return nil, err
	}
	qty := l.Quantity
	group := goblLineStatus(parent) == lineStatusGroup
	taxed := false

	for _, k := range kids {
		cl, err := goblConvertLine(k, taxCategoryMap, o)
		if err != nil {
			return nil, err
		}
		if cl == nil {
			continue
		}
		sl := &bill.SubLine{
			Quantity:   cl.Quantity.Divide(qty),
			Identifier: cl.Identifier,
			Period:     cl.Period,
			Order:      cl.Order,
			Cost:       cl.Cost,
			Item:       cl.Item,
			Discounts:  cl.Discounts,
			Charges:    cl.Charges,
			Notes:      cl.Notes,
		}
		for _, d := range sl.Discounts {
			d.Amount = d.Amount.Divide(qty)
			if d.Base != nil {
				b := d.Base.Divide(qty)
				d.Base = &b
			}
		}
		for _, c := range sl.Charges {
			c.Amount = c.Amount.Divide(qty)
			if c.Base != nil {
				b := c.Base.Divide(qty)
				c.Base = &b
			}
		}
		if !goblLineIsSummed(k) {
			sl.Item.Price = nil
		} else if group && !taxed {
			l.Taxes = cl.Taxes
			taxed = true
		}
		l.Breakdown = append(l.Breakdown, sl)
	}
	return l, nil
}

func goblConvertLine(docLine *InvoiceLine, taxCategoryMap map[string]*taxCategoryInfo, o *options) (*bill.Line, error) {
	// GOBL requires a price; the DETAIL lines carry the amounts.
	summed := goblLineIsSummed(docLine)
	price := num.AmountZero
	if summed {
		if docLine.Price == nil {
			// skip this line
			return nil, nil
		}
		var err error
		price, err = goblLinePrice(docLine.Price)
		if err != nil {
			return nil, err
		}
	}

	line := &bill.Line{
		Quantity: num.MakeAmount(1, 0),
		Item: &org.Item{
			Price: &price,
		},
	}
	if di := docLine.Item; di != nil {
		if err := goblConvertLineItem(di, line.Item); err != nil {
			return nil, err
		}
		goblConvertLineItemTaxes(di, line, taxCategoryMap)
		if di.ManufacturerParty != nil {
			line.Seller = goblParty(di.ManufacturerParty, o)
		}
	}

	notes := make([]*org.Note, 0)

	var err error
	iq := docLine.InvoicedQuantity
	if docLine.CreditedQuantity != nil {
		iq = docLine.CreditedQuantity
	}
	if iq != nil {
		line.Quantity, err = num.AmountFromString(normalizeNumericString(iq.Value))
		if err != nil {
			return nil, err
		}

		if iq.UnitCode != "" {
			line.Item.Ext, line.Item.Unit = goblUnit(line.Item.Ext, cbc.Code(iq.UnitCode))
		}
	}

	if len(docLine.Note) > 0 {
		for _, note := range docLine.Note {
			if note != "" {
				notes = append(notes, parseNote(note))
			}
		}
	}

	if docLine.AccountingCost != nil {
		// BT-133
		line.Cost = cbc.Code(*docLine.AccountingCost)
	}

	// BT-128: Invoice line object identifier
	if docLine.DocumentReference != nil && docLine.DocumentReference.ID.Value != "" {
		line.Identifier = &org.Identity{
			Code: cbc.Code(docLine.DocumentReference.ID.Value),
		}
		if docLine.DocumentReference.ID.SchemeID != nil {
			line.Identifier.Ext = tax.ExtensionsOf(cbc.CodeMap{
				untdid.ExtKeyReference: cbc.Code(*docLine.DocumentReference.ID.SchemeID),
			})
		}
	}

	if docLine.InvoicePeriod != nil {
		line.Period, err = goblPeriodDates(docLine.InvoicePeriod)
		if err != nil {
			return nil, err
		}
	}

	if docLine.OrderLineReference != nil && docLine.OrderLineReference.LineID != "" {
		line.Order = cbc.Code(docLine.OrderLineReference.LineID)
	}

	// Allowances on an uncounted line count nowhere either.
	if docLine.AllowanceCharge != nil && summed {
		line, err = goblLineCharges(docLine.AllowanceCharge, line)
		if err != nil {
			return nil, err
		}
	}

	if len(notes) > 0 {
		line.Notes = notes
	}
	return line, nil
}

func goblLinePrice(p *Price) (num.Amount, error) {
	price, err := num.AmountFromString(normalizeNumericString(p.PriceAmount.Value))
	if err != nil {
		return price, err
	}

	if p.BaseQuantity != nil {
		// Base quantity is the number of item units to which the price applies
		baseQuantity, err := num.AmountFromString(normalizeNumericString(p.BaseQuantity.Value))
		if err != nil {
			return price, err
		}
		if !baseQuantity.IsZero() {
			// Calculate required precision dynamically to avoid rounding errors
			// Formula: price_decimals + ceil(log10(base_quantity))
			precision := calculateRequiredPrecision(price, baseQuantity)
			price = price.RescaleUp(precision).Divide(baseQuantity)
		}
	}
	return price, nil
}

// calculateRequiredPrecision determines the decimal precision needed when
// dividing a price by a base quantity to avoid rounding errors.
// Formula: price_decimals + ceil(log10(base_quantity))
// Example: price with 2 decimals divided by 100 needs 2 + 2 = 4 decimals
func calculateRequiredPrecision(price, baseQuantity num.Amount) uint32 {
	priceExp := price.Exp()

	// Convert baseQuantity to a whole number to calculate needed decimal places
	baseQtyNormalized := baseQuantity.Rescale(0)
	baseQtyFloat := math.Abs(float64(baseQtyNormalized.Value()))

	additionalDecimals := uint32(0)
	if baseQtyFloat > 1 {
		// log10(100) = 2, log10(1000) = 3, etc.
		additionalDecimals = uint32(math.Ceil(math.Log10(baseQtyFloat)))
	}

	return priceExp + additionalDecimals
}

func goblConvertLineItem(di *Item, item *org.Item) error {
	if di.Name != "" {
		item.Name = cleanString(di.Name)
	}
	if di.Description != nil {
		item.Description = cleanString(*di.Description)
	}

	if di.OriginCountry != nil {
		item.Origin = l10n.ISOCountryCode(di.OriginCountry.IdentificationCode)
	}

	if di.SellersItemIdentification != nil && di.SellersItemIdentification.ID != nil {
		item.Ref = cbc.Code(di.SellersItemIdentification.ID.Value)
	}

	item.Identities = goblItemIdentities(di)

	if di.AdditionalItemProperty != nil {
		for _, property := range *di.AdditionalItemProperty {
			attr, err := goblItemAttribute(&property)
			if err != nil {
				return err
			}
			if attr != nil {
				item.Attributes = append(item.Attributes, attr)
			}
		}
	}

	return nil
}

// goblItemAttribute converts a UBL AdditionalItemProperty into a GOBL
// org.Attribute. NameCode isn't mapped: there's no corresponding slot on
// org.Attribute for it alongside a text/quantity value.
func goblItemAttribute(property *AdditionalItemProperty) (*org.Attribute, error) {
	if property.Name == "" {
		return nil, nil
	}
	attr := &org.Attribute{Label: cleanString(property.Name)}
	switch {
	case property.ValueQuantity != nil && property.ValueQuantity.Value != "":
		amount, err := num.AmountFromString(normalizeNumericString(property.ValueQuantity.Value))
		if err != nil {
			return nil, err
		}
		attr.Amount = &amount
		if property.ValueQuantity.UnitCode != "" {
			attr.Ext, attr.Unit = goblUnit(attr.Ext, cbc.Code(property.ValueQuantity.UnitCode))
		}
	case property.Value != "":
		attr.Text = cleanString(property.Value)
	default:
		return nil, nil
	}
	return attr, nil
}

func goblConvertLineItemTaxes(di *Item, line *bill.Line, taxCategoryMap map[string]*taxCategoryInfo) {
	ctc := di.ClassifiedTaxCategory
	if ctc == nil || ctc.TaxScheme == nil {
		return
	}

	line.Taxes = tax.Set{
		{
			Category: cbc.Code(ctc.TaxScheme.ID.Value),
		},
	}
	if ctc.ID != nil {
		line.Taxes[0].Ext = tax.ExtensionsOf(cbc.CodeMap{
			untdid.ExtKeyTaxCategory: cbc.Code(ctc.ID.Value),
		})

		// Try to get exemption code from TaxTotal
		key := buildTaxCategoryKey(ctc.TaxScheme.ID.Value, ctc.ID.Value, ctc.Percent)
		if info, ok := taxCategoryMap[key]; ok && info.exemptionReasonCode != "" {
			line.Taxes[0].Ext = line.Taxes[0].Ext.Set(cef.ExtKeyVATEX, cbc.Code(info.exemptionReasonCode))
		}

	}
	if ctc.Percent != nil {
		percentStr := normalizeNumericString(*ctc.Percent)
		if !strings.HasSuffix(percentStr, "%") {
			percentStr += "%"
		}
		percent, _ := num.PercentageFromString(percentStr)

		// Skip setting percent if it's 0% and tax category is not "Z" (zero-rated)
		// This prevents GOBL from normalizing to "zero" tax rate for exempt/reverse-charge cases
		if percent.IsZero() && ctc.ID != nil && ctc.ID.Value != "Z" {
			return
		}

		if line.Taxes == nil {
			line.Taxes = make([]*tax.Combo, 1)
			line.Taxes[0] = &tax.Combo{}
		}
		line.Taxes[0].Percent = &percent
	}
}

func goblItemIdentities(di *Item) []*org.Identity {
	ids := make([]*org.Identity, 0)

	if di.BuyersItemIdentification != nil && di.BuyersItemIdentification.ID != nil {
		id := goblIdentity(di.BuyersItemIdentification.ID)
		if id != nil {
			ids = append(ids, id)
		}
	}

	if di.StandardItemIdentification != nil &&
		di.StandardItemIdentification.ID != nil &&
		di.StandardItemIdentification.ID.SchemeID != nil {
		s := *di.StandardItemIdentification.ID.SchemeID
		id := &org.Identity{
			Ext: tax.ExtensionsOf(cbc.CodeMap{
				iso.ExtKeySchemeID: cbc.Code(s),
			}),
			Code: cbc.Code(di.StandardItemIdentification.ID.Value),
		}

		ids = append(ids, id)

	}

	if di.CommodityClassification != nil && len(*di.CommodityClassification) > 0 {
		for _, classification := range *di.CommodityClassification {
			id := goblIdentity(classification.ItemClassificationCode)
			if id != nil {
				ids = append(ids, id)
			}
		}
	}

	return ids
}

func goblIdentity(id *IDType) *org.Identity {
	if id == nil {
		return nil
	}
	identity := &org.Identity{
		Code: cbc.Code(id.Value),
	}
	for _, field := range []*string{id.SchemeID, id.ListID, id.ListVersionID, id.SchemeName, id.Name} {
		if field != nil {
			identity.Label = cleanString(*field)
			break
		}
	}
	return identity
}

func goblLineCharges(allowances []*AllowanceCharge, line *bill.Line) (*bill.Line, error) {
	for _, ac := range allowances {
		if ac.ChargeIndicator {
			charge, err := goblLineCharge(ac)
			if err != nil {
				return nil, err
			}
			if line.Charges == nil {
				line.Charges = make([]*bill.LineCharge, 0)
			}
			line.Charges = append(line.Charges, charge)
		} else {
			discount, err := goblLineDiscount(ac)
			if err != nil {
				return nil, err
			}
			if line.Discounts == nil {
				line.Discounts = make([]*bill.LineDiscount, 0)
			}
			line.Discounts = append(line.Discounts, discount)
		}
	}
	return line, nil
}
