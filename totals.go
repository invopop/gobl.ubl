package ubl

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/cef"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
)

// TaxTotal represents a tax total
type TaxTotal struct {
	TaxAmount   Amount        `xml:"cbc:TaxAmount"`
	TaxSubtotal []TaxSubtotal `xml:"cac:TaxSubtotal"`
}

// TaxSubtotal represents a tax subtotal
type TaxSubtotal struct {
	TaxableAmount Amount      `xml:"cbc:TaxableAmount,omitempty"`
	TaxAmount     Amount      `xml:"cbc:TaxAmount"`
	TaxCategory   TaxCategory `xml:"cac:TaxCategory"`
}

// TaxCategory represents a tax category
type TaxCategory struct {
	ID                     *IDType    `xml:"cbc:ID,omitempty"`
	Percent                *string    `xml:"cbc:Percent,omitempty"`
	TaxExemptionReasonCode *string    `xml:"cbc:TaxExemptionReasonCode,omitempty"`
	TaxExemptionReason     *string    `xml:"cbc:TaxExemptionReason,omitempty"`
	TaxScheme              *TaxScheme `xml:"cac:TaxScheme,omitempty"`
}

// MonetaryTotal represents the monetary totals of the invoice
type MonetaryTotal struct {
	LineExtensionAmount   Amount  `xml:"cbc:LineExtensionAmount"`
	TaxExclusiveAmount    Amount  `xml:"cbc:TaxExclusiveAmount"`
	TaxInclusiveAmount    Amount  `xml:"cbc:TaxInclusiveAmount"`
	AllowanceTotalAmount  *Amount `xml:"cbc:AllowanceTotalAmount,omitempty"`
	ChargeTotalAmount     *Amount `xml:"cbc:ChargeTotalAmount,omitempty"`
	PrepaidAmount         *Amount `xml:"cbc:PrepaidAmount,omitempty"`
	PayableRoundingAmount *Amount `xml:"cbc:PayableRoundingAmount,omitempty"`
	PayableAmount         *Amount `xml:"cbc:PayableAmount,omitempty"`
}

func (ui *Invoice) addTotals(inv *bill.Invoice) {
	if inv == nil || inv.Totals == nil {
		return
	}
	t := inv.Totals

	currency := inv.Currency.String()

	ui.LegalMonetaryTotal = MonetaryTotal{
		LineExtensionAmount: Amount{Value: t.Sum.String(), CurrencyID: &currency},
		TaxExclusiveAmount:  Amount{Value: t.Total.String(), CurrencyID: &currency},
		TaxInclusiveAmount:  Amount{Value: t.TotalWithTax.String(), CurrencyID: &currency},
		PayableAmount:       &Amount{Value: t.Payable.String(), CurrencyID: &currency},
	}

	if t.Discount != nil {
		ui.LegalMonetaryTotal.AllowanceTotalAmount = &Amount{Value: t.Discount.String(), CurrencyID: &currency}
	}
	if t.Charge != nil {
		ui.LegalMonetaryTotal.ChargeTotalAmount = &Amount{Value: t.Charge.String(), CurrencyID: &currency}
	}
	if t.Rounding != nil {
		ui.LegalMonetaryTotal.PayableRoundingAmount = &Amount{Value: t.Rounding.String(), CurrencyID: &currency}
	}
	if t.Advances != nil {
		ui.LegalMonetaryTotal.PrepaidAmount = &Amount{Value: t.Advances.String(), CurrencyID: &currency}
	}
	if t.Due != nil {
		ui.LegalMonetaryTotal.PayableAmount = &Amount{Value: t.Due.String(), CurrencyID: &currency}
	}

	ui.TaxTotal = []TaxTotal{
		{
			TaxAmount: Amount{Value: t.Tax.String(), CurrencyID: &currency},
		},
	}
	if t.Taxes != nil && len(t.Taxes.Categories) > 0 {
		for _, cat := range t.Taxes.Categories {
			for _, r := range cat.Rates {
				subtotal := TaxSubtotal{
					TaxAmount: Amount{Value: r.Amount.String(), CurrencyID: &currency},
				}
				if r.Base != (num.Amount{}) {
					subtotal.TaxableAmount = Amount{Value: r.Base.String(), CurrencyID: &currency}
				}
				taxCat := TaxCategory{}

				if r.Ext != nil {
					if r.Ext[untdid.ExtKeyTaxCategory].String() != "" {
						k := r.Ext[untdid.ExtKeyTaxCategory].String()
						taxCat.ID = &IDType{Value: k}
					}
					if r.Ext[cef.ExtKeyVATEX].String() != "" {
						v := r.Ext[cef.ExtKeyVATEX].String()
						taxCat.TaxExemptionReasonCode = &v
					}
				}

				// Set percent: required unless category is "O" (outside scope)
				if r.Percent != nil {
					p := r.Percent.StringWithoutSymbol()
					taxCat.Percent = &p
				} else if taxCat.ID == nil || taxCat.ID.Value != "O" {
					// Default to 0% when not outside scope
					p := "0"
					taxCat.Percent = &p
				}

				if inv.Notes != nil {
					for _, n := range inv.Notes {
						if n.Key == org.NoteKeyLegal {
							reason := n.Text
							taxCat.TaxExemptionReason = &reason
							break
						}
					}
				}

				if cat.Code != cbc.CodeEmpty {
					taxCat.TaxScheme = &TaxScheme{ID: IDType{Value: cat.Code.String()}}
				}
				subtotal.TaxCategory = taxCat
				ui.TaxTotal[0].TaxSubtotal = append(ui.TaxTotal[0].TaxSubtotal, subtotal)
			}
		}
	}
}

// taxCategoryInfo holds tax category information from TaxTotal
type taxCategoryInfo struct {
	exemptionReasonCode string
}

// buildTaxCategoryMap builds a map of tax category information from TaxTotal
func (ui *Invoice) buildTaxCategoryMap() map[string]*taxCategoryInfo {
	categoryMap := make(map[string]*taxCategoryInfo)

	for _, taxTotal := range ui.TaxTotal {
		for _, subtotal := range taxTotal.TaxSubtotal {
			if subtotal.TaxCategory.ID != nil && subtotal.TaxCategory.TaxScheme != nil {
				key := buildTaxCategoryKey(subtotal.TaxCategory.TaxScheme.ID.Value, subtotal.TaxCategory.ID.Value)
				info := &taxCategoryInfo{}
				if subtotal.TaxCategory.TaxExemptionReasonCode != nil {
					info.exemptionReasonCode = *subtotal.TaxCategory.TaxExemptionReasonCode
				}
				categoryMap[key] = info
			}
		}
	}

	return categoryMap
}
