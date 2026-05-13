package ubl

import (
	"strconv"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/currency"
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
	DocumentReference   *LineDocReference   `xml:"cac:DocumentReference,omitempty"`
	AllowanceCharge     []*AllowanceCharge  `xml:"cac:AllowanceCharge"`
	TaxTotal            []TaxTotal          `xml:"cac:TaxTotal,omitempty"`
	Item                *Item               `xml:"cac:Item"`
	Price               *Price              `xml:"cac:Price"`
}

// LineDocReference defines a document reference at line level (BT-128)
type LineDocReference struct {
	ID               IDType  `xml:"cbc:ID"`
	DocumentTypeCode *string `xml:"cbc:DocumentTypeCode,omitempty"`
}

func (ui *Invoice) addLines(inv *bill.Invoice, context Context) { //nolint:gocyclo
	if len(inv.Lines) == 0 {
		return
	}

	var lines []InvoiceLine
	invoiceType := ui.getInvoiceTypeBasedOnXMLName()

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
		if invoiceType.In(bill.InvoiceTypeCreditNote) {
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

		// BT-128: Invoice line object identifier
		if l.Identifier != nil {
			typeCode := "130"
			ref := &LineDocReference{
				ID:               IDType{Value: l.Identifier.Code.String()},
				DocumentTypeCode: &typeCode,
			}
			if l.Identifier.Ext.Has(untdid.ExtKeyReference) {
				s := l.Identifier.Ext.Get(untdid.ExtKeyReference).String()
				ref.ID.SchemeID = &s
			}
			invLine.DocumentReference = ref
		}

		if l.Period != nil {
			invLine.InvoicePeriod = &Period{
				StartDate: formatDate(l.Period.Start),
				EndDate:   formatDate(l.Period.End),
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

		// Zatca specific KSA-11
		if context.Is(ContextZATCA) && l.Total != nil && len(l.Taxes) > 0 && l.Taxes[0].Percent != nil {
			taxAmount := l.Taxes[0].Percent.Of(*l.Total)
			roundingAmount := l.Total.Add(taxAmount)
			invLine.TaxTotal = []TaxTotal{
				{
					TaxAmount:      Amount{Value: taxAmount.String(), CurrencyID: &ccy},
					RoundingAmount: &Amount{Value: roundingAmount.String(), CurrencyID: &ccy},
				},
			}
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
						ID: l.Taxes[0].Category.String(),
					},
				}

				if s := l.Taxes[0].Ext.Get(untdid.ExtKeyTaxCategory).String(); s != "" {
					rate := s
					it.ClassifiedTaxCategory.ID = &rate
				}

				// Set percent: required unless category is "O" (outside scope)
				if l.Taxes[0].Percent != nil {
					p := l.Taxes[0].Percent.StringWithoutSymbol()
					it.ClassifiedTaxCategory.Percent = &p
				} else if it.ClassifiedTaxCategory.ID == nil || *it.ClassifiedTaxCategory.ID != "O" {
					// Default to 0% when not outside scope
					p := "0"
					it.ClassifiedTaxCategory.Percent = &p
				}

				if s := l.Taxes[0].Ext.Get(untdid.ExtKeyTaxCategory).String(); s != "" {
					rate := s
					it.ClassifiedTaxCategory.ID = &rate
				}
			}

			if len(l.Item.Identities) > 0 {
				for _, id := range l.Item.Identities {
					// BT-158/159: Item classification (Label holds the listID)
					if id.Label != "" && !id.Ext.Has(iso.ExtKeySchemeID) {
						listID := id.Label
						if it.CommodityClassification == nil {
							it.CommodityClassification = &[]CommodityClassification{}
						}
						*it.CommodityClassification = append(*it.CommodityClassification, CommodityClassification{
							ItemClassificationCode: &IDType{
								Value:  id.Code.String(),
								ListID: &listID,
							},
						})
						continue
					}

					if it.BuyersItemIdentification != nil && it.StandardItemIdentification != nil {
						break
					}

					// Map first identity without extension to BuyersItemIdentification
					if id.Ext.Get(iso.ExtKeySchemeID).String() == "" {
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
						s := id.Ext.Get(iso.ExtKeySchemeID).String()
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

		lines = append(lines, invLine)
	}
	if invoiceType.In(bill.InvoiceTypeCreditNote) {
		ui.CreditNoteLines = lines
	} else {
		ui.InvoiceLines = lines
	}
}

// rescaleToCurrency rounds the amount to the natural precision of the given
// currency code (e.g. 2 for EUR, 0 for JPY). Falls back to the amount's
// existing precision if the currency code is unknown.
func rescaleToCurrency(a num.Amount, ccy string) string {
	if def := currency.Code(ccy).Def(); def != nil {
		return def.Rescale(a).String()
	}
	return a.String()
}

func makeLineCharges(charges []*bill.LineCharge, discounts []*bill.LineDiscount, ccy string, baseSum *num.Amount) []*AllowanceCharge {
	var allowanceCharges []*AllowanceCharge
	// BR-DEC-24 / UBL-DT-01: line allowance and charge amounts (BT-136/BT-141)
	// and their base amounts must match the currency's natural precision.
	// GOBL keeps higher precision internally — notably after RemoveIncludedTaxes
	// strips VAT from prices_include invoices — so round here at the boundary.
	var base *Amount
	if baseSum != nil {
		base = &Amount{
			Value:      rescaleToCurrency(*baseSum, ccy),
			CurrencyID: &ccy,
		}
	}
	for _, ch := range charges {
		ac := &AllowanceCharge{
			ChargeIndicator: true,
			Amount: Amount{
				Value:      rescaleToCurrency(ch.Amount, ccy),
				CurrencyID: &ccy,
			},
		}
		if s := ch.Ext.Get(untdid.ExtKeyCharge).String(); s != "" {
			e := s
			ac.AllowanceChargeReasonCode = &e
		}
		if ch.Reason != "" {
			ac.AllowanceChargeReason = &ch.Reason
		}
		if ch.Percent != nil {
			p := ch.Percent.StringWithoutSymbol()
			ac.MultiplierFactorNumeric = &p
			if base != nil {
				ac.BaseAmount = base
			}
		}
		allowanceCharges = append(allowanceCharges, ac)
	}
	for _, d := range discounts {
		ac := &AllowanceCharge{
			ChargeIndicator: false,
			Amount: Amount{
				Value:      rescaleToCurrency(d.Amount, ccy),
				CurrencyID: &ccy,
			},
		}
		if s := d.Ext.Get(untdid.ExtKeyAllowance).String(); s != "" {
			e := s
			ac.AllowanceChargeReasonCode = &e
		}
		if d.Reason != "" {
			ac.AllowanceChargeReason = &d.Reason
		}
		if d.Percent != nil {
			p := d.Percent.StringWithoutSymbol()
			ac.MultiplierFactorNumeric = &p
			if base != nil {
				ac.BaseAmount = base
			}
		}
		allowanceCharges = append(allowanceCharges, ac)
	}
	return allowanceCharges
}
