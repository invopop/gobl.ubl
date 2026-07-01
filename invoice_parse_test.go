package ubl_test

import (
	"bytes"
	"testing"

	oioubl "github.com/invopop/gobl.dk.oioubl/addon"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInvoiceTypes(t *testing.T) {
	t.Run("standard invoice (380)", func(t *testing.T) {
		e := parseXMLInvoice(t, "peppol/base-example.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.Equal(t, bill.InvoiceTypeStandard, inv.Type)
		assert.Empty(t, inv.Tags)
	})

	t.Run("credit note (381)", func(t *testing.T) {
		e := parseXMLInvoice(t, "peppol/base-creditnote-correction.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.Equal(t, bill.InvoiceTypeCreditNote, inv.Type)
		assert.Empty(t, inv.Tags)
	})

	t.Run("proforma invoice (325)", func(t *testing.T) {
		e := parseXMLInvoice(t, "peppol/proforma-invoice.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.Equal(t, bill.InvoiceTypeProforma, inv.Type)
		assert.Empty(t, inv.Tags)
	})

	t.Run("self-billed invoice (389)", func(t *testing.T) {
		e := parseXMLInvoice(t, "peppol/self-billed-invoice.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.Equal(t, bill.InvoiceTypeStandard, inv.Type)
		assert.True(t, inv.HasTags(tax.TagSelfBilled), "should have self-billed tag")
	})

	t.Run("partial invoice (326)", func(t *testing.T) {
		e := parseXMLInvoice(t, "peppol/partial-invoice.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.Equal(t, bill.InvoiceTypeStandard, inv.Type)
		assert.True(t, inv.HasTags(tax.TagPartial), "should have partial tag")
	})

	t.Run("self-billed credit note (261)", func(t *testing.T) {
		e := parseXMLInvoice(t, "peppol/self-billed-creditnote.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.Equal(t, bill.InvoiceTypeCreditNote, inv.Type)
		assert.True(t, inv.HasTags(tax.TagSelfBilled), "should have self-billed tag")
	})
}

func TestParseInvoiceOIOUBLNonProfile5Context(t *testing.T) {
	// A non-profile5 ProfileID (one of OIOUBL's ~40 procurement profiles) must
	// still resolve the OIOUBL context and its addon via the CustomizationID-only
	// fallback; otherwise the document parses with no addons.
	data, err := testLoadXML("oioubl21/invoice-minimal.xml")
	require.NoError(t, err)

	swapped := bytes.Replace(data,
		[]byte("urn:www.nesubl.eu:profiles:profile5:ver2.0"),
		[]byte("Procurement-BilSim-1.0"), 1)
	require.NotEqual(t, data, swapped, "fixture should carry the profile5 ProfileID")

	doc, err := ubl.Parse(swapped)
	require.NoError(t, err)
	inv, ok := doc.(*ubl.Invoice)
	require.True(t, ok)

	env, err := inv.Convert()
	require.NoError(t, err)
	out, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)
	assert.Contains(t, out.Addons.List, oioubl.V2,
		"OIOUBL context must resolve despite the non-profile5 ProfileID")
}

func TestParseInvoiceTags(t *testing.T) {
	t.Run("invoice with self-billed tag", func(t *testing.T) {
		e := parseXMLInvoice(t, "peppol/self-billed-invoice.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.True(t, inv.HasTags(tax.TagSelfBilled), "should have self-billed tag")
	})

	t.Run("invoice with partial tag", func(t *testing.T) {
		e := parseXMLInvoice(t, "peppol/partial-invoice.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.True(t, inv.HasTags(tax.TagPartial), "should have partial tag")
	})

	t.Run("credit note with self-billed tag", func(t *testing.T) {
		e := parseXMLInvoice(t, "peppol/self-billed-creditnote.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.True(t, inv.HasTags(tax.TagSelfBilled), "should have self-billed tag")
	})

	t.Run("standard invoice without tags", func(t *testing.T) {
		e := parseXMLInvoice(t, "peppol/base-example.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.False(t, inv.HasTags(tax.TagSelfBilled), "standard invoice should not have self-billed tag")
		assert.False(t, inv.HasTags(tax.TagPartial), "standard invoice should not have partial tag")
	})
}

func TestParseInvoiceTypeAndTagCombinations(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		expectedType string
		expectedTags []string
	}{
		{
			name:         "standard invoice (380)",
			filename:     "peppol/base-example.xml",
			expectedType: string(bill.InvoiceTypeStandard),
			expectedTags: nil,
		},
		{
			name:         "credit note (381)",
			filename:     "peppol/base-creditnote-correction.xml",
			expectedType: string(bill.InvoiceTypeCreditNote),
			expectedTags: nil,
		},
		{
			name:         "proforma (325)",
			filename:     "peppol/proforma-invoice.xml",
			expectedType: string(bill.InvoiceTypeProforma),
			expectedTags: nil,
		},
		{
			name:         "self-billed standard (389)",
			filename:     "peppol/self-billed-invoice.xml",
			expectedType: string(bill.InvoiceTypeStandard),
			expectedTags: []string{string(tax.TagSelfBilled)},
		},
		{
			name:         "partial standard (326)",
			filename:     "peppol/partial-invoice.xml",
			expectedType: string(bill.InvoiceTypeStandard),
			expectedTags: []string{string(tax.TagPartial)},
		},
		{
			name:         "self-billed credit note (261)",
			filename:     "peppol/self-billed-creditnote.xml",
			expectedType: string(bill.InvoiceTypeCreditNote),
			expectedTags: []string{string(tax.TagSelfBilled)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := parseXMLInvoice(t, tt.filename)

			inv, ok := e.Extract().(*bill.Invoice)
			require.True(t, ok)

			assert.Equal(t, tt.expectedType, string(inv.Type), "invoice type mismatch")

			if tt.expectedTags == nil {
				assert.False(t, inv.HasTags(tax.TagSelfBilled), "should not have self-billed tag")
				assert.False(t, inv.HasTags(tax.TagPartial), "should not have partial tag")
			} else {
				for _, tag := range tt.expectedTags {
					assert.True(t, inv.HasTags(cbc.Key(tag)), "missing expected tag: %s", tag)
				}
			}
		})
	}
}
