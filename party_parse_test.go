package ubl_test

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
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

// TestParseSupplierIdentifiers checks that the supplier keeps its own BT-31
// VAT number and BT-34 endpoint when the party carries extra identifiers and
// the invoice names a BG-11 tax representative, as the French "assujetti
// unique" (VAT group) invoices do.
func TestParseSupplierIdentifiers(t *testing.T) {
	tests := []struct {
		name        string
		file        string
		taxID       cbc.Code
		inboxScheme cbc.Code
		inboxCode   cbc.Code
		groupSIREN  cbc.Code // BT-29d, the 0231 VAT group identifier
		repName     string   // BT-62, empty when there is no tax representative
		repTaxID    cbc.Code // BT-63
	}{
		{
			name:        "ordinary identifiers",
			file:        "france-extended/b2g-invoice.xml",
			taxID:       "53341200068",
			inboxScheme: "0225",
			inboxCode:   "341200068",
		},
		{
			name:        "assujetti unique",
			file:        "france-extended/b2g-assujetti-unique.xml",
			taxID:       "53341200068",
			inboxScheme: "0225",
			inboxCode:   "341200068",
			groupSIREN:  "123456789",
			repName:     "Fournisseur 34120006871491 ASSUJETTI UNIQUE",
			repTaxID:    "00123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := parseXMLInvoice(t, tt.file)

			inv, ok := e.Extract().(*bill.Invoice)
			require.True(t, ok)

			supplier := inv.Supplier
			require.NotNil(t, supplier)
			require.NotNil(t, supplier.TaxID)
			assert.Equal(t, l10n.TaxCountryCode("FR"), supplier.TaxID.Country)
			assert.Equal(t, tt.taxID, supplier.TaxID.Code)

			require.Len(t, supplier.Inboxes, 1)
			assert.Equal(t, tt.inboxScheme, supplier.Inboxes[0].Scheme)
			assert.Equal(t, tt.inboxCode, supplier.Inboxes[0].Code)

			assert.Equal(t, tt.groupSIREN, identityWithScheme(supplier, "0231"))

			var rep *org.Party
			if inv.Ordering != nil {
				rep = inv.Ordering.Seller
			}
			if tt.repName == "" {
				assert.Nil(t, rep)
				return
			}
			require.NotNil(t, rep)
			assert.Equal(t, tt.repName, rep.Name)
			require.NotNil(t, rep.TaxID)
			assert.Equal(t, tt.repTaxID, rep.TaxID.Code)
		})
	}
}

// identityWithScheme returns the code of the party identity issued under the
// given ISO 6523 scheme, or an empty code when there is none.
func identityWithScheme(party *org.Party, scheme cbc.Code) cbc.Code {
	for _, id := range party.Identities {
		if id.Ext.Get(iso.ExtKeySchemeID) == scheme {
			return id.Code
		}
	}
	return cbc.CodeEmpty
}
