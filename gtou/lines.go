package gtou

import (
	"strconv"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
)

func (c *Conversor) newLines(inv *bill.Invoice) error {
	if len(inv.Lines) == 0 {
		return nil
	}

	var invoiceLines []InvoiceLine

	for _, line := range inv.Lines {
		currency := line.Item.Currency.String()
		if currency == "" {
			currency = inv.Currency.String()
		}
		invoiceLine := InvoiceLine{
			ID: strconv.Itoa(line.Index),

			LineExtensionAmount: Amount{
				CurrencyID: &currency,
				Value:      line.Total.String(),
			},
		}

		if line.Quantity != (num.Amount{}) {
			invoiceLine.InvoicedQuantity = &Quantity{
				Value: line.Quantity.String(),
			}
			if line.Item != nil && line.Item.Unit != "" {
				invoiceLine.InvoicedQuantity.UnitCode = string(line.Item.Unit.UNECE())
			}
		}

		if len(line.Notes) > 0 {
			var notes []string
			for _, note := range line.Notes {
				if note.Key == "buyer-accounting-ref" {
					invoiceLine.AccountingCost = &note.Text
				} else {
					notes = append(notes, note.Text)
				}
			}
			if len(notes) > 0 {
				invoiceLine.Note = notes
			}
		}

		if len(line.Charges) > 0 || len(line.Discounts) > 0 {
			invoiceLine.AllowanceCharge = makeLineCharges(line.Charges, line.Discounts)
		}

		if line.Item != nil {
			item := &Item{}

			if line.Item.Description != "" {
				d := line.Item.Description
				item.Description = &d
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

			if len(line.Taxes) > 0 && line.Taxes[0].Category != "" {
				category := line.Taxes[0].Category.String()
				item.ClassifiedTaxCategory = &ClassifiedTaxCategory{
					TaxScheme: &TaxScheme{
						ID: &category,
					},
				}
				if line.Taxes[0].Percent != nil {
					percent := line.Taxes[0].Percent.String()
					item.ClassifiedTaxCategory.Percent = &percent
				}
				if line.Taxes[0].Rate != "" {
					rate := findTaxCode(line.Taxes[0].Rate)
					item.ClassifiedTaxCategory.ID = &rate
				}
			}

			invoiceLine.Item = item

			if line.Item.Price != (num.Amount{}) {
				invoiceLine.Price = &Price{
					PriceAmount: Amount{
						CurrencyID: &currency,
						Value:      line.Item.Price.String(),
					},
				}
			}
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
			allowanceCharge.AllowanceChargeReasonCode = &charge.Code
		}
		if charge.Reason != "" {
			allowanceCharge.AllowanceChargeReason = &charge.Reason
		}
		if charge.Percent != nil {
			percent := charge.Percent.String()
			allowanceCharge.MultiplierFactorNumeric = &percent
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
			allowanceCharge.AllowanceChargeReasonCode = &discount.Code
		}
		if discount.Reason != "" {
			allowanceCharge.AllowanceChargeReason = &discount.Reason
		}
		if discount.Percent != nil {
			percent := discount.Percent.String()
			allowanceCharge.MultiplierFactorNumeric = &percent
		}
		allowanceCharges = append(allowanceCharges, allowanceCharge)
	}
	return allowanceCharges
}
