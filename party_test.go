package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewParty(t *testing.T) {
	t.Run("invoice-complete.json", func(t *testing.T) {
		doc := testInvoiceFrom(t, "invoice-complete.json")

		assert.Equal(t, "inbox@example.com", doc.AccountingSupplierParty.Party.EndpointID.Value)
		assert.Equal(t, "EM", doc.AccountingSupplierParty.Party.EndpointID.SchemeID)
	})

	t.Run("identities with iso scheme id propagate to SchemeID", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-complete.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Supplier identity without a Scope, carrying iso scheme ID:
		// exercises newParty's third-pass branch.
		inv.Supplier.Identities = []*org.Identity{
			{
				Code: "TEST-001",
				Ext:  tax.ExtensionsOf(cbc.CodeMap{iso.ExtKeySchemeID: "0088"}),
			},
		}

		// Payee with a legal identity carrying iso scheme ID:
		// exercises both passes inside newPayeeParty.
		if inv.Payment == nil {
			inv.Payment = &bill.PaymentDetails{}
		}
		inv.Payment.Payee = &org.Party{
			Name: "Test Payee",
			Identities: []*org.Identity{
				{
					Code:  "PAYEE-001",
					Scope: org.IdentityScopeLegal,
					Ext:   tax.ExtensionsOf(cbc.CodeMap{iso.ExtKeySchemeID: "0088"}),
				},
			},
		}

		require.NoError(t, env.Calculate())
		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)

		require.NotEmpty(t, doc.AccountingSupplierParty.Party.PartyIdentification)
		pid := doc.AccountingSupplierParty.Party.PartyIdentification[0]
		require.NotNil(t, pid.ID.SchemeID)
		assert.Equal(t, "0088", *pid.ID.SchemeID)
		assert.Equal(t, "TEST-001", pid.ID.Value)

		require.NotNil(t, doc.PayeeParty)
		require.NotEmpty(t, doc.PayeeParty.PartyIdentification)
		require.NotNil(t, doc.PayeeParty.PartyIdentification[0].ID.SchemeID)
		assert.Equal(t, "0088", *doc.PayeeParty.PartyIdentification[0].ID.SchemeID)
		require.NotNil(t, doc.PayeeParty.PartyLegalEntity)
		require.NotNil(t, doc.PayeeParty.PartyLegalEntity.CompanyID.SchemeID)
		assert.Equal(t, "0088", *doc.PayeeParty.PartyLegalEntity.CompanyID.SchemeID)
	})

	// France "code routage" / Chorus Pro "Code Service" is modelled as an
	// identity with key "private-id". The CTC (B2B CIUS) addon sets the
	// iso-scheme-id ext to 0224, but the Factur-X (France Extended / B2G) addon
	// is a placeholder and does not. The customer identity must still be
	// serialized as PartyIdentification schemeID="0224" (BR-FR-CPRO-11) here, so
	// the scheme is derived from the key when the ext is absent.
	t.Run("private-id key maps to scheme 0224 without ext", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-complete.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// No iso-scheme-id ext set, mirroring the Factur-X / Extended profile.
		inv.Customer.Identities = []*org.Identity{
			{Key: "private-id", Code: "SERVICE-ACHATS-01"},
		}

		require.NoError(t, env.Calculate())
		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)

		var found bool
		for _, pid := range doc.AccountingCustomerParty.Party.PartyIdentification {
			if pid.ID != nil && pid.ID.Value == "SERVICE-ACHATS-01" {
				require.NotNil(t, pid.ID.SchemeID)
				assert.Equal(t, "0224", *pid.ID.SchemeID)
				found = true
			}
		}
		assert.True(t, found, "expected code routage identity in PartyIdentification")
	})
}
