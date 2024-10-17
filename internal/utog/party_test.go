package ubl_test

import (
	"testing"

	utog "github.com/invopop/gobl.ubl/internal/utog"
	"github.com/invopop/gobl.ubl/test"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define tests for the ParseParty function
func TestParseUtoGParty(t *testing.T) {
	doc, err := test.LoadTestXMLDoc("UBL_example1.xml")
	require.NoError(t, err)

	seller := utog.ParseUtoGParty(&doc.AccountingSupplierParty.Party)
	require.NotNil(t, seller)

	assert.Equal(t, "Mustermann GmbH", seller.Name)
	assert.Equal(t, l10n.TaxCountryCode("DE"), seller.TaxID.Country)
	assert.Equal(t, cbc.Code("123456789"), seller.TaxID.Code)

	buyer := utog.ParseUtoGParty(&doc.AccountingCustomerParty.Party)
	require.NotNil(t, buyer)

	assert.Equal(t, "Beispiel AG", buyer.Name)
	assert.Equal(t, "Hauptstraße 1", buyer.Addresses[0].Street)
	assert.Equal(t, "Musterstadt", buyer.Addresses[0].Locality)
	assert.Equal(t, "12345", buyer.Addresses[0].Code)
	assert.Equal(t, l10n.ISOCountryCode("DE"), buyer.Addresses[0].Country)
}
