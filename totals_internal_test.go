package ubl

import (
	"testing"

	"github.com/invopop/gobl/currency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptrAmount(v string) *Amount { return &Amount{Value: v} }

func TestGoblExchangeRates(t *testing.T) {
	t.Run("OIOUBL single TaxTotal with TransactionCurrencyTaxAmount", func(t *testing.T) {
		totals := []TaxTotal{
			{
				TaxAmount: Amount{Value: "342.00"},
				TaxSubtotal: []TaxSubtotal{
					{
						TaxAmount:                    Amount{Value: "342.00"},
						TransactionCurrencyTaxAmount: ptrAmount("2551.32"),
					},
				},
			},
		}
		rates := goblExchangeRates(currency.EUR, currency.DKK, totals)
		require.Len(t, rates, 1)
		assert.Equal(t, currency.EUR, rates[0].From)
		assert.Equal(t, currency.DKK, rates[0].To)
		assert.Equal(t, "7.46", rates[0].Amount.Rescale(2).String())
	})

	t.Run("OIOUBL multiple subtotals sum the tax-currency amounts", func(t *testing.T) {
		totals := []TaxTotal{
			{
				TaxAmount: Amount{Value: "200.00"},
				TaxSubtotal: []TaxSubtotal{
					{TaxAmount: Amount{Value: "150.00"}, TransactionCurrencyTaxAmount: ptrAmount("1119.00")},
					{TaxAmount: Amount{Value: "50.00"}, TransactionCurrencyTaxAmount: ptrAmount("373.00")},
				},
			},
		}
		rates := goblExchangeRates(currency.EUR, currency.DKK, totals)
		require.Len(t, rates, 1)
		assert.Equal(t, "7.46", rates[0].Amount.Rescale(2).String())
	})

	t.Run("EN16931/Peppol second TaxTotal still works", func(t *testing.T) {
		totals := []TaxTotal{
			{TaxAmount: Amount{Value: "342.00"}},
			{TaxAmount: Amount{Value: "2551.32"}},
		}
		rates := goblExchangeRates(currency.EUR, currency.DKK, totals)
		require.Len(t, rates, 1)
		assert.Equal(t, "7.46", rates[0].Amount.Rescale(2).String())
	})

	t.Run("single TaxTotal without companion amount yields no rate", func(t *testing.T) {
		totals := []TaxTotal{{TaxAmount: Amount{Value: "342.00"}}}
		assert.Nil(t, goblExchangeRates(currency.EUR, currency.DKK, totals))
	})

	t.Run("empty input yields no rate", func(t *testing.T) {
		assert.Nil(t, goblExchangeRates(currency.EUR, currency.DKK, nil))
	})
}
