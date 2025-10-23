package ubl_test

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInvoiceTypes(t *testing.T) {
	t.Run("standard invoice (380)", func(t *testing.T) {
		e, err := testParseInvoice("base-example.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.Equal(t, bill.InvoiceTypeStandard, inv.Type)
		assert.Empty(t, inv.Tags)
	})

	t.Run("credit note (381)", func(t *testing.T) {
		e, err := testParseInvoice("base-creditnote-correction.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.Equal(t, bill.InvoiceTypeCreditNote, inv.Type)
		assert.Empty(t, inv.Tags)
	})

	t.Run("proforma invoice (325)", func(t *testing.T) {
		e, err := testParseInvoice("proforma-invoice.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.Equal(t, bill.InvoiceTypeProforma, inv.Type)
		assert.Empty(t, inv.Tags)
	})

	t.Run("self-billed invoice (389)", func(t *testing.T) {
		e, err := testParseInvoice("self-billed-invoice.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.Equal(t, bill.InvoiceTypeStandard, inv.Type)
		assert.True(t, inv.HasTags(tax.TagSelfBilled), "should have self-billed tag")
	})

	t.Run("partial invoice (326)", func(t *testing.T) {
		e, err := testParseInvoice("partial-invoice.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.Equal(t, bill.InvoiceTypeStandard, inv.Type)
		assert.True(t, inv.HasTags(tax.TagPartial), "should have partial tag")
	})

	t.Run("self-billed credit note (261)", func(t *testing.T) {
		e, err := testParseInvoice("self-billed-creditnote.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.Equal(t, bill.InvoiceTypeCreditNote, inv.Type)
		assert.True(t, inv.HasTags(tax.TagSelfBilled), "should have self-billed tag")
	})
}

func TestParseInvoiceTags(t *testing.T) {
	t.Run("invoice with self-billed tag", func(t *testing.T) {
		e, err := testParseInvoice("self-billed-invoice.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.True(t, inv.HasTags(tax.TagSelfBilled), "should have self-billed tag")
	})

	t.Run("invoice with partial tag", func(t *testing.T) {
		e, err := testParseInvoice("partial-invoice.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.True(t, inv.HasTags(tax.TagPartial), "should have partial tag")
	})

	t.Run("credit note with self-billed tag", func(t *testing.T) {
		e, err := testParseInvoice("self-billed-creditnote.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.True(t, inv.HasTags(tax.TagSelfBilled), "should have self-billed tag")
	})

	t.Run("standard invoice without tags", func(t *testing.T) {
		e, err := testParseInvoice("base-example.xml")
		require.NoError(t, err)

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
			e, err := testParseInvoice(tt.filename)
			require.NoError(t, err)

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
