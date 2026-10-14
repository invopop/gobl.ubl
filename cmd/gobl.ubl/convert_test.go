package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/flimzy/testy"
)

func TestConvertCommand(t *testing.T) {
	t.Run("UBL XML input converts to a GOBL envelope", func(t *testing.T) {
		out := runConvert(t, "../../test/data/parse/en16931/ubl-example1.xml")

		// Regression for #99: the XML branch must return a GOBL envelope, not
		// the raw parsed *ubl.Invoice struct (whose JSON has CACNamespace etc.).
		require.Contains(t, out, "https://gobl.org/draft-0/envelope")
		require.NotContains(t, out, "CACNamespace")

		env := new(gobl.Envelope)
		require.NoError(t, json.Unmarshal([]byte(out), env))
		assert.Equal(t, "https://gobl.org/draft-0/envelope", string(env.Schema))
		_, ok := env.Extract().(*bill.Invoice)
		assert.True(t, ok, "envelope should wrap a bill.Invoice")
	})

	t.Run("GOBL JSON input converts to a UBL document", func(t *testing.T) {
		out := runConvert(t, "../../test/data/parse/en16931/out/ubl-example1.json")
		assert.Contains(t, out, "<Invoice")
	})
}

// runConvert runs `gobl.ubl convert <infile>`, capturing stdout.
func runConvert(t *testing.T, infile string) string {
	t.Helper()
	cmd := root().cmd()
	cmd.SetArgs([]string{"convert", infile})
	var out bytes.Buffer
	cmd.SetOut(&out)
	require.NoError(t, cmd.Execute())
	return out.String()
}

func TestConvertCommandErrors(t *testing.T) {
	tests := map[string]struct {
		args  []string
		stdin string
		err   string
	}{
		"no arguments":    {args: []string{"convert"}, err: "expected one or two arguments"},
		"too many":        {args: []string{"convert", "a", "b", "c"}, err: "expected one or two arguments"},
		"missing file":    {args: []string{"convert", "does-not-exist.xml"}, err: "no such file"},
		"not an envelope": {args: []string{"convert", "-"}, stdin: `{"head":1}`, err: "parsing input as GOBL Envelope"},
		"no document":     {args: []string{"convert", "-"}, stdin: `{}`, err: "building UBL document"},
		"not ubl":         {args: []string{"convert", "-"}, stdin: `<Foo/>`, err: "building GOBL envelope"},
		"application response": {
			args:  []string{"convert", "-"},
			stdin: `<ApplicationResponse xmlns="urn:oasis:names:specification:ubl:schema:xsd:ApplicationResponse-2"/>`,
			err:   "unsupported document type",
		},
		"invalid invoice": {
			args:  []string{"convert", "-"},
			stdin: `<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"/>`,
			err:   "building GOBL envelope",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := root().cmd()
			cmd.SetArgs(tt.args)
			cmd.SetIn(strings.NewReader(tt.stdin))
			cmd.SetOut(new(bytes.Buffer))
			err := cmd.Execute()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.err)
		})
	}
}

func TestConvertCommandOutputFile(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.json")
	cmd := root().cmd()
	cmd.SetArgs([]string{"convert", "../../test/data/parse/en16931/ubl-example1.xml", out})
	require.NoError(t, cmd.Execute())

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Contains(t, string(data), "https://gobl.org/draft-0/envelope")
}

func TestRun(t *testing.T) {
	args := os.Args
	t.Cleanup(func() { os.Args = args })
	os.Args = []string{name, "version"}
	var err error
	testy.RedirIO(nil, func() {
		err = run()
	})
	assert.NoError(t, err)
}

func TestConvertCommandOutputError(t *testing.T) {
	out := filepath.Join(t.TempDir(), "missing", "out.json")
	cmd := root().cmd()
	cmd.SetArgs([]string{"convert", "../../test/data/parse/en16931/ubl-example1.xml", out})
	assert.Error(t, cmd.Execute())
}
