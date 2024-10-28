package gtou

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
)

func (c *Conversor) newTotals(totals *bill.Totals, currency string) error {
	if totals == nil {
		return nil
	}
	c.doc.LegalMonetaryTotal = MonetaryTotal{
		LineExtensionAmount: Amount{Value: totals.Sum.String(), CurrencyID: &currency},
		TaxExclusiveAmount:  Amount{Value: totals.Total.String(), CurrencyID: &currency},
		TaxInclusiveAmount:  Amount{Value: totals.TotalWithTax.String(), CurrencyID: &currency},
		PayableAmount:       &Amount{Value: totals.Payable.String(), CurrencyID: &currency},
	}
	if totals.Discount != nil {
		c.doc.LegalMonetaryTotal.AllowanceTotalAmount = &Amount{Value: totals.Discount.String(), CurrencyID: &currency}
	}
	if totals.Charge != nil {
		c.doc.LegalMonetaryTotal.ChargeTotalAmount = &Amount{Value: totals.Charge.String(), CurrencyID: &currency}
	}
	if totals.Rounding != nil {
		c.doc.LegalMonetaryTotal.PayableRoundingAmount = &Amount{Value: totals.Rounding.String(), CurrencyID: &currency}
	}
	if totals.Advances != nil {
		c.doc.LegalMonetaryTotal.PrepaidAmount = &Amount{Value: totals.Advances.String(), CurrencyID: &currency}
	}
	c.doc.TaxTotal = []TaxTotal{
		{
			TaxAmount: Amount{Value: totals.Tax.String(), CurrencyID: &currency},
		},
	}
	if totals.Taxes != nil && len(totals.Taxes.Categories) > 0 {
		for _, cat := range totals.Taxes.Categories {
			for _, rate := range cat.Rates {
				subtotal := TaxSubtotal{
					TaxAmount: Amount{Value: rate.Amount.String(), CurrencyID: &currency},
				}
				if rate.Base != (num.Amount{}) {
					subtotal.TaxableAmount = Amount{Value: rate.Base.String(), CurrencyID: &currency}
				}
				taxCat := TaxCategory{}
				if rate.Percent != nil {
					p := rate.Percent.String()
					taxCat.Percent = &p
				}
				if rate.Key != cbc.KeyEmpty {
					k := findTaxCode(rate.Key)
					taxCat.ID = &k
				}
				if cat.Code != cbc.CodeEmpty {
					c := cat.Code.String()
					taxCat.TaxScheme = &TaxScheme{ID: &c}
				}
				subtotal.TaxCategory = taxCat
				c.doc.TaxTotal[0].TaxSubtotal = append(c.doc.TaxTotal[0].TaxSubtotal, subtotal)
			}
		}
	}
	return nil
}
