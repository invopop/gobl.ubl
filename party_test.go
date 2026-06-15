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

	t.Run("norwegian VAT numbers carry the MVA suffix", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-complete.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Supplier.TaxID = &tax.Identity{Country: "NO", Code: "923456783"}
		require.NoError(t, env.Calculate())
		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)

		require.NotEmpty(t, doc.AccountingSupplierParty.Party.PartyTaxScheme)
		assert.Equal(t, "NO923456783MVA", *doc.AccountingSupplierParty.Party.PartyTaxScheme[0].CompanyID)

		// An already-suffixed code must not be doubled.
		inv.Supplier.TaxID.Code = "923456783MVA"
		doc, err = ubl.ConvertInvoice(env)
		require.NoError(t, err)
		assert.Equal(t, "NO923456783MVA", *doc.AccountingSupplierParty.Party.PartyTaxScheme[0].CompanyID)
	})
}
