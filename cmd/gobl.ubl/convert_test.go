package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
