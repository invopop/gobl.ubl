package utog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Define tests for the ParseParty function
func TestParseUtoGParty(t *testing.T) {
	t.Run("UBL_example1.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("UBL_example1.xml")
		require.NoError(t, err)
		conversor := NewConversor()
		inv, err := conversor.NewInvoice(doc)
		require.NoError(t, err)

		supplier := inv.Supplier
		require.NotNil(t, supplier)

		customer := inv.Customer
		require.NotNil(t, customer)
	})
}
