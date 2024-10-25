package gtou

import (
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
)

const (
	StandardSalesTax  = "S"
	ZeroRatedGoodsTax = "Z"
	TaxExempt         = "E"
)

// One GOBL Release, update this to use catalogues
var paymentMeans = map[cbc.Key]string{
	pay.MeansKeyCash:           "10",
	pay.MeansKeyCheque:         "20",
	pay.MeansKeyCreditTransfer: "30",
	pay.MeansKeyCard:           "48",
	pay.MeansKeyDirectDebit:    "49",
	// pay.MeansKeyCreditTransfer.With(pay.MeansKeySEPA): "58",
	// pay.MeansKeyDirectDebit.With(pay.MeansKeySEPA):    "59",
}

func formatDate(date cal.Date) string {
	if date.IsZero() {
		return ""
	}
	t := date.Time()
	return t.Format("2006-01-02")
}

func makePeriod(period *cal.Period) Period {
	start := formatDate(period.Start)
	end := formatDate(period.End)
	return Period{
		StartDate: &start,
		EndDate:   &end,
	}
}

func findTaxCode(taxRate cbc.Key) string {
	switch taxRate {
	case tax.RateStandard:
		return StandardSalesTax
	case tax.RateZero:
		return ZeroRatedGoodsTax
	case tax.RateExempt:
		return TaxExempt
	}

	return StandardSalesTax
}

func findPaymentKey(key cbc.Key) string {
	if val, ok := paymentMeans[key]; ok {
		return val
	}
	return "1"
}
