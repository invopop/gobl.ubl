package gtou

import (
	"strconv"

	"github.com/invopop/gobl.ubl/document"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/num"
)

func (c *Converter) newLines(inv *bill.Invoice) error {
	if len(inv.Lines) == 0 {
		return nil
	}

	var lines []document.InvoiceLine

	for _, l := range inv.Lines {
		ccy := l.Item.Currency.String()
		if ccy == "" {
			ccy = inv.Currency.String()
		}
		invLine := document.InvoiceLine{
			ID: strconv.Itoa(l.Index),

			LineExtensionAmount: document.Amount{
				CurrencyID: &ccy,
				Value:      l.Total.String(),
			},
		}

		if l.Quantity != (num.Amount{}) {
			invLine.InvoicedQuantity = &document.Quantity{
				Value: l.Quantity.String(),
			}
			if l.Item != nil && l.Item.Unit != "" {
				invLine.InvoicedQuantity.UnitCode = string(l.Item.Unit.UNECE())
			}
		}

		if len(l.Notes) > 0 {
			var notes []string
			for _, note := range l.Notes {
				if note.Key == "buyer-accounting-ref" {
					invLine.AccountingCost = &note.Text
				} else {
					notes = append(notes, note.Text)
				}
			}
			if len(notes) > 0 {
				invLine.Note = notes
			}
		}

		if len(l.Charges) > 0 || len(l.Discounts) > 0 {
			invLine.AllowanceCharge = makeLineCharges(l.Charges, l.Discounts, ccy)
		}

		if l.Item != nil {
			it := &document.Item{}

			if l.Item.Description != "" {
				d := l.Item.Description
				it.Description = &d
			}

			if l.Item.Name != "" {
				it.Name = l.Item.Name
			}

			if l.Item.Origin != "" {
				it.OriginCountry = &document.Country{
					IdentificationCode: l.Item.Origin.String(),
				}
			}

			if l.Item.Meta != nil {
				var properties []document.AdditionalItemProperty
				for key, value := range l.Item.Meta {
					properties = append(properties, document.AdditionalItemProperty{Name: key.String(), Value: value})
				}
				it.AdditionalItemProperty = &properties
			}

			if len(l.Taxes) > 0 && l.Taxes[0].Category != "" {
				it.ClassifiedTaxCategory = &document.ClassifiedTaxCategory{
					TaxScheme: &document.TaxScheme{
						ID: l.Taxes[0].Category.String(),
					},
				}
				if l.Taxes[0].Percent != nil {
					p := l.Taxes[0].Percent.StringWithoutSymbol()
					it.ClassifiedTaxCategory.Percent = &p
				}
				if l.Taxes[0].Ext != nil && l.Taxes[0].Ext[untdid.ExtKeyTaxCategory].String() != "" {
					rate := l.Taxes[0].Ext[untdid.ExtKeyTaxCategory].String()
					it.ClassifiedTaxCategory.ID = &rate
				}
			}

			invLine.Item = it

			if l.Item.Price != (num.Amount{}) {
				invLine.Price = &document.Price{
					PriceAmount: document.Amount{
						CurrencyID: &ccy,
						Value:      l.Item.Price.String(),
					},
				}
			}
		}

		lines = append(lines, invLine)
	}
	c.doc.InvoiceLine = lines
	return nil
}

func makeLineCharges(charges []*bill.LineCharge, discounts []*bill.LineDiscount, ccy string) []*document.AllowanceCharge {
	var allowanceCharges []*document.AllowanceCharge
	for _, ch := range charges {
		ac := &document.AllowanceCharge{
			ChargeIndicator: true,
			Amount: document.Amount{
				Value:      ch.Amount.String(),
				CurrencyID: &ccy,
			},
		}
		if ch.Ext != nil && ch.Ext[untdid.ExtKeyCharge].String() != "" {
			e := ch.Ext[untdid.ExtKeyCharge].String()
			ac.AllowanceChargeReasonCode = &e
		}
		if ch.Reason != "" {
			ac.AllowanceChargeReason = &ch.Reason
		}
		if ch.Percent != nil {
			p := ch.Percent.String()
			ac.MultiplierFactorNumeric = &p
		}
		allowanceCharges = append(allowanceCharges, ac)
	}
	for _, d := range discounts {
		ac := &document.AllowanceCharge{
			ChargeIndicator: false,
			Amount: document.Amount{
				Value:      d.Amount.String(),
				CurrencyID: &ccy,
			},
		}
		if d.Ext != nil && d.Ext[untdid.ExtKeyAllowance].String() != "" {
			e := d.Ext[untdid.ExtKeyAllowance].String()
			ac.AllowanceChargeReasonCode = &e
		}
		if d.Reason != "" {
			ac.AllowanceChargeReason = &d.Reason
		}
		if d.Percent != nil {
			p := d.Percent.String()
			ac.MultiplierFactorNumeric = &p
		}
		allowanceCharges = append(allowanceCharges, ac)
	}
	return allowanceCharges
}
