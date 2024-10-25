package gtou

import (
	"github.com/invopop/gobl/bill"
)

func (c *Conversor) newTotals(totals *bill.Totals, currency string) error {
	if totals == nil {
		return nil
	}
	c.doc.LegalMonetaryTotal = MonetaryTotal{
		LineExtensionAmount:   Amount{Value: totals.Sum.String(), CurrencyID: currency},
		AllowanceTotalAmount:  Amount{Value: totals.Discount.String(), CurrencyID: currency},
		ChargeTotalAmount:     Amount{Value: totals.Charge.String(), CurrencyID: currency},
		TaxExclusiveAmount:    Amount{Value: totals.Total.String(), CurrencyID: currency},
		TaxInclusiveAmount:    Amount{Value: totals.TotalWithTax.String(), CurrencyID: currency},
		PayableRoundingAmount: Amount{Value: totals.Rounding.String(), CurrencyID: currency},
		PayableAmount:         Amount{Value: totals.Payable.String(), CurrencyID: currency},
		PrepaidAmount:         Amount{Value: totals.Advances.String(), CurrencyID: currency},
	}

	c.doc.TaxTotal = []TaxTotal{
		{
			TaxAmount: Amount{Value: totals.Tax.String(), CurrencyID: currency},
		},
	}

	return nil
}
