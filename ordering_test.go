package ubl_test

import (
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrdering(t *testing.T) {
	t.Run("invoice-minimal.json", func(t *testing.T) {
		doc := testInvoiceFrom(t, "invoice-minimal.json")

		assert.Equal(t, "", doc.BuyerReference)
		assert.NotNil(t, doc.OrderReference)
		assert.Equal(t, "NA", doc.OrderReference.ID)
	})

}

func TestOrderingIssuer(t *testing.T) {
	// issuerEnv loads a complete invoice and attaches an ordering issuer.
	issuerEnv := func(t *testing.T) *gobl.Envelope {
		t.Helper()
		env := loadTestEnvelope(t, "invoice-complete.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		if inv.Ordering == nil {
			inv.Ordering = &bill.Ordering{}
		}
		inv.Ordering.Issuer = &org.Party{
			Name: "Billing Service Provider SL",
		}
		require.NoError(t, env.Calculate())
		return env
	}

	t.Run("maps ordering issuer to supplier ServiceProviderParty", func(t *testing.T) {
		doc, err := ubl.ConvertInvoice(issuerEnv(t))
		require.NoError(t, err)

		sp := doc.AccountingSupplierParty.Party.ServiceProviderParty
		require.NotNil(t, sp, "ServiceProviderParty should be set from ordering.issuer")
		require.NotNil(t, sp.Party)
		require.NotNil(t, sp.Party.PartyName)
		assert.Equal(t, "Billing Service Provider SL", sp.Party.PartyName.Name)
	})

	t.Run("round-trips issuer back to GOBL ordering", func(t *testing.T) {
		doc, err := ubl.ConvertInvoice(issuerEnv(t))
		require.NoError(t, err)
		data, err := ubl.Bytes(doc)
		require.NoError(t, err)

		parsed, err := ubl.Parse(data)
		require.NoError(t, err)
		out, ok := parsed.(*ubl.Invoice)
		require.True(t, ok)
		outEnv, err := out.Convert()
		require.NoError(t, err)
		outInv, ok := outEnv.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.NotNil(t, outInv.Ordering)
		require.NotNil(t, outInv.Ordering.Issuer)
		assert.Equal(t, "Billing Service Provider SL", outInv.Ordering.Issuer.Name)
	})
}
