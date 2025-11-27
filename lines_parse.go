package ubl

import (
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

func (in *Invoice) goblAddLines(out *bill.Invoice) error {
	items := in.InvoiceLines
	if len(in.CreditNoteLines) > 0 {
		items = in.CreditNoteLines
	}

	out.Lines = make([]*bill.Line, 0, len(items))

	for _, docLine := range items {
		line, err := goblConvertLine(&docLine)
		if err != nil {
			return err
		}
		if line != nil {
			out.Lines = append(out.Lines, line)
		}
	}

	return nil
}

func goblConvertLine(docLine *InvoiceLine) (*bill.Line, error) {
	if docLine.Price == nil {
		// skip this line
		return nil, nil
	}
	price, err := num.AmountFromString(docLine.Price.PriceAmount.Value)
	if err != nil {
		return nil, err
	}

	if docLine.Price.BaseQuantity != nil {
		// Base quantity is the number of item units to which the price applies
		baseQuantity, err := num.AmountFromString(docLine.Price.BaseQuantity.Value)
		if err != nil {
			return nil, err
		}
		price = price.Divide(baseQuantity)
	}

	line := &bill.Line{
		Quantity: num.MakeAmount(1, 0),
		Item: &org.Item{
			Price: &price,
		},
	}
	if di := docLine.Item; di != nil {
		goblConvertLineItem(di, line.Item)
		goblConvertLineItemTaxes(di, line)
	}

	notes := make([]*org.Note, 0)

	iq := docLine.InvoicedQuantity
	if docLine.CreditedQuantity != nil {
		iq = docLine.CreditedQuantity
	}
	if iq != nil {
		line.Quantity, err = num.AmountFromString(iq.Value)
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
					Text: note,
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

func goblConvertLineItem(di *Item, item *org.Item) {
	if di.Name != "" {
		item.Name = di.Name
	}
	if di.Description != nil {
		item.Description = *di.Description
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
				item.Meta[key] = property.Value
			}
		}
	}
}

func goblConvertLineItemTaxes(di *Item, line *bill.Line) {
	ctc := di.ClassifiedTaxCategory
	if ctc == nil || ctc.TaxScheme == nil {
		return
	}

	line.Taxes = tax.Set{
		{
			Category: cbc.Code(ctc.TaxScheme.ID),
		},
	}
	if ctc.ID != nil {
		line.Taxes[0].Ext = tax.Extensions{
			untdid.ExtKeyTaxCategory: cbc.Code(*ctc.ID),
		}
	}
	if ctc.Percent != nil {
		percentStr := *ctc.Percent
		if !strings.HasSuffix(percentStr, "%") {
			percentStr += "%"
		}
		percent, _ := num.PercentageFromString(percentStr)
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
