package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/bill"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertBuildOptionsNemhandel(t *testing.T) {
	env := loadTestEnvelope(t)

	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)
	inv.SetAddons(en16931.V2017)
	require.NoError(t, inv.Calculate())

	opts, err := (&convertOpts{contextName: "nemhandel"}).buildOptions()
	require.NoError(t, err)

	doc, err := ubl.ConvertInvoice(env, opts...)
	require.NoError(t, err)
	assert.Equal(t, "urn:fdc:oioubl.dk:trns:billing:invoice:3.0", doc.CustomizationID)
	assert.Equal(t, "urn:fdc:oioubl.dk:bis:billing_with_response:3", doc.ProfileID)
	assert.Equal(t, "2.1", doc.UBLVersionID)
	assert.Empty(t, doc.UUID)
}

func TestConvertBuildOptionsProfileOverride(t *testing.T) {
	env := loadTestEnvelope(t)

	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)
	inv.SetAddons(en16931.V2017)
	require.NoError(t, inv.Calculate())

	opts, err := (&convertOpts{contextName: "nemhandel", profileID: "custom-profile"}).buildOptions()
	require.NoError(t, err)

	doc, err := ubl.ConvertInvoice(env, opts...)
	require.NoError(t, err)
	assert.Equal(t, "custom-profile", doc.ProfileID)
}

func TestConvertBuildOptionsOIOUBL21(t *testing.T) {
	env := loadTestEnvelope(t)

	opts, err := (&convertOpts{contextName: "oioubl-2.1"}).buildOptions()
	require.NoError(t, err)

	doc, err := ubl.ConvertInvoice(env, opts...)
	require.NoError(t, err)
	assert.Equal(t, "OIOUBL-2.1", doc.CustomizationID)
	assert.Equal(t, "urn:www.nesubl.eu:profiles:profile5:ver2.0", doc.ProfileID)
}

func TestConvertBuildOptionsUnknownContext(t *testing.T) {
	_, err := (&convertOpts{contextName: "unknown"}).buildOptions()
	require.EqualError(t, err, `unknown context "unknown"`)
}

func TestConvertBuildOptionsContextAliases(t *testing.T) {
	tests := []struct {
		name        string
		context     string
		convertible bool
		expected    ubl.Context
	}{
		{name: "en alias", context: "en", convertible: true, expected: ubl.ContextEN16931},
		{name: "peppol alias", context: "peppol", convertible: true, expected: ubl.ContextPeppol},
		{name: "peppol self billed alias", context: "peppol-selfbilled", convertible: true, expected: ubl.ContextPeppolSelfBilled},
		{name: "xrechnung alias", context: "xrechnung", convertible: false},
		{name: "france cius alias", context: "fr-cius", convertible: false},
		{name: "france extended alias", context: "fr-extended", convertible: false},
		{name: "oioubl alias", context: "oioubl", convertible: true, expected: ubl.ContextOIOUBL},
		{name: "oioubl 2.1 alias", context: "oioubl-2.1", convertible: true, expected: ubl.ContextOIOUBL21},
		{name: "mixed case alias", context: "PePpOl-SeLf", convertible: true, expected: ubl.ContextPeppolSelfBilled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := (&convertOpts{contextName: tt.context}).buildOptions()
			require.NoError(t, err)

			if !tt.convertible {
				return
			}

			env := loadTestEnvelope(t)
			doc, err := ubl.ConvertInvoice(env, opts...)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.CustomizationID, doc.CustomizationID)
			assert.Equal(t, tt.expected.ProfileID, doc.ProfileID)
		})
	}
}

func TestConvertRunEErrors(t *testing.T) {
	t.Run("no args", func(t *testing.T) {
		cmd := root().cmd()
		cmd.SetArgs([]string{"convert"})
		err := cmd.Execute()
		require.EqualError(t, err, "expected one or two arguments, the command usage is `gobl.ubl convert <infile> [outfile]`")
	})

	t.Run("too many args", func(t *testing.T) {
		cmd := root().cmd()
		cmd.SetArgs([]string{"convert", "a", "b", "c"})
		err := cmd.Execute()
		require.EqualError(t, err, "expected one or two arguments, the command usage is `gobl.ubl convert <infile> [outfile]`")
	})

	t.Run("invalid context", func(t *testing.T) {
		inPath := filepath.Join("..", "..", "test", "data", "convert", "invoice-minimal.json")
		outPath := filepath.Join(t.TempDir(), "out.xml")
		cmd := root().cmd()
		cmd.SetArgs([]string{"convert", "--context", "nope", inPath, outPath})
		err := cmd.Execute()
		require.EqualError(t, err, `unknown context "nope"`)
	})

	t.Run("unknown xml document type", func(t *testing.T) {
		inPath := filepath.Join(t.TempDir(), "unknown.xml")
		require.NoError(t, os.WriteFile(inPath, []byte("<foo/>"), 0o644))
		outPath := filepath.Join(t.TempDir(), "out.json")

		cmd := root().cmd()
		cmd.SetArgs([]string{"convert", inPath, outPath})
		err := cmd.Execute()
		require.ErrorContains(t, err, "building GOBL envelope: unknown document type")
	})
}

