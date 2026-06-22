package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertBuildOptionsPeppol(t *testing.T) {
	env := loadTestEnvelope(t)

	opts, err := (&convertOpts{contextName: "peppol"}).buildOptions()
	require.NoError(t, err)

	doc, err := ubl.ConvertInvoice(env, opts...)
	require.NoError(t, err)
	assert.Equal(t, ubl.ContextPeppol.CustomizationID, doc.CustomizationID)
	assert.Equal(t, ubl.ContextPeppol.ProfileID, doc.ProfileID)
}

func TestConvertBuildOptionsProfileOverride(t *testing.T) {
	env := loadTestEnvelope(t)

	opts, err := (&convertOpts{contextName: "peppol", profileID: "custom-profile"}).buildOptions()
	require.NoError(t, err)

	doc, err := ubl.ConvertInvoice(env, opts...)
	require.NoError(t, err)
	assert.Equal(t, "custom-profile", doc.ProfileID)
}

func TestConvertBuildOptionsUnknownContext(t *testing.T) {
	_, err := (&convertOpts{contextName: "unknown"}).buildOptions()
	require.EqualError(t, err, `unknown context "unknown"`)
}

func TestConvertBuildOptionsContextNames(t *testing.T) {
	tests := []struct {
		name        string
		context     string
		convertible bool
		expected    ubl.Context
	}{
		{name: "en16931", context: "en16931", convertible: true, expected: ubl.ContextEN16931},
		{name: "peppol", context: "peppol", convertible: true, expected: ubl.ContextPeppol},
		{name: "peppol-self-billed", context: "peppol-self-billed", convertible: false},
		{name: "xrechnung", context: "xrechnung", convertible: false},
		{name: "peppol-france-cius", context: "peppol-france-cius", convertible: false},
		{name: "peppol-france-extended", context: "peppol-france-extended", convertible: false},
		{name: "zatca", context: "zatca", convertible: false},
		{name: "mixed case", context: "PeppOl", convertible: true, expected: ubl.ContextPeppol},
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
	inPath := filepath.Join("..", "..", "test", "data", "convert", "peppol", "out", "invoice-minimal.xml")
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

func loadTestEnvelope(t *testing.T) *gobl.Envelope {
	t.Helper()

	path := filepath.Join("..", "..", "test", "data", "convert", "invoice-complete.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	env := new(gobl.Envelope)
	require.NoError(t, json.Unmarshal(data, env))
	require.NoError(t, env.Calculate())
	require.NoError(t, env.Validate())

	return env
}
