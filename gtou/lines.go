package gtou

import (
	"strconv"

	"github.com/invopop/gobl/bill"
)

func (c *Conversor) newLines(inv *bill.Invoice) error {
	if len(inv.Lines) == 0 {
		return nil
	}

	var invoiceLines []InvoiceLine

	for _, line := range inv.Lines {
		invoiceLine := InvoiceLine{
			ID: strconv.Itoa(line.Index),
			InvoicedQuantity: &Quantity{
				UnitCode: string(line.Item.Unit.UNECE()),
				Value:    line.Quantity.String(),
			},
			LineExtensionAmount: Amount{
				CurrencyID: line.Item.Currency.String(),
				Value:      line.Total.String(),
			},
		}

		if len(line.Notes) > 0 {
			var notes []string
			for _, note := range line.Notes {
				notes = append(notes, note.Text)
			}
			invoiceLine.Note = notes
		}

		if len(line.Charges) > 0 || len(line.Discounts) > 0 {
			invoiceLine.AllowanceCharge = makeLineCharges(line.Charges, line.Discounts)
		}

		if line.Item != nil {
			item := &Item{}

			if line.Item.Description != "" {
				item.Description = line.Item.Description
			}

			if line.Item.Name != "" {
				item.Name = line.Item.Name
			}

			if line.Item.Origin != "" {
				item.OriginCountry = &Country{
					IdentificationCode: line.Item.Origin.String(),
				}
			}

			if line.Item.Meta != nil {
				var properties []AdditionalItemProperty
				for key, value := range line.Item.Meta {
					properties = append(properties, AdditionalItemProperty{Name: key.String(), Value: value})
				}
				item.AdditionalItemProperty = &properties
			}

			invoiceLine.Item = item
		}

		invoiceLines = append(invoiceLines, invoiceLine)
	}
	c.doc.InvoiceLine = invoiceLines
	return nil
}

func makeLineCharges(charges []*bill.LineCharge, discounts []*bill.LineDiscount) []*AllowanceCharge {
	var allowanceCharges []*AllowanceCharge
	for _, charge := range charges {
		allowanceCharge := &AllowanceCharge{
			ChargeIndicator: true,
			Amount: Amount{
				Value: charge.Amount.String(),
			},
		}
		if charge.Code != "" {
			allowanceCharge.AllowanceChargeReasonCode = charge.Code
		}
		if charge.Reason != "" {
			allowanceCharge.AllowanceChargeReason = charge.Reason
		}
		if charge.Percent != nil {
			allowanceCharge.MultiplierFactorNumeric = charge.Percent.String()
		}
		allowanceCharges = append(allowanceCharges, allowanceCharge)
	}
	for _, discount := range discounts {
		allowanceCharge := &AllowanceCharge{
			ChargeIndicator: false,
			Amount: Amount{
				Value: discount.Amount.String(),
			},
		}
		if discount.Code != "" {
			allowanceCharge.AllowanceChargeReasonCode = discount.Code
		}
		if discount.Reason != "" {
			allowanceCharge.AllowanceChargeReason = discount.Reason
		}
		if discount.Percent != nil {
			allowanceCharge.MultiplierFactorNumeric = discount.Percent.String()
		}
		allowanceCharges = append(allowanceCharges, allowanceCharge)
	}
	return allowanceCharges
}
