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
		assert.Equal(t, "VAT", doc.TaxTotal[0].TaxSubtotal[0].TaxCategory.TaxScheme.ID)
		assert.Equal(t, "19", *doc.TaxTotal[0].TaxSubtotal[0].TaxCategory.Percent)

	})
}
