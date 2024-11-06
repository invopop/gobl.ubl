package utog

import (
	"testing"

	"github.com/invopop/gobl/l10n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDelivery(t *testing.T) {
	t.Run("ubl-example4.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("ubl-example4.xml")
		require.NoError(t, err)

		converter := NewConverter()
		err = converter.NewInvoice(doc)
		require.NoError(t, err)

		inv := converter.GetInvoice()
		assert.NotNil(t, inv.Delivery)
		assert.Equal(t, "2013-04-15", inv.Delivery.Date.String())
		assert.NotNil(t, inv.Delivery.Receiver)
		assert.NotNil(t, inv.Delivery.Receiver.Addresses)
		assert.Equal(t, "Deliverystreet", inv.Delivery.Receiver.Addresses[0].Street)
		assert.Equal(t, "Deliverycity", inv.Delivery.Receiver.Addresses[0].Locality)
		assert.Equal(t, "9000", inv.Delivery.Receiver.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("DK"), inv.Delivery.Receiver.Addresses[0].Country)
	})

	t.Run("ubl-example5.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("ubl-example5.xml")
		require.NoError(t, err)

		converter := NewConverter()
		err = converter.NewInvoice(doc)
		require.NoError(t, err)

		inv := converter.GetInvoice()
		assert.NotNil(t, inv.Delivery)
		assert.Equal(t, "2013-04-15", inv.Delivery.Date.String())
		assert.NotNil(t, inv.Delivery.Receiver)
		assert.NotNil(t, inv.Delivery.Receiver.Addresses)
		assert.Equal(t, "Deliverystreet", inv.Delivery.Receiver.Addresses[0].Street)
		assert.Equal(t, "Deliverycity", inv.Delivery.Receiver.Addresses[0].Locality)
		assert.Equal(t, "Gate 15", inv.Delivery.Receiver.Addresses[0].StreetExtra)
		assert.Equal(t, "Jutland", inv.Delivery.Receiver.Addresses[0].Region)
		assert.Equal(t, "9000", inv.Delivery.Receiver.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("DK"), inv.Delivery.Receiver.Addresses[0].Country)
	})
}
