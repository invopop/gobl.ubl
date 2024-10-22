package utog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUtoGCharges(t *testing.T) {
	// Invoice with Charge
	t.Run("ubl-example2.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("ubl-example2.xml")
		require.NoError(t, err)
		c := NewConversor()
		inv, err := c.NewInvoice(doc)
		require.NoError(t, err)

		charges := inv.Charges
		discounts := inv.Discounts

		// Check if there's a charge in the parsed output
		assert.NotNil(t, charges)
		assert.Nil(t, discounts)
	})

}
