package ubl

import (
	"testing"

	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPartyDKSchemeGatedToOIOUBL(t *testing.T) {
	mk := func() *org.Party {
		return &org.Party{
			Name:  "Eksempel ApS",
			TaxID: &tax.Identity{Country: "DK", Code: "12345674"},
		}
	}

	t.Run("OIOUBL scopes the DK:SE tax scheme and fabricates the legal entity ID", func(t *testing.T) {
		p := newParty(mk(), ContextOIOUBL21)
		require.Len(t, p.PartyTaxScheme, 1)
		require.NotNil(t, p.PartyTaxScheme[0].CompanyID.SchemeID)
		assert.Equal(t, icdDKSE, *p.PartyTaxScheme[0].CompanyID.SchemeID)
		require.NotNil(t, p.PartyLegalEntity)
		require.NotNil(t, p.PartyLegalEntity.CompanyID)
		require.NotNil(t, p.PartyLegalEntity.CompanyID.SchemeID)
		assert.Equal(t, icdDKCVR, *p.PartyLegalEntity.CompanyID.SchemeID)
	})

	t.Run("non-OIOUBL leaves the DK tax id unscoped and the legal entity ID unfabricated", func(t *testing.T) {
		p := newParty(mk(), ContextEN16931)
		require.Len(t, p.PartyTaxScheme, 1)
		assert.Nil(t, p.PartyTaxScheme[0].CompanyID.SchemeID, "no DK:SE leak outside OIOUBL")
		require.NotNil(t, p.PartyLegalEntity)
		assert.Nil(t, p.PartyLegalEntity.CompanyID, "legal entity ID must not be fabricated from the tax id")
	})
}

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

	t.Run("country schemes restore to their canonical ICD", func(t *testing.T) {
		p := goblParty(&Party{
			EndpointID: &EndpointID{SchemeID: "IS:KT", Value: "5504033150"},
		}, o)
		require.Len(t, p.Endpoints, 1)
		assert.Equal(t, "iso6523-actorid-upis::0196:5504033150", p.Endpoints[0].URI.String(),
			"legacy EAS 9917 also feeds IS:KT; the canonical 0196 wins on parse")
	})

	t.Run("unmapped scheme falls back to an inbox", func(t *testing.T) {
		p := goblParty(&Party{
			EndpointID: &EndpointID{SchemeID: "DK:VANS", Value: "1234567890"},
		}, o)
		assert.Empty(t, p.Endpoints)
		require.Len(t, p.Inboxes, 1)
		assert.Equal(t, "DK:VANS", p.Inboxes[0].Scheme.String())
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
