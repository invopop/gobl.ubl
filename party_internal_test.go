package ubl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoblPartyOIOUBLEndpoints(t *testing.T) {
	o := &options{context: ContextOIOUBL21}

	t.Run("mapped scheme becomes an ISO 6523 endpoint", func(t *testing.T) {
		p := goblParty(&Party{
			EndpointID: &EndpointID{SchemeID: "DK:CVR", Value: "DK12345674"},
		}, o)
		require.Len(t, p.Endpoints, 1)
		assert.Equal(t, "iso6523-actorid-upis::0184:12345674", p.Endpoints[0].URI.String(),
			"the wire-only DK prefix is stripped from the participant code")
		assert.Empty(t, p.Inboxes)
	})

	t.Run("GLN scheme maps without value rewriting", func(t *testing.T) {
		p := goblParty(&Party{
			EndpointID: &EndpointID{SchemeID: "GLN", Value: "5790000000000"},
		}, o)
		require.Len(t, p.Endpoints, 1)
		assert.Equal(t, "iso6523-actorid-upis::0088:5790000000000", p.Endpoints[0].URI.String())
	})

	t.Run("unmapped scheme falls back to an inbox", func(t *testing.T) {
		p := goblParty(&Party{
			EndpointID: &EndpointID{SchemeID: "DK:P", Value: "1234567890"},
		}, o)
		assert.Empty(t, p.Endpoints)
		require.Len(t, p.Inboxes, 1)
		assert.Equal(t, "DK:P", p.Inboxes[0].Scheme.String())
		assert.Equal(t, "1234567890", p.Inboxes[0].Code.String())
	})

	t.Run("non-OIOUBL context keeps the inbox form", func(t *testing.T) {
		p := goblParty(&Party{
			EndpointID: &EndpointID{SchemeID: "0184", Value: "12345674"},
		}, &options{context: ContextEN16931})
		assert.Empty(t, p.Endpoints)
		require.Len(t, p.Inboxes, 1)
		assert.Equal(t, "0184", p.Inboxes[0].Scheme.String())
	})
}
