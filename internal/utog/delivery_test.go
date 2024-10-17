package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl.ubl/test"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/l10n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUtoGDelivery(t *testing.T) {
	t.Run("UBL_example1.xml", func(t *testing.T) {
		doc, err := test.LoadTestXMLDoc("UBL_example1.xml")
		require.NoError(t, err)

		goblEnv, err := ubl.NewGOBLFromUBL(doc)
		require.NoError(t, err)

		invoice, ok := goblEnv.Extract().(*bill.Invoice)
		require.True(t, ok, "Document should be an invoice")

		require.NotNil(t, invoice.Delivery, "Delivery should not be nil for Example 1")
		require.NotNil(t, invoice.Delivery.Receiver, "Delivery receiver should not be nil")

		assert.NotEmpty(t, invoice.Delivery.Receiver.Addresses, "Delivery receiver addresses should not be empty")
		assert.Equal(t, "1234", invoice.Delivery.Receiver.Addresses[0].Code, "Delivery receiver post code should match")
		assert.Equal(t, "Delivery Street 1", invoice.Delivery.Receiver.Addresses[0].Street, "Delivery receiver street should match")
		assert.Equal(t, "DeliveryCity", invoice.Delivery.Receiver.Addresses[0].Locality, "Delivery receiver city should match")
		assert.Equal(t, l10n.ISOCountryCode("NL"), invoice.Delivery.Receiver.Addresses[0].Country, "Delivery receiver country should match")
		assert.Equal(t, "2023-02-01", invoice.Delivery.Date.String(), "Delivery date should match")
	})

	t.Run("UBL_example2.xml", func(t *testing.T) {
		doc, err := test.LoadTestXMLDoc("UBL_example2.xml")
		require.NoError(t, err)

		goblEnv, err := ubl.NewGOBLFromUBL(doc)
		require.NoError(t, err)

		invoice, ok := goblEnv.Extract().(*bill.Invoice)
		require.True(t, ok, "Document should be an invoice")

		require.NotNil(t, invoice.Delivery, "Delivery should not be nil for Example 2")
		require.NotNil(t, invoice.Delivery.Receiver, "Delivery receiver should not be nil")
		assert.NotEmpty(t, invoice.Delivery.Receiver.Addresses, "Delivery receiver addresses should not be empty")
		assert.Equal(t, "Delivery Avenue 2", invoice.Delivery.Receiver.Addresses[0].Street, "Delivery receiver street should match")
		assert.Equal(t, "5678", invoice.Delivery.Receiver.Addresses[0].Code, "Delivery receiver post code should match")
		assert.Equal(t, "ReceiverTown", invoice.Delivery.Receiver.Addresses[0].Locality, "Delivery receiver city should match")
		assert.Equal(t, l10n.ISOCountryCode("BE"), invoice.Delivery.Receiver.Addresses[0].Country, "Delivery receiver country should match")
	})
}
