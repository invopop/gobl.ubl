package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaxExchangeRate(t *testing.T) {
	const fixture = "france-extended/invoice-tax-exchange-rate.json"

	t.Run("french extended maps the VAT accounting currency exchange rate", func(t *testing.T) {
		doc, err := testInvoiceFromContext(fixture, ubl.ContextPeppolFranceExtended)
		require.NoError(t, err)

		require.NotNil(t, doc.TaxExchangeRate)
		require.NotNil(t, doc.TaxExchangeRate.SourceCurrencyCode)
		assert.Equal(t, "USD", *doc.TaxExchangeRate.SourceCurrencyCode)
		require.NotNil(t, doc.TaxExchangeRate.TargetCurrencyCode)
		assert.Equal(t, "EUR", *doc.TaxExchangeRate.TargetCurrencyCode)
		require.NotNil(t, doc.TaxExchangeRate.CalculationRate)
		assert.Equal(t, "0.92", *doc.TaxExchangeRate.CalculationRate)
		require.NotNil(t, doc.TaxExchangeRate.Date)
		assert.Equal(t, "2024-06-13", *doc.TaxExchangeRate.Date)
	})

	t.Run("exchange rate is ignored outside the french extended context", func(t *testing.T) {
		doc, err := testInvoiceFromContext(fixture, ubl.ContextPeppol)
		require.NoError(t, err)

		assert.Nil(t, doc.TaxExchangeRate)
	})

	t.Run("parse restores the exchange rate", func(t *testing.T) {
		doc, err := testInvoiceFromContext(fixture, ubl.ContextPeppolFranceExtended)
		require.NoError(t, err)
		data, err := ubl.Bytes(doc)
		require.NoError(t, err)

		parsed, err := ubl.Parse(data)
		require.NoError(t, err)
		in, ok := parsed.(*ubl.Invoice)
		require.True(t, ok)
		env, err := in.Convert()
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		require.Len(t, inv.ExchangeRates, 1)
		rate := inv.ExchangeRates[0]
		assert.Equal(t, "USD", rate.From.String())
		assert.Equal(t, "EUR", rate.To.String())
		assert.Equal(t, "0.92", rate.Amount.String())
		require.NotNil(t, rate.At)
		assert.Equal(t, "2024-06-13T00:00:00", rate.At.String())
	})

	t.Run("parse honors cac:TaxExchangeRate regardless of context", func(t *testing.T) {
		doc, err := testInvoiceFromContext(fixture, ubl.ContextPeppolFranceExtended)
		require.NoError(t, err)
		data, err := ubl.Bytes(doc)
		require.NoError(t, err)

		parsed, err := ubl.Parse(data)
		require.NoError(t, err)
		in, ok := parsed.(*ubl.Invoice)
		require.True(t, ok)
		// Force a non-French-extended context on parse; the element should
		// still be honored since the source document carries it explicitly.
		env, err := in.Convert(ubl.WithContext(ubl.ContextPeppol))
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		require.Len(t, inv.ExchangeRates, 1)
		rate := inv.ExchangeRates[0]
		assert.Equal(t, "USD", rate.From.String())
		assert.Equal(t, "EUR", rate.To.String())
		assert.Equal(t, "0.92", rate.Amount.String())
	})

	t.Run("mismatched cac:TaxExchangeRate currencies are ignored", func(t *testing.T) {
		doc, err := testInvoiceFromContext(fixture, ubl.ContextPeppolFranceExtended)
		require.NoError(t, err)
		data, err := ubl.Bytes(doc)
		require.NoError(t, err)

		parsed, err := ubl.Parse(data)
		require.NoError(t, err)
		in, ok := parsed.(*ubl.Invoice)
		require.True(t, ok)

		// Corrupt the source currency so it no longer matches
		// DocumentCurrencyCode; it should not be trusted.
		mismatched := "GBP"
		in.TaxExchangeRate.SourceCurrencyCode = &mismatched

		env, err := in.Convert()
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		require.Len(t, inv.ExchangeRates, 1)
		// Falls back to the TaxTotal-derived heuristic instead of the
		// inconsistent element.
		assert.Equal(t, "USD", inv.ExchangeRates[0].From.String())
		assert.Equal(t, "EUR", inv.ExchangeRates[0].To.String())
	})
}
