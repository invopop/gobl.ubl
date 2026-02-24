package ubl

import (
	"strconv"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/num"
)

// InvoiceLine represents a line item in an invoice and credit note
type InvoiceLine struct {
	ID                  string              `xml:"cbc:ID"`
	Note                []string            `xml:"cbc:Note"`
	InvoicedQuantity    *Quantity           `xml:"cbc:InvoicedQuantity,omitempty"` // or CreditNoteQuantity
	CreditedQuantity    *Quantity           `xml:"cbc:CreditedQuantity,omitempty"`
	LineExtensionAmount Amount              `xml:"cbc:LineExtensionAmount"`
	AccountingCost      *string             `xml:"cbc:AccountingCost"`
	InvoicePeriod       *Period             `xml:"cac:InvoicePeriod"`
	OrderLineReference  *OrderLineReference `xml:"cac:OrderLineReference"`
	AllowanceCharge     []*AllowanceCharge  `xml:"cac:AllowanceCharge"`
	TaxTotal            []TaxTotal          `xml:"cac:TaxTotal,omitempty"`
	Item                *Item               `xml:"cac:Item"`
	Price               *Price              `xml:"cac:Price"`
}

func (ui *Invoice) addLines(inv *bill.Invoice, o *options) { //nolint:gocyclo
	if len(inv.Lines) == 0 {
		return
	}

	var lines []InvoiceLine

	for _, l := range inv.Lines {
		ccy := l.Item.Currency.String()
		if ccy == "" {
			ccy = inv.Currency.String()
		}
		invLine := InvoiceLine{
			ID: strconv.Itoa(l.Index),

			LineExtensionAmount: Amount{
				CurrencyID: &ccy,
				Value:      l.Total.String(),
			},
		}

		// Always set quantity (mandatory field)
		iq := &Quantity{
			Value: l.Quantity.String(),
		}
		if l.Item != nil && l.Item.Unit != "" {
			iq.UnitCode = string(l.Item.Unit.UNECE())
		}
		if inv.Type.In(bill.InvoiceTypeCreditNote) {
			invLine.CreditedQuantity = iq
		} else {
			invLine.InvoicedQuantity = iq
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

		if l.Order != "" {
			invLine.OrderLineReference = &OrderLineReference{
				LineID: l.Order.String(),
			}
		}

		if len(l.Charges) > 0 || len(l.Discounts) > 0 {
			invLine.AllowanceCharge = makeLineCharges(l.Charges, l.Discounts, ccy, l.Sum)
		}

		if l.Item != nil {
			it := &Item{}

			if l.Item.Description != "" {
				d := l.Item.Description
				it.Description = &d
			}

			if l.Item.Name != "" {
				it.Name = l.Item.Name
			}

			if l.Item.Origin != "" {
				it.OriginCountry = &Country{
					IdentificationCode: l.Item.Origin.String(),
				}
			}

			if l.Item.Meta != nil {
				var properties []AdditionalItemProperty
				for key, value := range l.Item.Meta {
					properties = append(properties, AdditionalItemProperty{Name: key.String(), Value: value})
				}
				it.AdditionalItemProperty = &properties
			}

			if len(l.Taxes) > 0 && l.Taxes[0].Category != "" {
				it.ClassifiedTaxCategory = &ClassifiedTaxCategory{
					TaxScheme: &TaxScheme{
						ID: IDType{Value: l.Taxes[0].Category.String()},
					},
				}

				if l.Taxes[0].Ext != nil && l.Taxes[0].Ext[untdid.ExtKeyTaxCategory].String() != "" {
					rate := l.Taxes[0].Ext[untdid.ExtKeyTaxCategory].String()
					it.ClassifiedTaxCategory.ID = &IDType{Value: rate}
				}

				// Set percent: required unless category is "O" (outside scope)
				if l.Taxes[0].Percent != nil {
					p := l.Taxes[0].Percent.StringWithoutSymbol()
					it.ClassifiedTaxCategory.Percent = &p
				} else if it.ClassifiedTaxCategory.ID == nil || it.ClassifiedTaxCategory.ID.Value != "O" {
					// Default to 0% when not outside scope
					p := "0"
					it.ClassifiedTaxCategory.Percent = &p
				}

				if l.Taxes[0].Ext != nil && l.Taxes[0].Ext[untdid.ExtKeyTaxCategory].String() != "" {
					rate := l.Taxes[0].Ext[untdid.ExtKeyTaxCategory].String()
					it.ClassifiedTaxCategory.ID = &IDType{Value: rate}
				}
			}

			if len(l.Item.Identities) > 0 {
				for _, id := range l.Item.Identities {
					if it.BuyersItemIdentification != nil && it.StandardItemIdentification != nil {
						break
					}

					// Map first identity without extension to BuyersItemIdentification
					if id.Ext == nil || id.Ext[iso.ExtKeySchemeID].String() == "" {
						if it.BuyersItemIdentification == nil {
							it.BuyersItemIdentification = &ItemIdentification{
								ID: &IDType{
									Value: id.Code.String(),
								},
							}
						}
						continue
					}

					// Map first identity with extension to StandardItemIdentification
					if it.StandardItemIdentification == nil {
						s := id.Ext[iso.ExtKeySchemeID].String()
						it.StandardItemIdentification = &ItemIdentification{
							ID: &IDType{
								SchemeID: &s,
								Value:    id.Code.String(),
							},
						}
					}
				}
			}

			invLine.Item = it

			if l.Item.Price != nil {
				invLine.Price = &Price{
					PriceAmount: Amount{
						CurrencyID: &ccy,
						Value:      l.Item.Price.String(),
					},
				}
			}

			if l.Item.Ref != "" {
				invLine.Item.SellersItemIdentification = &ItemIdentification{
					ID: &IDType{
						Value: l.Item.Ref.String(),
					},
				}
			}
		}

		if o.context.IsOIOUBL() {
			invLine.TaxTotal = makeLineTaxTotals(l, ccy)
		}

		lines = append(lines, invLine)
	}
	if inv.Type.In(bill.InvoiceTypeCreditNote) {
		ui.CreditNoteLines = lines
	} else {
		ui.InvoiceLines = lines
	}
}

func makeLineTaxTotals(line *bill.Line, ccy string) []TaxTotal {
	if line == nil || len(line.Taxes) == 0 {
		return nil
	}

	var taxable num.Amount
	switch {
	case line.Total != nil:
		taxable = *line.Total
	case line.Sum != nil:
		taxable = *line.Sum
	default:
		return nil
	}

	taxTotal := TaxTotal{
		TaxAmount: Amount{Value: "0", CurrencyID: &ccy},
	}
	totalAmount := num.MakeAmount(0, taxable.Exp())

	for _, tax := range line.Taxes {
		subtotal := TaxSubtotal{
			TaxableAmount: Amount{Value: taxable.String(), CurrencyID: &ccy},
		}
		taxCat := TaxCategory{}

		if tax.Ext != nil && tax.Ext[untdid.ExtKeyTaxCategory].String() != "" {
			k := tax.Ext[untdid.ExtKeyTaxCategory].String()
			taxCat.ID = &IDType{Value: k}
		}

		if tax.Percent != nil {
			p := tax.Percent.StringWithoutSymbol()
			taxCat.Percent = &p
			amount := tax.Percent.Of(taxable).Rescale(taxable.Exp())
			subtotal.TaxAmount = Amount{Value: amount.String(), CurrencyID: &ccy}
			totalAmount = totalAmount.Add(amount)
		} else {
			subtotal.TaxAmount = Amount{Value: "0", CurrencyID: &ccy}
		}

		if tax.Category != "" {
			taxCat.TaxScheme = &TaxScheme{ID: IDType{Value: tax.Category.String()}}
		}
		subtotal.TaxCategory = taxCat
		taxTotal.TaxSubtotal = append(taxTotal.TaxSubtotal, subtotal)
	}

	if totalAmount.IsZero() {
		return nil
	}
	taxTotal.TaxAmount = Amount{Value: totalAmount.String(), CurrencyID: &ccy}

	return []TaxTotal{taxTotal}
}

func makeLineCharges(charges []*bill.LineCharge, discounts []*bill.LineDiscount, ccy string, baseSum *num.Amount) []*AllowanceCharge {
	var allowanceCharges []*AllowanceCharge
	for _, ch := range charges {
		ac := &AllowanceCharge{
			ChargeIndicator: true,
			Amount: Amount{
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
			p := ch.Percent.StringWithoutSymbol()
			ac.MultiplierFactorNumeric = &p
			// Add BaseAmount when percentage is provided
			if baseSum != nil {
				ac.BaseAmount = &Amount{
					Value:      baseSum.String(),
					CurrencyID: &ccy,
				}
			}
		}
		allowanceCharges = append(allowanceCharges, ac)
	}
	for _, d := range discounts {
		ac := &AllowanceCharge{
			ChargeIndicator: false,
			Amount: Amount{
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
			p := d.Percent.StringWithoutSymbol()
			ac.MultiplierFactorNumeric = &p
			// Add BaseAmount when percentage is provided
			if baseSum != nil {
				ac.BaseAmount = &Amount{
					Value:      baseSum.String(),
					CurrencyID: &ccy,
				}
			}
		}
		allowanceCharges = append(allowanceCharges, ac)
	}
	return allowanceCharges
}
