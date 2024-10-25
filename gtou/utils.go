package gtou

import (
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
)

const (
	StandardSalesTax  = "S"
	ZeroRatedGoodsTax = "Z"
	TaxExempt         = "E"
)

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
