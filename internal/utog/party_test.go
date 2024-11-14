package utog

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define tests for the ParseParty function
func TestGetParty(t *testing.T) {
	t.Run("ubl-example2.xml", func(t *testing.T) {
		e, err := newDocumentFrom("ubl-example2.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		supplier := inv.Supplier
		require.NotNil(t, supplier)
		assert.Equal(t, "Tax handling company AS", supplier.Name)
		assert.Equal(t, cbc.Code("967611265MVA"), supplier.TaxID.Code)
		assert.Equal(t, l10n.TaxCountryCode("NO"), supplier.TaxID.Country)
		assert.Equal(t, "Regent street", supplier.Addresses[0].Street)
		assert.Equal(t, "Newtown", supplier.Addresses[0].Locality)
		assert.Equal(t, "Front door", supplier.Addresses[0].StreetExtra)
		assert.Equal(t, "RegionC", supplier.Addresses[0].Region)
		assert.Equal(t, cbc.Code("202"), supplier.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("NO"), supplier.Addresses[0].Country)

		seller := inv.Ordering.Seller
		require.NotNil(t, seller)
		assert.Equal(t, "Salescompany ltd.", seller.Name)
		assert.Equal(t, cbc.Code("123456789MVA"), seller.TaxID.Code)
		assert.Equal(t, l10n.TaxCountryCode("NO"), seller.TaxID.Country)
		require.Len(t, seller.Identities, 2)
		assert.Equal(t, "CompanyID", seller.Identities[0].Label)
		assert.Equal(t, cbc.Code("123456789"), seller.Identities[0].Code)
		assert.Equal(t, "0088", seller.Identities[1].Label)
		assert.Equal(t, cbc.Code("1238764941386"), seller.Identities[1].Code)

		assert.Equal(t, "Main street 34", seller.Addresses[0].Street)
		assert.Equal(t, "Suite 123", seller.Addresses[0].StreetExtra)
		assert.Equal(t, "Big city", seller.Addresses[0].Locality)
		assert.Equal(t, "RegionA", seller.Addresses[0].Region)
		assert.Equal(t, cbc.Code("303"), seller.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("NO"), seller.Addresses[0].Country)

		require.Len(t, seller.People, 1)
		assert.Equal(t, "Antonio Salesmacher", seller.People[0].Name.Given)
		assert.Equal(t, "antonio@salescompany.no", seller.Emails[0].Address)
		assert.Equal(t, "46211230", seller.Telephones[0].Number)

		customer := inv.Customer
		require.NotNil(t, customer)
		assert.Equal(t, "The Buyercompany", customer.Name)
		assert.Equal(t, cbc.Code("987654321MVA"), customer.TaxID.Code)
		assert.Equal(t, l10n.TaxCountryCode("NO"), customer.TaxID.Country)
		assert.Equal(t, "Anystreet 8", customer.Addresses[0].Street)
		assert.Equal(t, "Back door", customer.Addresses[0].StreetExtra)
		assert.Equal(t, "Anytown", customer.Addresses[0].Locality)
		assert.Equal(t, "RegionB", customer.Addresses[0].Region)
		assert.Equal(t, cbc.Code("101"), customer.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("NO"), customer.Addresses[0].Country)

		require.Len(t, customer.Identities, 2)
		assert.Equal(t, "CompanyID", customer.Identities[0].Label)
		assert.Equal(t, cbc.Code("987654321"), customer.Identities[0].Code)
		assert.Equal(t, "0088", customer.Identities[1].Label)
		assert.Equal(t, cbc.Code("3456789012098"), customer.Identities[1].Code)

		assert.Equal(t, "John Doe", customer.People[0].Name.Given)
		assert.Equal(t, "5121230", customer.Telephones[0].Number)
		assert.Equal(t, "john@buyercompany.no", customer.Emails[0].Address)
	})

	t.Run("ubl-example3.xml", func(t *testing.T) {
		e, err := newDocumentFrom("ubl-example3.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		supplier := inv.Supplier
		require.NotNil(t, supplier)
		assert.Equal(t, "SubscriptionSeller", supplier.Name)
		assert.Equal(t, cbc.Code("16356706"), supplier.TaxID.Code)
		assert.Equal(t, l10n.TaxCountryCode("DK"), supplier.TaxID.Country)
		assert.Equal(t, "Main street 2, Building 4", supplier.Addresses[0].Street)
		assert.Equal(t, "Big city", supplier.Addresses[0].Locality)
		assert.Equal(t, cbc.Code("54321"), supplier.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("DK"), supplier.Addresses[0].Country)

		assert.Equal(t, "antonio@SubscriptionsSeller.dk", supplier.Emails[0].Address)
		require.Len(t, supplier.Identities, 2)
		assert.Equal(t, "CompanyID", supplier.Identities[0].Label)
		assert.Equal(t, cbc.Code("DK16356706"), supplier.Identities[0].Code)
		assert.Equal(t, "0088", supplier.Identities[1].Label)
		assert.Equal(t, cbc.Code("1238764941386"), supplier.Identities[1].Code)

		customer := inv.Customer
		require.NotNil(t, customer)
		assert.Equal(t, "Buyercompany ltd", customer.Name)
		assert.Equal(t, cbc.Code("NO987654321MVA"), customer.TaxID.Code)
		assert.Equal(t, l10n.TaxCountryCode("DK"), customer.TaxID.Country)
		assert.Equal(t, "Anystreet, Building 1", customer.Addresses[0].Street)
		assert.Equal(t, "Anytown", customer.Addresses[0].Locality)
		assert.Equal(t, cbc.Code("101"), customer.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("DK"), customer.Addresses[0].Country)
	})
}
