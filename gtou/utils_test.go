package gtou

import (
	"testing"

	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
)

func TestFormatDate(t *testing.T) {
	tests := []struct {
		input    cal.Date
		expected string
	}{
		{cal.Date{}, ""},
		{cal.MakeDate(2023, 10, 1), "2023-10-01"},
		{cal.MakeDate(2024, 2, 29), "2024-02-29"},
		{cal.MakeDate(2023, 4, 30), "2023-04-30"},
	}

	for _, test := range tests {
		result := formatDate(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestMakePeriod(t *testing.T) {
	startDate := "2023-01-01"
	endDate := "2023-12-31"
	period := &cal.Period{
		Start: cal.MakeDate(2023, 1, 1),
		End:   cal.MakeDate(2023, 12, 31),
	}

	result := makePeriod(period)
	assert.Equal(t, startDate, *result.StartDate)
	assert.Equal(t, endDate, *result.EndDate)
}

func TestFindTaxCode(t *testing.T) {
	tests := []struct {
		input    cbc.Key
		expected string
	}{
		{tax.RateStandard, standardSalesTax},
		{tax.RateZero, zeroRatedGoodsTax},
		{tax.RateExempt, taxExempt},
		{cbc.Key("unknown"), standardSalesTax},
	}

	for _, test := range tests {
		result := findTaxCode(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestFindPaymentKey(t *testing.T) {
	tests := []struct {
		input    cbc.Key
		expected string
	}{
		{pay.MeansKeyCash, "10"},
		{pay.MeansKeyCheque, "20"},
		{pay.MeansKeyCreditTransfer, "30"},
		{cbc.Key("unknown"), "1"},
	}

	for _, test := range tests {
		result := findPaymentKey(test.input)
		assert.Equal(t, test.expected, result)
	}
}
