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
	t.Run("invoice-de-de.json", func(t *testing.T) {
		doc := testInvoiceFrom(t, "invoice-de-de.json")

		assert.Equal(t, "DE111111125", doc.AccountingSupplierParty.Party.PartyTaxScheme[0].CompanyID.Value)
		assert.Equal(t, "Provide One GmbH", *doc.AccountingSupplierParty.Party.PartyLegalEntity.RegistrationName)
		assert.Equal(t, "+49100200300", *doc.AccountingSupplierParty.Party.Contact.Telephone)
		assert.Equal(t, "billing@example.com", *doc.AccountingSupplierParty.Party.Contact.ElectronicMail)

		assert.Equal(t, "Dietmar-Hopp-Allee 16", *doc.AccountingSupplierParty.Party.PostalAddress.StreetName)
		assert.Equal(t, "Walldorf", *doc.AccountingSupplierParty.Party.PostalAddress.CityName)
		assert.Equal(t, "69190", *doc.AccountingSupplierParty.Party.PostalAddress.PostalZone)
		assert.Equal(t, "DE", doc.AccountingSupplierParty.Party.PostalAddress.Country.IdentificationCode)

		assert.Equal(t, "DE282741168", doc.AccountingCustomerParty.Party.PartyTaxScheme[0].CompanyID.Value)
		assert.Equal(t, "Sample Consumer", *doc.AccountingCustomerParty.Party.PartyLegalEntity.RegistrationName)
		assert.Equal(t, "email@sample.com", *doc.AccountingCustomerParty.Party.Contact.ElectronicMail)

		assert.Equal(t, "Werner-Heisenberg-Allee 25", *doc.AccountingCustomerParty.Party.PostalAddress.StreetName)
		assert.Equal(t, "München", *doc.AccountingCustomerParty.Party.PostalAddress.CityName)
		assert.Equal(t, "80939", *doc.AccountingCustomerParty.Party.PostalAddress.PostalZone)
		assert.Equal(t, "DE", doc.AccountingCustomerParty.Party.PostalAddress.Country.IdentificationCode)

		assert.Equal(t, "0088", *doc.AccountingCustomerParty.Party.PartyIdentification[0].ID.SchemeID)
		assert.Equal(t, "1234567890128", doc.AccountingCustomerParty.Party.PartyIdentification[0].ID.Value)
	})

	t.Run("invoice-complete.json", func(t *testing.T) {
		doc := testInvoiceFrom(t, "invoice-complete.json")

		assert.Equal(t, "inbox@example.com", doc.AccountingSupplierParty.Party.EndpointID.Value)
		assert.Equal(t, "EM", doc.AccountingSupplierParty.Party.EndpointID.SchemeID)
	})

	t.Run("oioubl21 party uses real data without fallbacks", func(t *testing.T) {
		doc := testInvoiceFrom(t, "oioubl21/invoice-bare.json")

		supplier := doc.AccountingSupplierParty.Party
		customer := doc.AccountingCustomerParty.Party

		// EndpointID is sourced from the party inbox (no CVR fallback).
		require.NotNil(t, supplier.EndpointID)
		assert.Equal(t, "DK:CVR", supplier.EndpointID.SchemeID)
		assert.Equal(t, "DK37990485", supplier.EndpointID.Value)

		require.NotNil(t, customer.EndpointID)
		assert.Equal(t, "DK:CVR", customer.EndpointID.SchemeID)
		assert.Equal(t, "DK47458714", customer.EndpointID.Value)

		// PartyName falls back to RegistrationName when no alias is present.
		require.NotNil(t, supplier.PartyName)
		assert.Equal(t, "Worksome Aps", supplier.PartyName.Name)

		require.NotNil(t, customer.PartyName)
		assert.Equal(t, "Lego System A/S", customer.PartyName.Name)

		// Contact/ID is sourced from the customer's person identity, not fabricated.
		require.NotNil(t, customer.Contact)
		require.NotNil(t, customer.Contact.ID)
		assert.Equal(t, "C-001", *customer.Contact.ID)

		// The supplier has no contact person, so no Contact is emitted; OIOUBL
		// does not require a supplier Contact.
		assert.Nil(t, supplier.Contact)
	})

	t.Run("identity scopes map to legal and tax identifiers", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Supplier.Identities = []*org.Identity{
			{
				Scope: org.IdentityScopeLegal,
				Code:  cbc.Code("99887766"),
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					iso.ExtKeySchemeID: cbc.Code("0184"),
				}),
			},
			{
				Scope: org.IdentityScopeTax,
				Type:  cbc.Code("VAT"),
				Code:  cbc.Code("DK99887766"),
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					iso.ExtKeySchemeID: cbc.Code("0198"),
				}),
			},
			{
				Code: cbc.Code("1234567890128"),
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					iso.ExtKeySchemeID: cbc.Code("0088"),
				}),
			},
		}

		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)
		require.NotNil(t, doc.AccountingSupplierParty)
		require.NotNil(t, doc.AccountingSupplierParty.Party)
		require.NotNil(t, doc.AccountingSupplierParty.Party.PartyLegalEntity)
		require.NotNil(t, doc.AccountingSupplierParty.Party.PartyLegalEntity.CompanyID)
		require.NotNil(t, doc.AccountingSupplierParty.Party.PartyLegalEntity.CompanyID.SchemeID)
		assert.Equal(t, "99887766", doc.AccountingSupplierParty.Party.PartyLegalEntity.CompanyID.Value)
		assert.Equal(t, "0184", *doc.AccountingSupplierParty.Party.PartyLegalEntity.CompanyID.SchemeID)

		foundTaxIdentity := false
		for _, pts := range doc.AccountingSupplierParty.Party.PartyTaxScheme {
			if pts.CompanyID != nil && pts.CompanyID.Value == "DK99887766" {
				foundTaxIdentity = true
				require.NotNil(t, pts.CompanyID.SchemeID)
				assert.Equal(t, "0198", *pts.CompanyID.SchemeID)
				break
			}
		}
		assert.True(t, foundTaxIdentity, "expected tax-scope identity in PartyTaxScheme")

		require.NotEmpty(t, doc.AccountingSupplierParty.Party.PartyIdentification)
		require.NotNil(t, doc.AccountingSupplierParty.Party.PartyIdentification[0].ID)
		assert.Equal(t, "1234567890128", doc.AccountingSupplierParty.Party.PartyIdentification[0].ID.Value)
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
}
