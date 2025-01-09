package gtou

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewParty(t *testing.T) {
	t.Run("invoice-de-de.json", func(t *testing.T) {
		doc, err := newDocumentFrom("invoice-de-de.json")
		require.NoError(t, err)

		assert.Equal(t, "DE111111125", *doc.AccountingSupplierParty.Party.PartyTaxScheme[0].CompanyID)
		assert.Equal(t, "Provide One GmbH", *doc.AccountingSupplierParty.Party.PartyLegalEntity.RegistrationName)
		assert.Equal(t, "+49100200300", *doc.AccountingSupplierParty.Party.Contact.Telephone)
		assert.Equal(t, "billing@example.com", *doc.AccountingSupplierParty.Party.Contact.ElectronicMail)

		assert.Equal(t, "Dietmar-Hopp-Allee 16", *doc.AccountingSupplierParty.Party.PostalAddress.StreetName)
		assert.Equal(t, "Walldorf", *doc.AccountingSupplierParty.Party.PostalAddress.CityName)
		assert.Equal(t, "69190", *doc.AccountingSupplierParty.Party.PostalAddress.PostalZone)
		assert.Equal(t, "DE", doc.AccountingSupplierParty.Party.PostalAddress.Country.IdentificationCode)

		assert.Equal(t, "DE282741168", *doc.AccountingCustomerParty.Party.PartyTaxScheme[0].CompanyID)
		assert.Equal(t, "Sample Consumer", *doc.AccountingCustomerParty.Party.PartyLegalEntity.RegistrationName)
		assert.Equal(t, "email@sample.com", *doc.AccountingCustomerParty.Party.Contact.ElectronicMail)

		assert.Equal(t, "Werner-Heisenberg-Allee 25", *doc.AccountingCustomerParty.Party.PostalAddress.StreetName)
		assert.Equal(t, "München", *doc.AccountingCustomerParty.Party.PostalAddress.CityName)
		assert.Equal(t, "80939", *doc.AccountingCustomerParty.Party.PostalAddress.PostalZone)
		assert.Equal(t, "DE", doc.AccountingCustomerParty.Party.PostalAddress.Country.IdentificationCode)

		assert.Equal(t, "0088", *doc.AccountingCustomerParty.Party.PartyIdentification.ID.SchemeID)
		assert.Equal(t, "1234567890128", doc.AccountingCustomerParty.Party.PartyIdentification.ID.Value)
	})

	t.Run("invoice-complete.json", func(t *testing.T) {
		doc, err := newDocumentFrom("invoice-complete.json")
		require.NoError(t, err)

		assert.Equal(t, "inbox@example.com", doc.AccountingSupplierParty.Party.EndpointID.Value)
		assert.Equal(t, "EM", doc.AccountingSupplierParty.Party.EndpointID.SchemeID)
	})

}
