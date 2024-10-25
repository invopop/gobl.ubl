package gtou

import (
	"github.com/invopop/gobl/bill"
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
	return nil
}
