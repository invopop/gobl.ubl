package ubl_test

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define tests for the ParseParty function
func TestParseParty(t *testing.T) {
	t.Run("ubl-example2.xml", func(t *testing.T) {
		e := parseXMLInvoice(t, "en16931/ubl-example2.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		supplier := inv.Supplier
		require.NotNil(t, supplier)
		assert.Equal(t, "Salescompany ltd.", supplier.Name)
		assert.Equal(t, cbc.Code("123456789MVA"), supplier.TaxID.Code)
		assert.Equal(t, l10n.TaxCountryCode("NO"), supplier.TaxID.Country)
		require.Len(t, supplier.Identities, 2)
		assert.Equal(t, cbc.Code("123456789"), supplier.Identities[0].Code)
		assert.Equal(t, "0088", supplier.Identities[1].Ext.Get(iso.ExtKeySchemeID).String())
		assert.Equal(t, cbc.Code("1238764941386"), supplier.Identities[1].Code)

		assert.Equal(t, "Main street 34", supplier.Addresses[0].Street)
		assert.Equal(t, "Suite 123", supplier.Addresses[0].StreetExtra)
		assert.Equal(t, "Big city", supplier.Addresses[0].Locality)
		assert.Equal(t, "RegionA", supplier.Addresses[0].Region)
		assert.Equal(t, cbc.Code("303"), supplier.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("NO"), supplier.Addresses[0].Country)

		require.Len(t, supplier.People, 1)
		assert.Equal(t, "Antonio Salesmacher", supplier.People[0].Name.Given)
		assert.Equal(t, "antonio@salescompany.no", supplier.Emails[0].Address)
		assert.Equal(t, "46211230", supplier.Telephones[0].Number)
		assert.Equal(t, "seller@email.de", supplier.Inboxes[0].Email)
		assert.Equal(t, "", supplier.Inboxes[0].Scheme.String())

		// BG-11 tax representative, the party liable for the tax.
		seller := inv.Ordering.Seller
		require.NotNil(t, seller)
		assert.Equal(t, "Tax handling company AS", seller.Name)
		assert.Equal(t, cbc.Code("967611265MVA"), seller.TaxID.Code)
		assert.Equal(t, l10n.TaxCountryCode("NO"), seller.TaxID.Country)
		assert.Equal(t, "Regent street", seller.Addresses[0].Street)
		assert.Equal(t, "Newtown", seller.Addresses[0].Locality)
		assert.Equal(t, "Front door", seller.Addresses[0].StreetExtra)
		assert.Equal(t, "RegionC", seller.Addresses[0].Region)
		assert.Equal(t, cbc.Code("202"), seller.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("NO"), seller.Addresses[0].Country)

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
		assert.Equal(t, cbc.Code("987654321"), customer.Identities[0].Code)
		assert.Equal(t, "0088", customer.Identities[1].Ext.Get(iso.ExtKeySchemeID).String())
		assert.Equal(t, cbc.Code("3456789012098"), customer.Identities[1].Code)

		assert.Equal(t, "John Doe", customer.People[0].Name.Given)
		assert.Equal(t, "5121230", customer.Telephones[0].Number)
		assert.Equal(t, "john@buyercompany.no", customer.Emails[0].Address)
	})

	t.Run("ubl-example3.xml", func(t *testing.T) {
		e := parseXMLInvoice(t, "en16931/ubl-example3.xml")

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
		assert.Equal(t, cbc.Code("DK16356706"), supplier.Identities[0].Code)
		assert.Equal(t, "0088", supplier.Identities[1].Ext.Get(iso.ExtKeySchemeID).String())
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

	t.Run("invoice-peppol.xml", func(t *testing.T) {
		e := parseXMLInvoice(t, "peppol/invoice-peppol.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		supplier := inv.Supplier
		require.NotNil(t, supplier)
		assert.Equal(t, "Acme Corporation", supplier.Name)
		assert.Equal(t, cbc.Code("0000000000"), supplier.TaxID.Code)
		assert.Equal(t, l10n.TaxCountryCode("BE"), supplier.TaxID.Country)
		assert.Equal(t, "Acme Street 4001", supplier.Addresses[0].Street)
		assert.Equal(t, "Acme Town", supplier.Addresses[0].Locality)
		assert.Equal(t, cbc.Code("123 45"), supplier.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("BE"), supplier.Addresses[0].Country)

		assert.Equal(t, "0151", supplier.Inboxes[0].Scheme.String())
		assert.Equal(t, "99100100100", supplier.Inboxes[0].Code.String())

	})
}
