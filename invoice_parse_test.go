package ubl_test

import (
	"bytes"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInvoiceTypes(t *testing.T) {
	t.Run("standard invoice (380)", func(t *testing.T) {
		e := parseXMLInvoice(t, "base-example.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.Equal(t, bill.InvoiceTypeStandard, inv.Type)
		assert.Empty(t, inv.Tags)
	})

	t.Run("credit note (381)", func(t *testing.T) {
		e := parseXMLInvoice(t, "base-creditnote-correction.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.Equal(t, bill.InvoiceTypeCreditNote, inv.Type)
		assert.Empty(t, inv.Tags)
	})

	t.Run("proforma invoice (325)", func(t *testing.T) {
		e := parseXMLInvoice(t, "proforma-invoice.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.Equal(t, bill.InvoiceTypeProforma, inv.Type)
		assert.Empty(t, inv.Tags)
	})

	t.Run("self-billed invoice (389)", func(t *testing.T) {
		e := parseXMLInvoice(t, "self-billed-invoice.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.Equal(t, bill.InvoiceTypeStandard, inv.Type)
		assert.True(t, inv.HasTags(tax.TagSelfBilled), "should have self-billed tag")
	})

	t.Run("partial invoice (326)", func(t *testing.T) {
		e := parseXMLInvoice(t, "partial-invoice.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.Equal(t, bill.InvoiceTypeStandard, inv.Type)
		assert.True(t, inv.HasTags(tax.TagPartial), "should have partial tag")
	})

	t.Run("self-billed credit note (261)", func(t *testing.T) {
		e := parseXMLInvoice(t, "self-billed-creditnote.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.Equal(t, bill.InvoiceTypeCreditNote, inv.Type)
		assert.True(t, inv.HasTags(tax.TagSelfBilled), "should have self-billed tag")
	})
}

func TestParseInvoiceTags(t *testing.T) {
	t.Run("invoice with self-billed tag", func(t *testing.T) {
		e := parseXMLInvoice(t, "self-billed-invoice.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.True(t, inv.HasTags(tax.TagSelfBilled), "should have self-billed tag")
	})

	t.Run("invoice with partial tag", func(t *testing.T) {
		e := parseXMLInvoice(t, "partial-invoice.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.True(t, inv.HasTags(tax.TagPartial), "should have partial tag")
	})

	t.Run("credit note with self-billed tag", func(t *testing.T) {
		e := parseXMLInvoice(t, "self-billed-creditnote.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.True(t, inv.HasTags(tax.TagSelfBilled), "should have self-billed tag")
	})

	t.Run("standard invoice without tags", func(t *testing.T) {
		e := parseXMLInvoice(t, "base-example.xml")

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
			filename:     "base-example.xml",
			expectedType: string(bill.InvoiceTypeStandard),
			expectedTags: nil,
		},
		{
			name:         "credit note (381)",
			filename:     "base-creditnote-correction.xml",
			expectedType: string(bill.InvoiceTypeCreditNote),
			expectedTags: nil,
		},
		{
			name:         "proforma (325)",
			filename:     "proforma-invoice.xml",
			expectedType: string(bill.InvoiceTypeProforma),
			expectedTags: nil,
		},
		{
			name:         "self-billed standard (389)",
			filename:     "self-billed-invoice.xml",
			expectedType: string(bill.InvoiceTypeStandard),
			expectedTags: []string{string(tax.TagSelfBilled)},
		},
		{
			name:         "partial standard (326)",
			filename:     "partial-invoice.xml",
			expectedType: string(bill.InvoiceTypeStandard),
			expectedTags: []string{string(tax.TagPartial)},
		},
		{
			name:         "self-billed credit note (261)",
			filename:     "self-billed-creditnote.xml",
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

// TestParseInvoiceDocumentTypeCodes covers the UNTDID 1001 codes that the
// EN 16931 addon only derives from a tag: the parsed type and tags must
// regenerate the stated code, or the document type extension goes missing.
func TestParseInvoiceDocumentTypeCodes(t *testing.T) {
	const (
		invoice380 = "<cbc:InvoiceTypeCode>380</cbc:InvoiceTypeCode>"
		credit381  = "<cbc:CreditNoteTypeCode>381</cbc:CreditNoteTypeCode>"
	)

	tests := []struct {
		name string
		file string
		from string
		to   string
		code cbc.Code
		typ  cbc.Key
		tag  cbc.Key
	}{
		{"prepayment invoice", "peppol/base-example.xml", invoice380, "<cbc:InvoiceTypeCode>386</cbc:InvoiceTypeCode>", "386", bill.InvoiceTypeStandard, tax.TagPrepayment},
		{"factored invoice", "peppol/base-example.xml", invoice380, "<cbc:InvoiceTypeCode>393</cbc:InvoiceTypeCode>", "393", bill.InvoiceTypeStandard, tax.TagFactoring},
		{"factored credit note", "peppol/base-creditnote-correction.xml", credit381, "<cbc:CreditNoteTypeCode>396</cbc:CreditNoteTypeCode>", "396", bill.InvoiceTypeCreditNote, tax.TagFactoring},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := parseWithTypeCode(t, tt.file, tt.from, tt.to)

			assert.Equal(t, tt.typ, inv.Type)
			assert.True(t, inv.HasTags(tt.tag), "missing tag %s", tt.tag)
			assert.Equal(t, tt.code, inv.Tax.Ext.Get(untdid.ExtKeyDocumentType))
		})
	}
}

// An unmapped BT-3 keeps its code in the extension, so the document type is
// not lost, whether the root element is an Invoice or a CreditNote.
func TestParseInvoiceUnmappedTypeCodes(t *testing.T) {
	tests := []struct {
		name string
		file string
		from string
		to   string
		typ  cbc.Key
		code cbc.Code
	}{
		{"unmapped invoice code", "peppol/base-example.xml", "<cbc:InvoiceTypeCode>380</cbc:InvoiceTypeCode>", "<cbc:InvoiceTypeCode>471</cbc:InvoiceTypeCode>", bill.InvoiceTypeOther, "471"},
		{"unmapped credit note code", "peppol/base-creditnote-correction.xml", "<cbc:CreditNoteTypeCode>381</cbc:CreditNoteTypeCode>", "<cbc:CreditNoteTypeCode>83</cbc:CreditNoteTypeCode>", bill.InvoiceTypeOther, "83"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := parseWithTypeCode(t, tt.file, tt.from, tt.to)

			assert.Equal(t, tt.typ, inv.Type)
			assert.Empty(t, inv.GetTags())
			assert.Equal(t, tt.code, inv.Tax.Ext.Get(untdid.ExtKeyDocumentType))
		})
	}
}

// parseWithTypeCode imports a test file with its BT-3 element swapped.
func parseWithTypeCode(t *testing.T, name, from, to string) *bill.Invoice {
	t.Helper()

	raw, err := testLoadXML(name)
	require.NoError(t, err)
	data := bytes.Replace(raw, []byte(from), []byte(to), 1)
	require.NotEqual(t, string(raw), string(data), "BT-3 was not replaced")

	doc, err := ubl.Decode(data)
	require.NoError(t, err)
	env, err := ubl.Import(doc)
	require.NoError(t, err)
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)
	return inv
}
