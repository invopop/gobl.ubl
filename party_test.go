package ubl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewParty(t *testing.T) {
	t.Run("invoice-complete.json", func(t *testing.T) {
		doc, err := testInvoiceFrom("invoice-complete.json")
		require.NoError(t, err)

		assert.Equal(t, "inbox@example.com", doc.AccountingSupplierParty.Party.EndpointID.Value)
		assert.Equal(t, "EM", doc.AccountingSupplierParty.Party.EndpointID.SchemeID)
	})

	t.Run("invoice-with-logos.json", func(t *testing.T) {
		doc, err := testInvoiceFrom("invoice-with-logos.json")
		require.NoError(t, err)

		// Verify supplier logo is mapped to LogoReferenceID
		require.NotNil(t, doc.AccountingSupplierParty.Party.LogoReferenceID)
		assert.Equal(t, "https://www.example.com/images/logo.png", *doc.AccountingSupplierParty.Party.LogoReferenceID)

		// Verify customer logo is mapped to LogoReferenceID
		require.NotNil(t, doc.AccountingCustomerParty.Party.LogoReferenceID)
		assert.Equal(t, "https://www.customer-example.com/logo.svg", *doc.AccountingCustomerParty.Party.LogoReferenceID)
	})

}
