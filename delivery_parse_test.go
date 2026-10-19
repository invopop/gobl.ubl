package ubl_test

import (
	"bytes"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDelivery(t *testing.T) {
	t.Run("ubl-example4.xml", func(t *testing.T) {
		e := parseXMLInvoice(t, "en16931/ubl-example4.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.NotNil(t, inv.Delivery)
		assert.Equal(t, "2013-04-15", inv.Delivery.Date.String())
		assert.NotNil(t, inv.Delivery.Receiver)
		assert.NotNil(t, inv.Delivery.Receiver.Addresses)
		assert.Equal(t, "Deliverystreet", inv.Delivery.Receiver.Addresses[0].Street)
		assert.Equal(t, "Deliverycity", inv.Delivery.Receiver.Addresses[0].Locality)
		assert.Equal(t, cbc.Code("9000"), inv.Delivery.Receiver.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("DK"), inv.Delivery.Receiver.Addresses[0].Country)
	})

	t.Run("ubl-example5.xml", func(t *testing.T) {
		e := parseXMLInvoice(t, "en16931/ubl-example5.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		assert.NotNil(t, inv.Delivery)
		assert.Equal(t, "2013-04-15", inv.Delivery.Date.String())
		assert.NotNil(t, inv.Delivery.Receiver)
		assert.NotNil(t, inv.Delivery.Receiver.Addresses)
		assert.Equal(t, "Deliverystreet", inv.Delivery.Receiver.Addresses[0].Street)
		assert.Equal(t, "Deliverycity", inv.Delivery.Receiver.Addresses[0].Locality)
		assert.Equal(t, "Gate 15", inv.Delivery.Receiver.Addresses[0].StreetExtra)
		assert.Equal(t, "Jutland", inv.Delivery.Receiver.Addresses[0].Region)
		assert.Equal(t, cbc.Code("9000"), inv.Delivery.Receiver.Addresses[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("DK"), inv.Delivery.Receiver.Addresses[0].Country)
	})

	t.Run("location ID scheme is kept as the ISO 6523 extension", func(t *testing.T) {
		e := parseXMLInvoice(t, "base-example.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.NotNil(t, inv.Delivery)
		require.Len(t, inv.Delivery.Identities, 1)
		id := inv.Delivery.Identities[0]
		assert.Equal(t, cbc.Code("9483759475923478"), id.Code)
		assert.Equal(t, cbc.Code("0088"), id.Ext.Get(iso.ExtKeySchemeID))
		assert.Empty(t, id.Label)

		// The scheme must survive a round trip back to UBL (BT-71-1).
		out, err := ubl.ExportInvoice(e, ubl.WithFormat(ubl.FormatEN16931))
		require.NoError(t, err)
		require.Len(t, out.Delivery, 1)
		loc := out.Delivery[0].DeliveryLocation
		require.NotNil(t, loc)
		require.NotNil(t, loc.ID)
		require.NotNil(t, loc.ID.SchemeID)
		assert.Equal(t, "0088", *loc.ID.SchemeID)
	})

	t.Run("empty location ID scheme is skipped", func(t *testing.T) {
		data, err := testLoadXML("base-example.xml")
		require.NoError(t, err)
		data = bytes.Replace(data,
			[]byte(`<cbc:ID schemeID="0088">9483759475923478</cbc:ID>`),
			[]byte(`<cbc:ID schemeID="">9483759475923478</cbc:ID>`), 1)
		require.Contains(t, string(data), `schemeID="">9483759475923478`)

		doc, err := ubl.Decode(data)
		require.NoError(t, err)
		e, err := ubl.Import(doc)
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)
		require.NotNil(t, inv.Delivery)
		require.Len(t, inv.Delivery.Identities, 1)
		id := inv.Delivery.Identities[0]
		assert.Equal(t, cbc.Code("9483759475923478"), id.Code)
		assert.Empty(t, id.Ext)
		assert.Empty(t, id.Label)

		out, err := ubl.ExportInvoice(e, ubl.WithFormat(ubl.FormatEN16931))
		require.NoError(t, err)
		require.Len(t, out.Delivery, 1)
		require.NotNil(t, out.Delivery[0].DeliveryLocation)
		require.NotNil(t, out.Delivery[0].DeliveryLocation.ID)
		assert.Nil(t, out.Delivery[0].DeliveryLocation.ID.SchemeID)
	})
}
