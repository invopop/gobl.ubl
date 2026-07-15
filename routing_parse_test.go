package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConvertRoutingFromArgs verifies that a received UBL document records the
// transport addresses it was routed with (WithRouting) on Head.From / Head.To,
// rather than GOBL's document-derived, outgoing-direction guess.
func TestConvertRoutingFromArgs(t *testing.T) {
	xmlData, err := testLoadXML("discount-taxes.xml")
	require.NoError(t, err)
	parse := func(t *testing.T) *ubl.Invoice {
		t.Helper()
		doc, err := ubl.Parse(xmlData)
		require.NoError(t, err)
		inv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)
		return inv
	}

	t.Run("routing args are recorded verbatim on Head.From/To", func(t *testing.T) {
		// The routing layer supplies fully-qualified participant URIs; Convert
		// records them verbatim rather than deriving From/To from the document.
		env, err := parse(t).Convert(ubl.WithRouting(
			"iso6523-actorid-upis::0192:sender",
			"iso6523-actorid-upis::0192:receiver",
		))
		require.NoError(t, err)
		assert.Equal(t, "iso6523-actorid-upis::0192:sender", string(env.Head.From))
		assert.Equal(t, "iso6523-actorid-upis::0192:receiver", string(env.Head.To))
	})
}
