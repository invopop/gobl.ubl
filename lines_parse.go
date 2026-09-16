package ubl

import (
	"fmt"
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

func (ui *Invoice) goblAddLines(out *bill.Invoice, o *options) error {
	items := ui.InvoiceLines
	if len(ui.CreditNoteLines) > 0 {
		items = ui.CreditNoteLines
	}

	out.Lines = make([]*bill.Line, 0, len(items))

	// Build tax category map from TaxTotal
	taxCategoryMap := ui.buildTaxCategoryMap()

	for _, docLine := range items {
		line, err := goblConvertLine(&docLine, taxCategoryMap, o)
		if err != nil {
			return err
		}
		if line != nil {
			out.Lines = append(out.Lines, line)
		}
	}

	return nil
}

func goblConvertLine(docLine *InvoiceLine, taxCategoryMap map[string]*taxCategoryInfo, o *options) (*bill.Line, error) {
	if docLine.Price == nil {
		// skip this line
		return nil, nil
	}
	price, err := num.AmountFromString(normalizeNumericString(docLine.Price.PriceAmount.Value))
	if err != nil {
		return nil, err
	}

	if docLine.Price.BaseQuantity != nil {
		// Base quantity is the number of item units to which the price applies
		baseQuantity, err := num.AmountFromString(normalizeNumericString(docLine.Price.BaseQuantity.Value))
		if err != nil {
			return nil, err
		}
		if !baseQuantity.IsZero() {
			// Calculate required precision dynamically to avoid rounding errors
			// Formula: price_decimals + ceil(log10(base_quantity))
			precision := calculateRequiredPrecision(price, baseQuantity)
			price = price.RescaleUp(precision).Divide(baseQuantity)
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
			line.Item.Unit = goblUnitFromUNECE(cbc.Code(iq.UnitCode))
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

	if docLine.AllowanceCharge != nil {
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
			attr.Unit = goblUnitFromUNECE(cbc.Code(property.ValueQuantity.UnitCode))
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
			identity.Label = *field
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

// NoteSrcReconciliation marks notes generated during conversion rather than
// sent by the issuer. Shared with gobl.cii; stripped again on re-export.
const NoteSrcReconciliation cbc.Key = "reconciliation"

// lineTotalTolerance matches the slack PEPPOL-EN16931-R120 allows on BT-131.
var lineTotalTolerance = num.MakeAmount(2, 2)

// reconcileLines rebuilds any line whose price and quantity disagree with the
// total the issuer stated. BR-CO-10 ties the document totals to BT-131, but
// nothing in EN 16931 checks BT-131 against the line's own components, so the
// stated total is the figure to trust.
func (ui *Invoice) reconcileLines(out *bill.Invoice) error {
	if err := out.Calculate(); err != nil {
		return err
	}
	items := ui.InvoiceLines
	if len(ui.CreditNoteLines) > 0 {
		items = ui.CreditNoteLines
	}
	var rebuilt bool
	for i := range items {
		if i >= len(out.Lines) {
			break
		}
		ok, err := trustStatedLineTotal(&items[i], out.Lines[i])
		if err != nil {
			return fmt.Errorf("line %d: %w", i, err)
		}
		rebuilt = rebuilt || ok
	}
	if !rebuilt {
		return nil
	}
	return out.Calculate()
}

func trustStatedLineTotal(item *InvoiceLine, l *bill.Line) (bool, error) {
	if item.LineExtensionAmount.Value == "" || l.Total == nil || l.Item == nil {
		return false, nil
	}
	stated, err := num.AmountFromString(normalizeNumericString(item.LineExtensionAmount.Value))
	if err != nil {
		return false, fmt.Errorf("parsing BT-131: %w", err)
	}
	if withinTolerance(*l.Total, stated) || l.Quantity.IsZero() {
		return false, nil
	}

	// Deriving the price from BT-131 means the line's own allowances and charges
	// are already in it, so record them before dropping them.
	note := fmt.Sprintf("Line net amount as received: %s. Price and quantity as sent gave %s%s.",
		stated.String(), l.Total.String(), absorbedAmounts(l))
	price := stated.RescaleUp(stated.Exp() + 4).Divide(l.Quantity)
	l.Item.Price = &price
	l.Discounts = nil
	l.Charges = nil
	l.Notes = append(l.Notes, &org.Note{
		Key:  org.NoteKeyGeneral,
		Src:  NoteSrcReconciliation,
		Text: note,
	})
	return true, nil
}

func absorbedAmounts(l *bill.Line) string {
	var alw, chg num.Amount
	for _, d := range l.Discounts {
		alw = alw.MatchPrecision(d.Amount).Add(d.Amount)
	}
	for _, c := range l.Charges {
		chg = chg.MatchPrecision(c.Amount).Add(c.Amount)
	}
	switch {
	case !alw.IsZero() && !chg.IsZero():
		return fmt.Sprintf(", absorbing an allowance of %s and a charge of %s", alw, chg)
	case !alw.IsZero():
		return fmt.Sprintf(", absorbing an allowance of %s", alw)
	case !chg.IsZero():
		return fmt.Sprintf(", absorbing a charge of %s", chg)
	}
	return ""
}

func withinTolerance(a, b num.Amount) bool {
	a = a.MatchPrecision(b)
	b = b.MatchPrecision(a)
	return a.Subtract(b).Abs().Compare(lineTotalTolerance.MatchPrecision(a)) <= 0
}

// addRounding maps BT-114. Totals.Rounding is the one total GOBL does not
// reset when recalculating.
func (ui *Invoice) addRounding(out *bill.Invoice) error {
	a := ui.LegalMonetaryTotal.PayableRoundingAmount
	if a == nil || a.Value == "" {
		return nil
	}
	r, err := num.AmountFromString(normalizeNumericString(a.Value))
	if err != nil {
		return fmt.Errorf("parsing BT-114: %w", err)
	}
	if r.IsZero() {
		return nil
	}
	if out.Totals == nil {
		out.Totals = new(bill.Totals)
	}
	out.Totals.Rounding = &r
	return nil
}
