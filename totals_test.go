package ubl_test

import (
	"strings"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTotals(t *testing.T) {
	t.Run("peppol-1-advance.json", func(t *testing.T) {
		doc := testInvoiceFrom(t, "peppol/peppol-1-advance.json")

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

	t.Run("standard_invoice_no_exemption_reason", func(t *testing.T) {
		doc := testInvoiceFrom(t, "peppol/invoice-minimal.json")

		require.Len(t, doc.TaxTotal, 1)
		require.Len(t, doc.TaxTotal[0].TaxSubtotal, 1)
		tc := doc.TaxTotal[0].TaxSubtotal[0].TaxCategory
		assert.Nil(t, tc.TaxExemptionReasonCode)
		assert.Nil(t, tc.TaxExemptionReason)
	})

	t.Run("reverse_charge_exemption_from_tax_notes", func(t *testing.T) {
		doc := testInvoiceFrom(t, "peppol/peppol-reverse-charge.json")

		require.Len(t, doc.TaxTotal, 1)
		require.Len(t, doc.TaxTotal[0].TaxSubtotal, 1)
		tc := doc.TaxTotal[0].TaxSubtotal[0].TaxCategory

		assert.Equal(t, "AE", tc.ID.Value)
		assert.Equal(t, "0", *tc.Percent)
		require.NotNil(t, tc.TaxExemptionReasonCode)
		assert.Equal(t, "VATEX-EU-AE", *tc.TaxExemptionReasonCode)
		require.NotNil(t, tc.TaxExemptionReason)
		assert.Equal(t, "Reverse Charge / Umkehr der Steuerschuld.", *tc.TaxExemptionReason)
	})
}

func TestParseTaxNotes(t *testing.T) {
	t.Run("reverse_charge", func(t *testing.T) {
		env := parseXMLInvoice(t, "peppol/nbio-stuck-ubl.xml")

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
		env := parseXMLInvoice(t, "peppol/base-example.xml")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		if inv.Tax != nil {
			assert.Empty(t, inv.Tax.Notes)
		}
	})
}

func TestTaxIncludedIndicatorEncoding(t *testing.T) {
	doc := testInvoiceFrom(t, "invoice-minimal.json")

	data, err := ubl.Encode(doc)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "TaxIncludedIndicator", "omitted by default")

	included := false
	doc.TaxTotal[0].TaxIncludedIndicator = &included
	data, err = ubl.Encode(doc)
	require.NoError(t, err)
	xml := string(data)

	indicator := strings.Index(xml, "<cbc:TaxIncludedIndicator>false</cbc:TaxIncludedIndicator>")
	require.NotEqual(t, -1, indicator)
	// UBL 2.1 orders TaxIncludedIndicator after TaxAmount and before TaxSubtotal.
	assert.Less(t, strings.Index(xml, "<cac:TaxTotal>"), indicator)
	assert.Less(t, indicator, strings.Index(xml, "<cac:TaxSubtotal>"))

	parsed, err := ubl.Decode(data)
	require.NoError(t, err)
	out, ok := parsed.(*ubl.Invoice)
	require.True(t, ok)
	require.NotNil(t, out.TaxTotal[0].TaxIncludedIndicator)
	assert.False(t, *out.TaxTotal[0].TaxIncludedIndicator)
}
