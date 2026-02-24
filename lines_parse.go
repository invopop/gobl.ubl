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

func (ui *Invoice) goblAddLines(out *bill.Invoice) error {
	items := ui.InvoiceLines
	if len(ui.CreditNoteLines) > 0 {
		items = ui.CreditNoteLines
	}

	out.Lines = make([]*bill.Line, 0, len(items))

	// Build tax category map from TaxTotal
	taxCategoryMap := ui.buildTaxCategoryMap()

	for _, docLine := range items {
		line, err := goblConvertLine(&docLine, taxCategoryMap)
		if err != nil {
			return err
		}
		if line != nil {
			out.Lines = append(out.Lines, line)
		}
	}

	return nil
}

func goblConvertLine(docLine *InvoiceLine, taxCategoryMap map[string]*taxCategoryInfo) (*bill.Line, error) {
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
		goblConvertLineItem(di, line.Item)
		goblConvertLineItemTaxes(di, line, taxCategoryMap)
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
				notes = append(notes, &org.Note{
					Text: cleanString(note),
				})
			}
		}
	}

	if docLine.AccountingCost != nil {
		// BT-133
		line.Cost = cbc.Code(*docLine.AccountingCost)
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

func goblConvertLineItem(di *Item, item *org.Item) {
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
		item.Meta = make(cbc.Meta)
		for _, property := range *di.AdditionalItemProperty {
			if property.Name != "" && property.Value != "" {
				key := formatKey(property.Name)
				item.Meta[key] = cleanString(property.Value)
			}
		}
	}
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
		line.Taxes[0].Ext = tax.Extensions{
			untdid.ExtKeyTaxCategory: cbc.Code(ctc.ID.Value),
		}

		// Look up exemption code from TaxTotal if not present in line
		if ctc.TaxExemptionReasonCode != nil {
			line.Taxes[0].Ext[cef.ExtKeyVATEX] = cbc.Code(*ctc.TaxExemptionReasonCode)
		} else {
			// Try to get exemption code from TaxTotal
			key := buildTaxCategoryKey(ctc.TaxScheme.ID.Value, ctc.ID.Value)
			if info, ok := taxCategoryMap[key]; ok && info.exemptionReasonCode != "" {
				line.Taxes[0].Ext[cef.ExtKeyVATEX] = cbc.Code(info.exemptionReasonCode)
			}
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
			Ext: tax.Extensions{
				iso.ExtKeySchemeID: cbc.Code(s),
			},
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
