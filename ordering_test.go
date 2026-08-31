package ubl_test

import (
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
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

func TestOrderingSeller(t *testing.T) {
	// sellerEnv loads a complete invoice and attaches the party liable for
	// the tax, which UBL carries as the BG-11 tax representative.
	sellerEnv := func(t *testing.T) *gobl.Envelope {
		t.Helper()
		env := loadTestEnvelope(t, "invoice-complete.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		if inv.Ordering == nil {
			inv.Ordering = &bill.Ordering{}
		}
		inv.Ordering.Seller = &org.Party{
			Name: "Tax Handling Company GmbH",
			TaxID: &tax.Identity{
				Country: "DE",
				Code:    "282741168",
			},
		}
		require.NoError(t, env.Calculate())
		return env
	}

	t.Run("maps ordering seller to TaxRepresentativeParty", func(t *testing.T) {
		doc, err := ubl.ConvertInvoice(sellerEnv(t))
		require.NoError(t, err)

		// The supplier keeps the BG-4 seller position.
		require.NotNil(t, doc.AccountingSupplierParty.Party.PartyName)
		assert.Equal(t, "Provide One GmbH", doc.AccountingSupplierParty.Party.PartyName.Name)

		rep := doc.TaxRepresentativeParty
		require.NotNil(t, rep, "TaxRepresentativeParty should be set from ordering.seller")
		require.NotNil(t, rep.PartyName)
		assert.Equal(t, "Tax Handling Company GmbH", rep.PartyName.Name)
		require.Len(t, rep.PartyTaxScheme, 1)
		assert.Equal(t, "DE282741168", rep.PartyTaxScheme[0].CompanyID.Value)
	})

	t.Run("round-trips seller back to GOBL ordering", func(t *testing.T) {
		doc, err := ubl.ConvertInvoice(sellerEnv(t))
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

		require.NotNil(t, outInv.Supplier)
		assert.Equal(t, "Provide One GmbH", outInv.Supplier.Name)
		require.NotNil(t, outInv.Ordering)
		require.NotNil(t, outInv.Ordering.Seller)
		assert.Equal(t, "Tax Handling Company GmbH", outInv.Ordering.Seller.Name)
		require.NotNil(t, outInv.Ordering.Seller.TaxID)
		assert.Equal(t, cbc.Code("282741168"), outInv.Ordering.Seller.TaxID.Code)
	})
}

func TestOrderingCost(t *testing.T) {
	// costEnv loads a complete invoice and attaches a buyer accounting reference (BT-19).
	costEnv := func(t *testing.T) *gobl.Envelope {
		t.Helper()
		env := loadTestEnvelope(t, "invoice-complete.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		if inv.Ordering == nil {
			inv.Ordering = &bill.Ordering{}
		}
		inv.Ordering.Cost = "1287:65464"
		require.NoError(t, env.Calculate())
		return env
	}

	t.Run("maps ordering cost to AccountingCost", func(t *testing.T) {
		doc, err := ubl.ConvertInvoice(costEnv(t))
		require.NoError(t, err)

		assert.Equal(t, "1287:65464", doc.AccountingCost)
	})

	t.Run("round-trips accounting cost back to GOBL ordering", func(t *testing.T) {
		doc, err := ubl.ConvertInvoice(costEnv(t))
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
		assert.Equal(t, cbc.Code("1287:65464"), outInv.Ordering.Cost)
	})
}

func TestContractReferenceType(t *testing.T) {
	env := loadTestEnvelope(t, "peppol/invoice-with-contract-ref.json")

	doc, err := ubl.ConvertInvoice(env, ubl.WithContext(ubl.ContextPeppol))
	require.NoError(t, err)

	require.NotEmpty(t, doc.ContractDocumentReference)
	assert.Equal(t, "MARCHE", doc.ContractDocumentReference[0].DocumentType)

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

	require.NotEmpty(t, outInv.Ordering.Contracts)
	assert.Equal(t, "MARCHE", outInv.Ordering.Contracts[0].Reason)
}
