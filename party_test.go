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

		// Payee with a legal identity carrying an ISO scheme ID.
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
		// The sole identity lands in PartyLegalEntity.CompanyID without
		// being duplicated into PartyIdentification (UBL-SR-20).
		assert.Empty(t, doc.PayeeParty.PartyIdentification)
		require.NotNil(t, doc.PayeeParty.PartyLegalEntity)
		require.NotNil(t, doc.PayeeParty.PartyLegalEntity.CompanyID.SchemeID)
		assert.Equal(t, "0088", *doc.PayeeParty.PartyLegalEntity.CompanyID.SchemeID)
	})

	t.Run("norwegian VAT numbers carry the MVA suffix", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-complete.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Supplier.TaxID = &tax.Identity{Country: "NO", Code: "923456783"}
		require.NoError(t, env.Calculate())
		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)

		require.NotEmpty(t, doc.AccountingSupplierParty.Party.PartyTaxScheme)
		assert.Equal(t, "NO923456783MVA", doc.AccountingSupplierParty.Party.PartyTaxScheme[0].CompanyID.Value)

		// An already-suffixed code must not be doubled.
		inv.Supplier.TaxID.Code = "923456783MVA"
		doc, err = ubl.ConvertInvoice(env)
		require.NoError(t, err)
		assert.Equal(t, "NO923456783MVA", doc.AccountingSupplierParty.Party.PartyTaxScheme[0].CompanyID.Value)
	})
}

// TestNewPartyTaxRegistration pins BT-32: the tax scheme code is the French
// one under a French context, and the identity's own type elsewhere.
func TestNewPartyTaxRegistration(t *testing.T) {
	convert := func(t *testing.T, fixture string, id *org.Identity, opts ...ubl.Option) []PartyTaxSchemeView {
		t.Helper()
		env := loadTestEnvelope(t, fixture)
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		inv.Supplier.Identities = []*org.Identity{id}
		require.NoError(t, env.Calculate())

		doc, err := ubl.ConvertInvoice(env, opts...)
		require.NoError(t, err)

		out := make([]PartyTaxSchemeView, 0)
		for _, pts := range doc.AccountingSupplierParty.Party.PartyTaxScheme {
			out = append(out, PartyTaxSchemeView{
				Scheme: pts.TaxScheme.ID.Value,
				Code:   pts.CompanyID.Value,
			})
		}
		return out
	}

	t.Run("french context pins the scheme", func(t *testing.T) {
		schemes := convert(t, "france-cius/invoice-fr-cius.json",
			&org.Identity{Scope: org.IdentityScopeTax, Code: "483671517"},
			ubl.WithContext(ubl.ContextPeppolFranceCIUS))

		require.NotEmpty(t, schemes)
		last := schemes[len(schemes)-1]
		assert.Equal(t, "LOC", last.Scheme)
		assert.Equal(t, "483671517", last.Code)
	})

	t.Run("elsewhere the identity type is used", func(t *testing.T) {
		schemes := convert(t, "invoice-complete.json",
			&org.Identity{Scope: org.IdentityScopeTax, Type: "TAX", Code: "Foretaksregisteret"})

		require.NotEmpty(t, schemes)
		last := schemes[len(schemes)-1]
		assert.Equal(t, "TAX", last.Scheme)
		assert.Equal(t, "Foretaksregisteret", last.Code)
	})

	t.Run("without a type the french code is the fallback", func(t *testing.T) {
		schemes := convert(t, "invoice-complete.json",
			&org.Identity{Scope: org.IdentityScopeTax, Code: "483671517"})

		require.NotEmpty(t, schemes)
		assert.Equal(t, "LOC", schemes[len(schemes)-1].Scheme)
	})
}

// PartyTaxSchemeView flattens a PartyTaxScheme for the assertions above.
type PartyTaxSchemeView struct {
	Scheme string
	Code   string
}

// TestNewPartyEndpointID covers BT-34 / BT-49, the party's electronic address.
// GOBL v0.505 moved it to org.Endpoint and deprecated org.Inbox, so both models
// have to be read: Peppol rejects a party without an address (BR-62, BR-63).
func TestNewPartyEndpointID(t *testing.T) {
	convert := func(t *testing.T, apply func(inv *bill.Invoice)) *ubl.Invoice {
		t.Helper()
		env := loadTestEnvelope(t, "peppol/invoice-complete.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		apply(inv)
		require.NoError(t, env.Calculate())

		doc, err := ubl.ConvertInvoice(env, ubl.WithContext(ubl.ContextPeppol))
		require.NoError(t, err)
		return doc
	}

	t.Run("endpoints only", func(t *testing.T) {
		doc := convert(t, func(inv *bill.Invoice) {
			inv.Supplier.Inboxes = nil
			inv.Supplier.Endpoints = []*org.Endpoint{
				{URI: "iso6523-actorid-upis::9930:111111125"},
			}
		})

		eid := doc.AccountingSupplierParty.Party.EndpointID
		require.NotNil(t, eid)
		assert.Equal(t, "9930", eid.SchemeID)
		assert.Equal(t, "111111125", eid.Value)
	})

	t.Run("mailto endpoint", func(t *testing.T) {
		doc := convert(t, func(inv *bill.Invoice) {
			inv.Supplier.Inboxes = nil
			inv.Supplier.Endpoints = []*org.Endpoint{
				{URI: "mailto:billing@example.com"},
			}
		})

		eid := doc.AccountingSupplierParty.Party.EndpointID
		require.NotNil(t, eid)
		assert.Equal(t, ubl.SchemeIDEmail, eid.SchemeID)
		assert.Equal(t, "billing@example.com", eid.Value)
	})

	t.Run("deprecated inboxes only", func(t *testing.T) {
		doc := convert(t, func(inv *bill.Invoice) {
			inv.Supplier.Endpoints = nil
			inv.Supplier.Inboxes = []*org.Inbox{
				{Scheme: "0088", Code: "7300010000001"},
			}
		})

		eid := doc.AccountingSupplierParty.Party.EndpointID
		require.NotNil(t, eid)
		assert.Equal(t, "0088", eid.SchemeID)
		assert.Equal(t, "7300010000001", eid.Value)
	})

	t.Run("no electronic address", func(t *testing.T) {
		doc := convert(t, func(inv *bill.Invoice) {
			inv.Supplier.Endpoints = nil
			inv.Supplier.Inboxes = nil
		})

		assert.Nil(t, doc.AccountingSupplierParty.Party.EndpointID)
	})
}
