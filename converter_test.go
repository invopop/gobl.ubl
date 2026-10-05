package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConverterContexts(t *testing.T) {
	for _, k := range []cbc.Key{
		ubl.KeyUBL,
		ubl.ContextEN16931.Key,
		ubl.ContextPeppol.Key,
		ubl.ContextPeppolSelfBilled.Key,
		ubl.ContextPeppolInvoiceResponse.Key,
	} {
		ctx := convert.ContextFor(k)
		require.NotNil(t, ctx, k)
		assert.Equal(t, cbc.Key("ubl"), ctx.Syntax)
		assert.Empty(t, ctx.Countries)
	}
	assert.Len(t, convert.Contexts(), 5)

	t.Run("regional contexts have no key", func(t *testing.T) {
		for _, ctx := range []ubl.Context{
			ubl.ContextXRechnung,
			ubl.ContextPeppolFranceCIUS,
			ubl.ContextPeppolFranceExtended,
			ubl.ContextZATCA,
		} {
			assert.Empty(t, ctx.Key)
		}
	})
}

func TestConverterDetect(t *testing.T) {
	tests := []struct {
		file string
		key  cbc.Key
	}{
		{"en16931/ubl-example1.xml", ubl.ContextEN16931.Key},
		{"peppol/base-example.xml", ubl.ContextPeppol.Key},
		{"peppol/sg-invoice.xml", ubl.KeyUBL},
		{"en16931/credit-note1.xml", ubl.KeyUBL},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			data, err := testLoadXML(tt.file)
			require.NoError(t, err)
			ctx, err := convert.Detect(data)
			require.NoError(t, err)
			assert.Equal(t, tt.key, ctx.Key)
		})
	}

	t.Run("regional", func(t *testing.T) {
		for _, f := range []string{
			"france-cius/b2b-reg.xml",
			"france-extended/b2g-invoice.xml",
			"zatca/standard-invoice.xml",
		} {
			data, err := testLoadXML(f)
			require.NoError(t, err)
			_, err = convert.Detect(data)
			assert.ErrorIs(t, err, convert.ErrUnknownContext, f)
		}
	})

	t.Run("not ubl", func(t *testing.T) {
		for _, data := range []string{
			`<?xml version="1.0"?><Invoice xmlns="urn:example"/>`,
			`{"$schema":"https://gobl.org/draft-0/bill/invoice"}`,
			``,
		} {
			_, err := convert.Detect([]byte(data))
			assert.ErrorIs(t, err, convert.ErrUnknownContext, data)
		}
	})

	t.Run("limited to keys", func(t *testing.T) {
		data, err := testLoadXML("peppol/base-example.xml")
		require.NoError(t, err)
		_, err = convert.Detect(data, ubl.ContextEN16931.Key)
		assert.ErrorIs(t, err, convert.ErrUnknownContext)
	})
}

func TestReadDocumentContext(t *testing.T) {
	data, err := testLoadXML("peppol/base-example.xml")
	require.NoError(t, err)
	in := convert.NewInput(data)

	dc := ubl.ReadDocumentContext(in)
	require.NoError(t, dc.Err)
	assert.Equal(t, ubl.NamespaceUBLInvoice, dc.Namespace)
	assert.Equal(t, ubl.ContextPeppol.CustomizationID, dc.CustomizationID)
	assert.Equal(t, ubl.ContextPeppol.ProfileID, dc.ProfileID)
	assert.Same(t, dc, ubl.ReadDocumentContext(in), "read only once")

	t.Run("malformed", func(t *testing.T) {
		dc := ubl.ReadDocumentContext(convert.NewInput([]byte(`<Invoice><cbc:CustomizationID>`)))
		assert.Error(t, dc.Err)
	})
}

func TestConverterImport(t *testing.T) {
	data, err := testLoadXML("peppol/base-example.xml")
	require.NoError(t, err)
	env, err := convert.Import(data)
	require.NoError(t, err)
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)
	assert.Contains(t, inv.GetAddons(), en16931.V2017)
}

func TestConverterExport(t *testing.T) {
	t.Run("peppol", func(t *testing.T) {
		env := loadTestEnvelope(t, "peppol/invoice-minimal.json")
		out, err := convert.Export(env, ubl.ContextPeppol.Key)
		require.NoError(t, err)
		assert.Equal(t, ubl.ContextPeppol.Key, out.Context.Key)
		assert.Contains(t, string(out.Data), ubl.ContextPeppol.CustomizationID)

		ctx, err := convert.Detect(out.Data)
		require.NoError(t, err)
		assert.Equal(t, ubl.ContextPeppol.Key, ctx.Key, "detected again")
	})
	t.Run("skips context for another schema", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")
		out, err := convert.Export(env, ubl.ContextPeppolInvoiceResponse.Key, ubl.ContextEN16931.Key)
		require.NoError(t, err)
		assert.Equal(t, ubl.ContextEN16931.Key, out.Context.Key)
	})
	t.Run("not supported", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")
		_, err := convert.Export(env, ubl.ContextPeppolInvoiceResponse.Key, ubl.KeyUBL)
		assert.ErrorIs(t, err, convert.ErrNotSupported)
	})
	t.Run("regional context not registered", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")
		_, err := convert.Export(env, "ubl+de-xrechnung-v3")
		assert.ErrorIs(t, err, convert.ErrUnknownContext)
	})
}
