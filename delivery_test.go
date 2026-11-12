package ubl_test

import (
	"strings"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDelivery(t *testing.T) {
	t.Run("invoice-without-buyers-tax-id.json", func(t *testing.T) {
		doc := testInvoiceFrom(t, "invoice-without-buyers-tax-id.json")

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
		env := loadTestEnvelope(t, "invoice-without-buyers-tax-id.json")

		inv := env.Extract().(*bill.Invoice)

		inv.Delivery.Receiver = nil

		doc, err := ubl.ExportInvoice(env)
		require.NoError(t, err)

		assert.NotNil(t, doc.Delivery)
		assert.Len(t, doc.Delivery, 1)
		assert.Equal(t, "2024-02-10", *doc.Delivery[0].ActualDeliveryDate)
		assert.Nil(t, doc.Delivery[0].DeliveryLocation)
	})

	t.Run("delivery location identity with iso scheme id propagates to SchemeID", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-without-buyers-tax-id.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Delivery.Identities = []*org.Identity{
			{
				Code: "6754238987643",
				Ext:  tax.ExtensionsOf(cbc.CodeMap{iso.ExtKeySchemeID: "0088"}),
			},
		}

		require.NoError(t, env.Calculate())
		doc, err := ubl.ExportInvoice(env)
		require.NoError(t, err)

		require.NotNil(t, doc.Delivery[0].DeliveryLocation)
		id := doc.Delivery[0].DeliveryLocation.ID
		require.NotNil(t, id)
		assert.Equal(t, "6754238987643", id.Value)
		require.NotNil(t, id.SchemeID)
		assert.Equal(t, "0088", *id.SchemeID)
	})

	t.Run("delivery location identity without iso scheme id leaves SchemeID unset", func(t *testing.T) {
		doc := testInvoiceFrom(t, "invoice-without-buyers-tax-id.json")

		id := doc.Delivery[0].DeliveryLocation.ID
		require.NotNil(t, id)
		assert.Nil(t, id.SchemeID)
	})

	t.Run("delivery with no date", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-without-buyers-tax-id.json")

		inv := env.Extract().(*bill.Invoice)

		inv.Delivery.Date = nil

		doc, err := ubl.ExportInvoice(env)
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

}

func TestDeliveryTermsEncoding(t *testing.T) {
	doc := testInvoiceFrom(t, "invoice-without-buyers-tax-id.json")
	require.Len(t, doc.Delivery, 1)

	t.Run("omitted by default", func(t *testing.T) {
		data, err := ubl.Encode(doc)
		require.NoError(t, err)
		assert.NotContains(t, string(data), "DeliveryTerms")
	})

	t.Run("all terms written after the delivery party and read back", func(t *testing.T) {
		doc.Delivery[0].DeliveryTerms = []ubl.DeliveryTerms{{ID: "CIF"}, {ID: "FOB"}}
		data, err := ubl.Encode(doc)
		require.NoError(t, err)
		xml := string(data)

		start := strings.Index(xml, "<cac:Delivery>")
		require.NotEqual(t, -1, start)
		block := xml[start:]
		party := strings.Index(block, "<cac:DeliveryParty>")
		terms := strings.Index(block, "<cac:DeliveryTerms>")
		require.NotEqual(t, -1, terms)
		// UBL 2.1 orders DeliveryTerms after DeliveryParty.
		assert.Less(t, party, terms)

		parsed, err := ubl.Decode(data)
		require.NoError(t, err)
		out, ok := parsed.(*ubl.Invoice)
		require.True(t, ok)
		require.Len(t, out.Delivery[0].DeliveryTerms, 2)
		assert.Equal(t, "CIF", out.Delivery[0].DeliveryTerms[0].ID)
		assert.Equal(t, "FOB", out.Delivery[0].DeliveryTerms[1].ID)
	})
}
