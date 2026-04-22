package ubl_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/addons/fr/ctc"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/note"
	"github.com/invopop/gobl/rules"
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
	t.Run("injects missing addon from context", func(t *testing.T) {
		// Load a France CTC-shaped invoice, strip the ctc addon, and verify
		// that Convert injects it back in before producing the UBL document.
		env := loadTestEnvelope(t, "france-cius/invoice-fr-cius.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Drop the ctc addon; keep the en16931 one. SetAddons replaces the list.
		inv.SetAddons(en16931.V2017)
		require.NotContains(t, inv.GetAddons(), ctc.Flow2V1,
			"precondition: ctc addon must be absent before Convert runs")

		_, err := ubl.Convert(env, ubl.WithContext(ubl.ContextPeppolFranceCIUS))
		require.NoError(t, err)

		// After Convert the addon should have been appended in-place.
		assert.Contains(t, inv.GetAddons(), ctc.Flow2V1)
		// And the pre-existing addon must be preserved.
		assert.Contains(t, inv.GetAddons(), en16931.V2017)
	})

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

func TestConvertSurfacesValidationFaultsAfterAutoAddon(t *testing.T) {
	// When Convert auto-injects a stricter addon, the resulting validation
	// failure must be surfaced as a *gobl.Error whose cause is rules.Faults,
	// so consumers can render the []*rules.Fault list (code, paths, message)
	// instead of a flattened string.

	// Minimal DE invoice doesn't satisfy the France CTC rule set. Convert
	// with the France CIUS context to force ensureAddons to add ctc.Flow2V1
	// and then fail validation.
	env := loadTestEnvelope(t, "invoice-minimal.json")

	_, err := ubl.Convert(env, ubl.WithContext(ubl.ContextPeppolFranceCIUS))
	require.Error(t, err)

	// Must be the GOBL validation error — not wrapped in anything ubl-specific.
	assert.ErrorIs(t, err, gobl.ErrValidation)

	var ge *gobl.Error
	require.ErrorAs(t, err, &ge, "error must be a *gobl.Error so faults survive")

	faults := ge.Faults()
	require.NotNil(t, faults, "cause must be rules.Faults, not a plain error")
	require.Greater(t, faults.Len(), 0)

	// Faults().List() returns []*rules.Fault — each fault keeps its
	// structured code, paths, and message so it can be rendered by a client.
	list := faults.List()
	assert.IsType(t, []*rules.Fault{}, list)
	require.NotEmpty(t, list)

	first := list[0]
	assert.NotEmpty(t, first.Code(), "fault must carry a rule code")
	assert.NotEmpty(t, first.Message(), "fault must carry a message")
	assert.NotEmpty(t, first.Paths(), "fault must carry at least one JSON path")

	// The France CTC addon's "billing mode extension is required" rule
	// must be among the reported faults.
	assert.True(t, faults.HasCode("GOBL-FR-CTC-FLOW2-BILL-INVOICE-08"),
		"expected billing-mode-required fault; got: %s", err)
}

func TestConvertUnsupportedDocumentType(t *testing.T) {
	// Build an envelope around a non-invoice document. Use a context with no
	// required addons so ensureAddons exits early and we reach the type switch.
	env, err := gobl.Envelop(&note.Message{Content: "hello"})
	require.NoError(t, err)

	_, err = ubl.Convert(env, ubl.WithContext(ubl.Context{}))
	assert.ErrorIs(t, err, ubl.ErrUnsupportedDocumentType)
}

func TestConvertRejectsNonInvoiceWhenAddonsRequired(t *testing.T) {
	// When the context declares required addons, ensureAddons must
	// fail fast on a non-invoice payload rather than reaching the type switch.
	env, err := gobl.Envelop(&note.Message{Content: "hello"})
	require.NoError(t, err)

	_, err = ubl.Convert(env, ubl.WithContext(ubl.ContextEN16931))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected bill.Invoice")
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
