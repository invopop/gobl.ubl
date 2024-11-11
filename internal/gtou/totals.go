package gtou

import (
	"github.com/invopop/gobl.ubl/document"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
)

func (c *Converter) newTotals(t *bill.Totals, currency string) error {
	if t == nil {
		return nil
	}
	c.doc.LegalMonetaryTotal = document.MonetaryTotal{
		LineExtensionAmount: document.Amount{Value: t.Sum.String(), CurrencyID: &currency},
		TaxExclusiveAmount:  document.Amount{Value: t.Total.String(), CurrencyID: &currency},
		TaxInclusiveAmount:  document.Amount{Value: t.TotalWithTax.String(), CurrencyID: &currency},
		PayableAmount:       &document.Amount{Value: t.Payable.String(), CurrencyID: &currency},
	}
	if t.Discount != nil {
		c.doc.LegalMonetaryTotal.AllowanceTotalAmount = &document.Amount{Value: t.Discount.String(), CurrencyID: &currency}
	}
	if t.Charge != nil {
		c.doc.LegalMonetaryTotal.ChargeTotalAmount = &document.Amount{Value: t.Charge.String(), CurrencyID: &currency}
	}
	if t.Rounding != nil {
		c.doc.LegalMonetaryTotal.PayableRoundingAmount = &document.Amount{Value: t.Rounding.String(), CurrencyID: &currency}
	}
	if t.Advances != nil {
		c.doc.LegalMonetaryTotal.PrepaidAmount = &document.Amount{Value: t.Advances.String(), CurrencyID: &currency}
	}
	c.doc.TaxTotal = []document.TaxTotal{
		{
			TaxAmount: document.Amount{Value: t.Tax.String(), CurrencyID: &currency},
		},
	}
	if t.Taxes != nil && len(t.Taxes.Categories) > 0 {
		for _, cat := range t.Taxes.Categories {
			for _, r := range cat.Rates {
				subtotal := document.TaxSubtotal{
					TaxAmount: document.Amount{Value: r.Amount.String(), CurrencyID: &currency},
				}
				if r.Base != (num.Amount{}) {
					subtotal.TaxableAmount = document.Amount{Value: r.Base.String(), CurrencyID: &currency}
				}
				taxCat := document.TaxCategory{}
				if r.Percent != nil {
					p := r.Percent.StringWithoutSymbol()
					taxCat.Percent = &p
				}
				if r.Ext != nil && r.Ext[untdid.ExtKeyTaxCategory].String() != "" {
					k := r.Ext[untdid.ExtKeyTaxCategory].String()
					taxCat.ID = &k
				}
				if cat.Code != cbc.CodeEmpty {
					taxCat.TaxScheme = &document.TaxScheme{ID: cat.Code.String()}
				}
				subtotal.TaxCategory = taxCat
				c.doc.TaxTotal[0].TaxSubtotal = append(c.doc.TaxTotal[0].TaxSubtotal, subtotal)
			}
		}
	}
	return nil
}
