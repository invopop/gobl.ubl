package ubl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTotals(t *testing.T) {
	t.Run("invoice-de-de.json", func(t *testing.T) {
		doc, err := testInvoiceFrom("invoice-de-de.json")
		require.NoError(t, err)

		assert.Equal(t, "1800.00", doc.LegalMonetaryTotal.LineExtensionAmount.Value)
		assert.Equal(t, "1800.00", doc.LegalMonetaryTotal.TaxExclusiveAmount.Value)
		assert.Equal(t, "2142.00", doc.LegalMonetaryTotal.TaxInclusiveAmount.Value)
		assert.Equal(t, "2142.00", doc.LegalMonetaryTotal.PayableAmount.Value)

		assert.Equal(t, "342.00", doc.TaxTotal[0].TaxAmount.Value)
		assert.Equal(t, "VAT", doc.TaxTotal[0].TaxSubtotal[0].TaxCategory.TaxScheme.ID.Value)
		assert.Equal(t, "19", *doc.TaxTotal[0].TaxSubtotal[0].TaxCategory.Percent)

	})

	t.Run("peppol-1-advance.json", func(t *testing.T) {
		doc, err := testInvoiceFrom("peppol/peppol-1-advance.json")
		require.NoError(t, err)

		assert.Equal(t, "1620.00", doc.LegalMonetaryTotal.LineExtensionAmount.Value)
		assert.Equal(t, "1620.00", doc.LegalMonetaryTotal.TaxExclusiveAmount.Value)
		assert.Equal(t, "1960.20", doc.LegalMonetaryTotal.TaxInclusiveAmount.Value)
		assert.NotNil(t, doc.LegalMonetaryTotal.PrepaidAmount)
		assert.Equal(t, "196.02", doc.LegalMonetaryTotal.PrepaidAmount.Value)
		assert.NotNil(t, doc.LegalMonetaryTotal.PayableAmount)
		assert.Equal(t, "1764.18", doc.LegalMonetaryTotal.PayableAmount.Value)

		assert.Equal(t, "340.20", doc.TaxTotal[0].TaxAmount.Value)
		assert.Equal(t, "VAT", doc.TaxTotal[0].TaxSubtotal[0].TaxCategory.TaxScheme.ID.Value)
		assert.Equal(t, "21.0", *doc.TaxTotal[0].TaxSubtotal[0].TaxCategory.Percent)
	})
}
