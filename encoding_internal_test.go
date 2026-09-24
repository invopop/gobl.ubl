package ubl

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A sender whose own encoding broke upstream ships U+FFFD in place of the
// character it lost. GOBL's canonical JSON refuses any document holding one, so
// a single field would otherwise cost us the whole invoice.
func TestCleanDocumentStripsReplacementCharacters(t *testing.T) {
	t.Run("reaches a field no cleanString call covers", func(t *testing.T) {
		cost := "Offre n� 0797247"
		in := &Invoice{InvoiceLines: []InvoiceLine{{AccountingCost: &cost}}}
		cleanDocument(in)
		assert.Equal(t, "Offre n 0797247", *in.InvoiceLines[0].AccountingCost)
	})

	t.Run("leaves a document without one untouched", func(t *testing.T) {
		in := &Invoice{ID: "INV-°-1"}
		cleanDocument(in)
		assert.Equal(t, "INV-°-1", in.ID)
	})

	t.Run("survives nil branches", func(t *testing.T) {
		require.NotPanics(t, func() { cleanDocument(&Invoice{}) })
		require.NotPanics(t, func() { cleanDocument((*Invoice)(nil)) })
	})
}

// The failure was not a mangled field but a lost document: GOBL's canonical
// JSON refuses a replacement character, so building the envelope errored out
// and nothing was converted at all.
func TestParseAcceptsReplacementCharacters(t *testing.T) {
	data, err := os.ReadFile("test/data/parse/en16931/line-totals-mismatch.xml")
	require.NoError(t, err)

	broken := bytes.Replace(data,
		[]byte("<cbc:BuyerReference>"),
		[]byte("<cbc:BuyerReference>Offre n� "), 1)
	require.NotEqual(t, data, broken, "fixture should carry a buyer reference")

	doc, err := Parse(broken)
	require.NoError(t, err)
	in, ok := doc.(*Invoice)
	require.True(t, ok)

	// Convert exercises the canonical JSON that rejected the document.
	env, err := in.Convert()
	require.NoError(t, err, "a replacement character must not cost us the document")
	require.NotNil(t, env)
	assert.NotContains(t, in.BuyerReference, "�")
}
