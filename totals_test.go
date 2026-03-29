package ubl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTotals(t *testing.T) {
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
		assert.Equal(t, "VAT", doc.TaxTotal[0].TaxSubtotal[0].TaxCategory.TaxScheme.ID)
		assert.Equal(t, "21.0", *doc.TaxTotal[0].TaxSubtotal[0].TaxCategory.Percent)
	})

	t.Run("exemption reason from legal note with reverse-charge tag", func(t *testing.T) {
		doc, err := testInvoiceFrom("peppol/peppol-reverse-charge.json")
		require.NoError(t, err)

		subtotal := doc.TaxTotal[0].TaxSubtotal[0]
		require.NotNil(t, subtotal.TaxCategory.TaxExemptionReason)
		assert.Equal(t, "Reverse Charge / Umkehr der Steuerschuld.", *subtotal.TaxCategory.TaxExemptionReason)
		require.NotNil(t, subtotal.TaxCategory.TaxExemptionReasonCode)
		assert.Equal(t, "VATEX-EU-AE", *subtotal.TaxCategory.TaxExemptionReasonCode)
	})

	t.Run("no exemption reason without reverse-charge tag", func(t *testing.T) {
		doc, err := testInvoiceFrom("peppol/peppol-1.json")
		require.NoError(t, err)

		subtotal := doc.TaxTotal[0].TaxSubtotal[0]
		assert.Nil(t, subtotal.TaxCategory.TaxExemptionReason)
	})
}
