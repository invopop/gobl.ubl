package ubl

import (
	"errors"
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

func goblAddLines(in *Invoice, out *bill.Invoice) error {
	items := in.InvoiceLines
	if len(in.CreditNoteLines) > 0 {
		items = in.CreditNoteLines
	}

	lines := make([]*bill.Line, 0, len(items))

	for _, docLine := range items {
		line, err := goblLine(&docLine)
		if err != nil {
			return err
		}
		lines = append(lines, line)
	}
	out.Lines = lines
	return nil
}

func goblLine(docLine *InvoiceLine) (*bill.Line, error) {
	if docLine.Price == nil {
		return nil, errors.New("invalid input: invoice line price is required")
	}
	if docLine.Item == nil {
		return nil, errors.New("invalid input: invoice line item is required")
	}

	price, err := num.AmountFromString(docLine.Price.PriceAmount.Value)
	if err != nil {
		return nil, err
	}
	line := &bill.Line{
		Quantity: num.MakeAmount(1, 0),
		Item: &org.Item{
			Name:  docLine.Item.Name,
			Price: &price,
		},
	}

	ids := make([]*org.Identity, 0)
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

	if docLine.Item.SellersItemIdentification != nil && docLine.Item.SellersItemIdentification.ID != nil {
		line.Item.Ref = cbc.Code(docLine.Item.SellersItemIdentification.ID.Value)
	}

	// As there is no specific GOBL field for BT-133, we use a note to store it
	if docLine.AccountingCost != nil {
		notes = append(notes, &org.Note{
			Key:  "buyer-accounting-ref",
			Text: *docLine.AccountingCost,
		})
	}

	line.Item.Identities = goblIdentities(docLine)

	if docLine.Item.Description != nil {
		line.Item.Description = *docLine.Item.Description
	}

	if docLine.Item.OriginCountry != nil {
		line.Item.Origin = l10n.ISOCountryCode(docLine.Item.OriginCountry.IdentificationCode)
	}

	if ctc := docLine.Item.ClassifiedTaxCategory; ctc != nil && ctc.TaxScheme != nil {
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

	if docLine.AllowanceCharge != nil {
		line, err = goblLineCharges(docLine.AllowanceCharge, line)
		if err != nil {
			return nil, err
		}
	}

	if docLine.Item.AdditionalItemProperty != nil {
		line.Item.Meta = make(cbc.Meta)
		for _, property := range *docLine.Item.AdditionalItemProperty {
			if property.Name != "" && property.Value != "" {
				key := formatKey(property.Name)
				line.Item.Meta[key] = property.Value
			}
		}
	}

	if len(ids) > 0 {
		line.Item.Identities = ids
	}

	if len(notes) > 0 {
		line.Notes = notes
	}

	return line, nil
}

func goblIdentities(docLine *InvoiceLine) []*org.Identity {
	ids := make([]*org.Identity, 0)

	if docLine.Item.BuyersItemIdentification != nil && docLine.Item.BuyersItemIdentification.ID != nil {
		id := goblIdentity(docLine.Item.BuyersItemIdentification.ID)
		if id != nil {
			ids = append(ids, id)
		}
	}

	if docLine.Item.StandardItemIdentification != nil &&
		docLine.Item.StandardItemIdentification.ID != nil &&
		docLine.Item.StandardItemIdentification.ID.SchemeID != nil {
		s := *docLine.Item.StandardItemIdentification.ID.SchemeID
		id := &org.Identity{
			Ext: tax.Extensions{
				iso.ExtKeySchemeID: cbc.Code(s),
			},
			Code: cbc.Code(docLine.Item.StandardItemIdentification.ID.Value),
		}

		ids = append(ids, id)

	}

	if docLine.Item.CommodityClassification != nil && len(*docLine.Item.CommodityClassification) > 0 {
		for _, classification := range *docLine.Item.CommodityClassification {
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
