package ubl_test

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
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

	t.Run("standard_invoice_no_exemption_reason", func(t *testing.T) {
		doc, err := testInvoiceFrom("peppol/invoice-minimal.json")
		require.NoError(t, err)

		require.Len(t, doc.TaxTotal, 1)
		require.Len(t, doc.TaxTotal[0].TaxSubtotal, 1)
		tc := doc.TaxTotal[0].TaxSubtotal[0].TaxCategory
		assert.Nil(t, tc.TaxExemptionReasonCode)
		assert.Nil(t, tc.TaxExemptionReason)
	})

	t.Run("reverse_charge_exemption_from_tax_notes", func(t *testing.T) {
		doc, err := testInvoiceFrom("peppol/peppol-reverse-charge.json")
		require.NoError(t, err)

		require.Len(t, doc.TaxTotal, 1)
		require.Len(t, doc.TaxTotal[0].TaxSubtotal, 1)
		tc := doc.TaxTotal[0].TaxSubtotal[0].TaxCategory

		assert.Equal(t, "AE", *tc.ID)
		assert.Equal(t, "0", *tc.Percent)
		require.NotNil(t, tc.TaxExemptionReasonCode)
		assert.Equal(t, "VATEX-EU-AE", *tc.TaxExemptionReasonCode)
		require.NotNil(t, tc.TaxExemptionReason)
		assert.Equal(t, "Reverse Charge / Umkehr der Steuerschuld.", *tc.TaxExemptionReason)
	})
}

func TestParseTaxNotes(t *testing.T) {
	t.Run("reverse_charge", func(t *testing.T) {
		env, err := testParseInvoice("peppol/nbio-stuck-ubl.xml")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.NotNil(t, inv.Tax)
		require.Len(t, inv.Tax.Notes, 1)

		note := inv.Tax.Notes[0]
		assert.Equal(t, cbc.Code("VAT"), note.Category)
		assert.Equal(t, cbc.Key("reverse-charge"), note.Key)
		assert.Equal(t, "Reverse charge Article 20", note.Text)
		assert.Equal(t, cbc.Code("AE"), note.Ext.Get(untdid.ExtKeyTaxCategory))
	})

	t.Run("standard_no_tax_notes", func(t *testing.T) {
		env, err := testParseInvoice("peppol/base-example.xml")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		if inv.Tax != nil {
			assert.Empty(t, inv.Tax.Notes)
		}
	})
}
