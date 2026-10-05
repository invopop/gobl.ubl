package ubl_test

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/note"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("invoice namespace returns *Invoice", func(t *testing.T) {
		data, err := testLoadXML("en16931/ubl-example1.xml")
		require.NoError(t, err)

		doc, err := ubl.Parse(data)
		require.NoError(t, err)

		inv, ok := doc.(*ubl.Invoice)
		require.True(t, ok, "expected *ubl.Invoice")
		assert.Equal(t, "urn:cen.eu:en16931:2017", inv.CustomizationID)
	})

	t.Run("credit note namespace returns *Invoice", func(t *testing.T) {
		data, err := testLoadXML("en16931/credit-note1.xml")
		require.NoError(t, err)

		doc, err := ubl.Parse(data)
		require.NoError(t, err)

		_, ok := doc.(*ubl.Invoice)
		require.True(t, ok, "expected *ubl.Invoice for CreditNote documents")
	})

	t.Run("unknown root namespace returns ErrUnknownDocumentType", func(t *testing.T) {
		data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<Foo xmlns="urn:example:foo"><Bar/></Foo>`)

		_, err := ubl.Parse(data)
		assert.ErrorIs(t, err, ubl.ErrUnknownDocumentType)
	})

	t.Run("empty input returns ErrUnknownDocumentType", func(t *testing.T) {
		_, err := ubl.Parse(nil)
		assert.ErrorIs(t, err, ubl.ErrUnknownDocumentType)
	})

	t.Run("malformed XML returns a parse error", func(t *testing.T) {
		_, err := ubl.Parse([]byte("<not-closed"))
		require.Error(t, err)
		assert.False(t, errors.Is(err, ubl.ErrUnknownDocumentType))
		assert.Contains(t, err.Error(), "error parsing XML")
	})
}

func TestConvertDefaultContext(t *testing.T) {
	// Calling Convert without WithContext should fall back to EN16931.
	env := loadTestEnvelope(t, "invoice-minimal.json")

	doc, err := ubl.Convert(env)
	require.NoError(t, err)

	inv, ok := doc.(*ubl.Invoice)
	require.True(t, ok)
	assert.Equal(t, ubl.ContextEN16931.CustomizationID, inv.CustomizationID)
	assert.Empty(t, inv.ProfileID)
}

func TestConvertAutomaticallyAddsRequiredAddons(t *testing.T) {

	t.Run("no-op when addon is already present", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		before := append([]cbc.Key(nil), inv.GetAddons()...)
		require.Contains(t, before, en16931.V2017)

		_, err := ubl.Convert(env, ubl.WithContext(ubl.ContextEN16931))
		require.NoError(t, err)

		assert.Equal(t, before, inv.GetAddons(),
			"addon list should be unchanged when all required addons are already set")
	})
}

func TestConvertUnsupportedDocumentType(t *testing.T) {
	// Build an envelope around a non-invoice document. Use a context with no
	// required addons so ensureAddons exits early and we reach the type switch.
	env, err := gobl.Envelop(&note.Message{Content: "hello"})
	require.NoError(t, err)

	_, err = ubl.Convert(env, ubl.WithContext(ubl.Context{}))
	assert.ErrorIs(t, err, ubl.ErrUnsupportedDocumentType)
}

func TestConvertRejectsUnsupportedDocument(t *testing.T) {
	// A document type with no UBL mapping is rejected as unsupported.
	env, err := gobl.Envelop(&note.Message{Content: "hello"})
	require.NoError(t, err)

	_, err = ubl.Convert(env, ubl.WithContext(ubl.ContextEN16931))
	require.Error(t, err)
	assert.ErrorIs(t, err, ubl.ErrUnsupportedDocumentType)
}

func TestBytes(t *testing.T) {
	env := loadTestEnvelope(t, "invoice-minimal.json")

	doc, err := ubl.ConvertInvoice(env)
	require.NoError(t, err)

	out, err := ubl.Bytes(doc)
	require.NoError(t, err)

	s := string(out)
	assert.True(t, strings.HasPrefix(s, `<?xml version="1.0" encoding="UTF-8"?>`),
		"output should start with the standard XML header")
	assert.Contains(t, s, "<Invoice")
}

func TestBytesCompact(t *testing.T) {
	env := loadTestEnvelope(t, "invoice-minimal.json")

	doc, err := ubl.ConvertInvoice(env)
	require.NoError(t, err)

	compact, err := ubl.BytesCompact(doc)
	require.NoError(t, err)

	s := string(compact)
	assert.True(t, strings.HasPrefix(s, `<?xml version="1.0" encoding="UTF-8"?>`),
		"output should start with the standard XML header")
	assert.Contains(t, s, "<Invoice")

	// Same document, without the indentation Bytes adds.
	indented, err := ubl.Bytes(doc)
	require.NoError(t, err)
	assert.NotContains(t, s, "\n  <cbc:ID>", "compact output should not be indented")
	assert.Less(t, len(compact), len(indented), "compact output should be smaller")

	// Both forms carry the same content once whitespace between tags is gone.
	strip := func(b []byte) string {
		return regexp.MustCompile(`>\s+<`).ReplaceAllString(string(b), "><")
	}
	assert.Equal(t, strip(indented), strip(compact))
}

func TestBytesRejectsUnmarshalableDocument(t *testing.T) {
	// Channels cannot be marshalled, so both forms must surface the error
	// rather than return a half-written document.
	_, err := ubl.Bytes(make(chan int))
	assert.Error(t, err)

	_, err = ubl.BytesCompact(make(chan int))
	assert.Error(t, err)
}
