package ubl_test

import (
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConverterContexts(t *testing.T) {
	for _, k := range []cbc.Key{
		ubl.KeyUBL,
		ubl.FormatEN16931.Key,
		ubl.FormatPeppol.Key,
	} {
		ctx := convert.FormatFor(k)
		require.NotNil(t, ctx, k)
		assert.Equal(t, cbc.Key("ubl"), ctx.Syntax)
		assert.Empty(t, ctx.Countries)
	}

	t.Run("layered keys", func(t *testing.T) {
		assert.Equal(t, cbc.Key("ubl+en16931-v2017"), ubl.FormatEN16931.Key)
		assert.Equal(t, cbc.Key("ubl+peppol+bis-v3"), ubl.FormatPeppol.Key)
	})
}

func TestConverterDetect(t *testing.T) {
	tests := []struct {
		file string
		key  cbc.Key
	}{
		{"en16931/ubl-example1.xml", ubl.FormatEN16931.Key},
		{"peppol/base-example.xml", ubl.FormatPeppol.Key},
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

	t.Run("not ubl", func(t *testing.T) {
		for _, data := range []string{
			`<?xml version="1.0"?><Invoice xmlns="urn:example"/>`,
			`{"$schema":"https://gobl.org/draft-0/bill/invoice"}`,
			``,
		} {
			_, err := convert.Detect([]byte(data))
			assert.ErrorIs(t, err, convert.ErrUnknownFormat, data)
		}
	})

	t.Run("limited to keys", func(t *testing.T) {
		data, err := testLoadXML("peppol/base-example.xml")
		require.NoError(t, err)
		_, err = convert.Detect(data, ubl.FormatEN16931.Key)
		assert.ErrorIs(t, err, convert.ErrUnknownFormat)
	})
}

func TestReadDocumentContext(t *testing.T) {
	data, err := testLoadXML("peppol/base-example.xml")
	require.NoError(t, err)
	in := convert.NewInput(data)

	dc := ubl.ReadHeader(in)
	require.NoError(t, dc.Err)
	assert.Equal(t, ubl.NamespaceUBLInvoice, dc.Namespace)
	assert.Equal(t, ubl.FormatPeppol.CustomizationID, dc.CustomizationID)
	assert.Equal(t, ubl.FormatPeppol.ProfileID, dc.ProfileID)
	assert.Same(t, dc, ubl.ReadHeader(in), "read only once")

	t.Run("malformed", func(t *testing.T) {
		dc := ubl.ReadHeader(convert.NewInput([]byte(`<Invoice><cbc:CustomizationID>`)))
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
		out, err := convert.Export(env, ubl.FormatPeppol.Key)
		require.NoError(t, err)
		assert.Equal(t, ubl.FormatPeppol.Key, out.Format.Key)
		assert.Contains(t, string(out.Data), ubl.FormatPeppol.CustomizationID)

		ctx, err := convert.Detect(out.Data)
		require.NoError(t, err)
		assert.Equal(t, ubl.FormatPeppol.Key, ctx.Key, "detected again")
	})
	t.Run("skips format for another schema", func(t *testing.T) {
		env, err := gobl.Envelop(basePeppolStatus())
		require.NoError(t, err)
		out, err := convert.Export(env, ubl.FormatEN16931.Key, ubl.FormatPeppol.Key)
		require.NoError(t, err)
		assert.Equal(t, ubl.FormatPeppol.Key, out.Format.Key)
		assert.Contains(t, string(out.Data), ubl.SpecificationPeppolInvoiceResponse.CustomizationID)
	})
	t.Run("not supported", func(t *testing.T) {
		env, err := gobl.Envelop(basePeppolStatus())
		require.NoError(t, err)
		_, err = convert.Export(env, ubl.FormatEN16931.Key, ubl.KeyUBL)
		assert.ErrorIs(t, err, convert.ErrNotSupported)
	})
	t.Run("missing specification addon", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")
		env.Extract().(*bill.Invoice).SetAddons()
		_, err := convert.Export(env, ubl.FormatPeppol.Key)
		assert.ErrorIs(t, err, convert.ErrNotSupported)
		var ge *gobl.Error
		require.ErrorAs(t, err, &ge)
		require.NotNil(t, ge.Faults())
		f := ge.Faults().First()
		assert.Equal(t, rules.Code("UBL-PEPPOL-BIS-V3-BILL-INVOICE-01"), f.Code())
		assert.Equal(t, []string{"$.$addons"}, f.Paths())
		assert.Equal(t, "requires addon eu-en16931-v2017", f.Message())
	})
	t.Run("regional context not registered", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")
		_, err := convert.Export(env, "ubl+de-xrechnung-v3")
		assert.ErrorIs(t, err, convert.ErrUnknownFormat)
	})
}
