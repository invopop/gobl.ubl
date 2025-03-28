package ubl

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
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
	ID                     *string    `xml:"cbc:ID,omitempty"`
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

func (out *Invoice) addTotals(t *bill.Totals, currency string) {
	if t == nil {
		return
	}
	out.LegalMonetaryTotal = MonetaryTotal{
		LineExtensionAmount: Amount{Value: t.Sum.String(), CurrencyID: &currency},
		TaxExclusiveAmount:  Amount{Value: t.Total.String(), CurrencyID: &currency},
		TaxInclusiveAmount:  Amount{Value: t.TotalWithTax.String(), CurrencyID: &currency},
		PayableAmount:       &Amount{Value: t.Payable.String(), CurrencyID: &currency},
	}
	if t.Discount != nil {
		out.LegalMonetaryTotal.AllowanceTotalAmount = &Amount{Value: t.Discount.String(), CurrencyID: &currency}
	}
	if t.Charge != nil {
		out.LegalMonetaryTotal.ChargeTotalAmount = &Amount{Value: t.Charge.String(), CurrencyID: &currency}
	}
	if t.Rounding != nil {
		out.LegalMonetaryTotal.PayableRoundingAmount = &Amount{Value: t.Rounding.String(), CurrencyID: &currency}
	}
	if t.Advances != nil {
		out.LegalMonetaryTotal.PrepaidAmount = &Amount{Value: t.Advances.String(), CurrencyID: &currency}
	}
	out.TaxTotal = []TaxTotal{
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
				if r.Percent != nil {
					p := r.Percent.StringWithoutSymbol()
					taxCat.Percent = &p
				}
				if r.Ext != nil && r.Ext[untdid.ExtKeyTaxCategory].String() != "" {
					k := r.Ext[untdid.ExtKeyTaxCategory].String()
					taxCat.ID = &k
				}
				if cat.Code != cbc.CodeEmpty {
					taxCat.TaxScheme = &TaxScheme{ID: cat.Code.String()}
				}
				subtotal.TaxCategory = taxCat
				out.TaxTotal[0].TaxSubtotal = append(out.TaxTotal[0].TaxSubtotal, subtotal)
			}
		}
	}
}