func TestConvertXMLToJSONEnvelope(t *testing.T) {
	inPath := filepath.Join("..", "..", "test", "data", "convert", "out", "oioubl30-invoice-minimal.xml")
	outPath := filepath.Join(t.TempDir(), "out.json")

	cmd := root().cmd()
	cmd.SetArgs([]string{"convert", inPath, outPath})
	require.NoError(t, cmd.Execute())

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)

	env := new(gobl.Envelope)
	require.NoError(t, json.Unmarshal(data, env))

	doc := env.Extract()
	inv, ok := doc.(*bill.Invoice)
	require.True(t, ok, "expected extracted document to be a bill.Invoice")
	assert.Equal(t, "SAMPLE-001", inv.Code.String())
	assert.Equal(t, "EUR", string(inv.Currency))
}

func TestConvertJSONToXMLWithOIOUBL21Context(t *testing.T) {
	inPath := filepath.Join("..", "..", "test", "data", "convert", "oioubl21-invoice-minimal.json")
	outPath := filepath.Join(t.TempDir(), "out.xml")

	cmd := root().cmd()
	cmd.SetArgs([]string{"convert", "--context", "oioubl-2.1", inPath, outPath})
	require.NoError(t, cmd.Execute())

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	xml := string(data)
	assert.Contains(t, xml, "<cbc:CustomizationID>OIOUBL-2.1</cbc:CustomizationID>")
	assert.Contains(t, xml, "<cbc:ProfileID schemeAgencyID=\"320\" schemeID=\"urn:oioubl:id:profileid-1.4\">urn:www.nesubl.eu:profiles:profile5:ver2.0</cbc:ProfileID>")
	assert.Contains(t, xml, "<cbc:InvoiceTypeCode listAgencyID=\"320\" listID=\"urn:oioubl:codelist:invoicetypecode-1.1\">380</cbc:InvoiceTypeCode>")
}

func TestConvertOIOUBL21BarePartyFallbacks(t *testing.T) {
	inPath := filepath.Join("..", "..", "test", "data", "convert", "oioubl21-invoice-bare.json")
	outPath := filepath.Join(t.TempDir(), "out.xml")

	cmd := root().cmd()
	cmd.SetArgs([]string{"convert", "--context", "oioubl-2.1", inPath, outPath})
	require.NoError(t, cmd.Execute())

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	xml := string(data)

	// EndpointID should fall back to CVR number when no inboxes are present
	assert.Contains(t, xml, `<cbc:EndpointID schemeID="DK:CVR">DK37990485</cbc:EndpointID>`)
	assert.Contains(t, xml, `<cbc:EndpointID schemeID="DK:CVR">DK47458714</cbc:EndpointID>`)

	// PartyName should fall back to RegistrationName when no alias is present
	assert.Contains(t, xml, "<cac:PartyName>\n        <cbc:Name>Worksome Aps</cbc:Name>")
	assert.Contains(t, xml, "<cac:PartyName>\n        <cbc:Name>Lego System A/S</cbc:Name>")

	// Contact must always be present with a default ID
	assert.Contains(t, xml, "<cac:Contact>\n        <cbc:ID>1</cbc:ID>\n      </cac:Contact>")
}

func TestConvertCreditNoteToXMLWithOIOUBL21Context(t *testing.T) {
	inPath := filepath.Join("..", "..", "test", "data", "convert", "oioubl21-credit-note-minimal.json")
	outPath := filepath.Join(t.TempDir(), "out.xml")

	cmd := root().cmd()
	cmd.SetArgs([]string{"convert", "--context", "oioubl-2.1", inPath, outPath})
	require.NoError(t, cmd.Execute())

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	xml := string(data)
	assert.Contains(t, xml, "<CreditNote ")
	assert.Contains(t, xml, "<cbc:CustomizationID>OIOUBL-2.1</cbc:CustomizationID>")
	assert.Contains(t, xml, "<cbc:ProfileID schemeAgencyID=\"320\" schemeID=\"urn:oioubl:id:profileid-1.4\">urn:www.nesubl.eu:profiles:profile5:ver2.0</cbc:ProfileID>")
	assert.Contains(t, xml, "<cbc:CreditNoteTypeCode listAgencyID=\"320\" listID=\"urn:oioubl:codelist:invoicetypecode-1.1\">381</cbc:CreditNoteTypeCode>")
}

func loadTestEnvelope(t *testing.T) *gobl.Envelope {
	t.Helper()

	path := filepath.Join("..", "..", "test", "data", "convert", "invoice-minimal.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	env := new(gobl.Envelope)
	require.NoError(t, json.Unmarshal(data, env))
	require.NoError(t, env.Calculate())
	require.NoError(t, env.Validate())

	return env
}
