package gtou

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDelivery(t *testing.T) {
	t.Run("invoice-without-buyers-tax-id.json", func(t *testing.T) {
		env, err := LoadTestEnvelope("invoice-without-buyers-tax-id.json")
		require.NoError(t, err)

		inv := env.Extract().(*bill.Invoice)

		conversor := NewConversor()
		err = conversor.newDocument(inv)
		require.NoError(t, err)

		doc := conversor.GetDocument()
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

}
