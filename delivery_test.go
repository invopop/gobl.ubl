package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDelivery(t *testing.T) {
	t.Run("invoice-without-buyers-tax-id.json", func(t *testing.T) {
		doc, err := testInvoiceFrom("invoice-without-buyers-tax-id.json")
		require.NoError(t, err)

		assert.NotNil(t, doc.Delivery)
		assert.Len(t, doc.Delivery, 1)
		assert.Equal(t, "2024-02-10", *doc.Delivery[0].ActualDeliveryDate)
		assert.NotNil(t, doc.Delivery[0].DeliveryLocation)
		assert.NotNil(t, doc.Delivery[0].DeliveryLocation.Address)
		assert.Equal(t, "Deliverystreet 2", *doc.Delivery[0].DeliveryLocation.Address.StreetName)
		assert.Equal(t, "Side door", *doc.Delivery[0].DeliveryLocation.Address.AdditionalStreetName)
		assert.Equal(t, "DeliveryCity", *doc.Delivery[0].DeliveryLocation.Address.CityName)
		assert.Equal(t, "523427", *doc.Delivery[0].DeliveryLocation.Address.PostalZone)
		assert.Equal(t, "RegionD", *doc.Delivery[0].DeliveryLocation.Address.CountrySubentity)
		assert.Equal(t, "NO", doc.Delivery[0].DeliveryLocation.Address.Country.IdentificationCode)

	})

	t.Run("delivery with no receiver", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-without-buyers-tax-id.json")
		require.NoError(t, err)

		inv := env.Extract().(*bill.Invoice)

		inv.Delivery.Receiver = nil

		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)

		assert.NotNil(t, doc.Delivery)
		assert.Len(t, doc.Delivery, 1)
		assert.Equal(t, "2024-02-10", *doc.Delivery[0].ActualDeliveryDate)
		assert.Nil(t, doc.Delivery[0].DeliveryLocation)
	})

	t.Run("delivery with no date", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-without-buyers-tax-id.json")
		require.NoError(t, err)

		inv := env.Extract().(*bill.Invoice)

		inv.Delivery.Date = nil

		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)

		assert.NotNil(t, doc.Delivery)
		assert.Len(t, doc.Delivery, 1)
		assert.Nil(t, doc.Delivery[0].ActualDeliveryDate)
		assert.NotNil(t, doc.Delivery[0].DeliveryLocation)
		assert.NotNil(t, doc.Delivery[0].DeliveryLocation.Address)
		assert.Equal(t, "Deliverystreet 2", *doc.Delivery[0].DeliveryLocation.Address.StreetName)
		assert.Equal(t, "Side door", *doc.Delivery[0].DeliveryLocation.Address.AdditionalStreetName)
		assert.Equal(t, "DeliveryCity", *doc.Delivery[0].DeliveryLocation.Address.CityName)
		assert.Equal(t, "523427", *doc.Delivery[0].DeliveryLocation.Address.PostalZone)
		assert.Equal(t, "RegionD", *doc.Delivery[0].DeliveryLocation.Address.CountrySubentity)
		assert.Equal(t, "NO", doc.Delivery[0].DeliveryLocation.Address.Country.IdentificationCode)
	})

	t.Run("delivery with no receiver and no date", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-without-buyers-tax-id.json")
		require.NoError(t, err)

		inv := env.Extract().(*bill.Invoice)

		inv.Delivery.Receiver = nil
		inv.Delivery.Date = nil

		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)

		assert.Nil(t, doc.Delivery)
	})

	t.Run("nil delivery", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-without-buyers-tax-id.json")
		require.NoError(t, err)

		inv := env.Extract().(*bill.Invoice)

		inv.Delivery = nil

		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)

		assert.Nil(t, doc.Delivery)
	})
}

func TestNewDeliveryTerms(t *testing.T) {
	t.Run("delivery with identities", func(t *testing.T) {
		doc, err := testInvoiceFrom("invoice-without-buyers-tax-id.json")
		require.NoError(t, err)

		assert.NotNil(t, doc.DeliveryTerms)
		assert.Equal(t, "6754238987643", doc.DeliveryTerms.ID)
	})

	t.Run("delivery without identities", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-without-buyers-tax-id.json")
		require.NoError(t, err)

		inv := env.Extract().(*bill.Invoice)

		inv.Delivery.Identities = nil

		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)

		assert.Nil(t, doc.DeliveryTerms)
	})

	t.Run("nil delivery", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-without-buyers-tax-id.json")
		require.NoError(t, err)

		inv := env.Extract().(*bill.Invoice)

		inv.Delivery = nil

		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)

		assert.Nil(t, doc.DeliveryTerms)
	})
}
