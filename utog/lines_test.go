package utog

import (
	"testing"

	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define tests for the ParseXMLLines function
func TestGetLines(t *testing.T) {
	t.Run("ubl-example1.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("ubl-example1.xml")
		require.NoError(t, err)

		conversor := NewConversor()
		err = conversor.NewInvoice(doc)
		require.NoError(t, err)

		inv := conversor.GetInvoice()
		lines := inv.Lines
		assert.NotNil(t, lines)

		assert.Len(t, lines, 20)

		line := lines[0]
		assert.Equal(t, "1", line.Index)
		assert.Equal(t, "PATAT FRITES 10MM 10KG", line.Item.Name)
		assert.Equal(t, "2", line.Quantity.String())
		assert.Equal(t, org.Unit("item"), line.Item.Unit)
		assert.Equal(t, "9.95", line.Item.Price.String())
		assert.Equal(t, "19.90", line.Sum.String())
		assert.Equal(t, cbc.Code("VAT"), line.Taxes[0].Category)
		assert.Equal(t, cbc.Key("standard"), line.Taxes[0].Rate)
		assert.Equal(t, "21.0%", line.Taxes[0].Percent.String())

		line = lines[19]
		assert.Equal(t, "20", line.Index)
		assert.Equal(t, "FRITUUR VET 10 KG RETOUR", line.Item.Name)
		assert.Equal(t, "6", line.Quantity.String())
		assert.Equal(t, org.Unit("item"), line.Item.Unit)
		assert.Equal(t, "18.33", line.Item.Price.String())
		assert.Equal(t, "109.98", line.Sum.String())
		assert.Equal(t, cbc.Code("VAT"), line.Taxes[0].Category)
		assert.Equal(t, cbc.Key("standard"), line.Taxes[0].Rate)
		assert.Equal(t, "21.0%", line.Taxes[0].Percent.String())
	})

	// Line Charges and Discounts
	t.Run("ubl-example2.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("ubl-example2.xml")
		require.NoError(t, err)

		conversor := NewConversor()
		err = conversor.NewInvoice(doc)
		require.NoError(t, err)

		inv := conversor.GetInvoice()
		lines := inv.Lines
		assert.NotNil(t, lines)
		assert.Len(t, lines, 4)

		// Check the first line
		line := lines[0]
		assert.Equal(t, "1", line.Index)
		assert.Equal(t, "Laptop computer", line.Item.Name)
		assert.Equal(t, "2", line.Quantity.String())
		assert.Equal(t, "JB007", line.Item.Ref)
		assert.Equal(t, "Processor: Intel Core 2 Duo SU9400 LV (1.4GHz). RAM: 3MB. Screen 1440x900", line.Item.Description)
		assert.Equal(t, "EA", line.Item.Unit)
		assert.Equal(t, "DE", line.Item.Origin)
		assert.Equal(t, "1273.00", line.Item.Price.String())
		assert.Equal(t, "2546.00", line.Sum.String())
		assert.Equal(t, cbc.Code("VAT"), line.Taxes[0].Category)
		assert.Equal(t, cbc.Key("standard"), line.Taxes[0].Rate)
		assert.Equal(t, "25%", line.Taxes[0].Percent.String())
		assert.Equal(t, "2546.00", line.Total.String())

		assert.Len(t, line.Charges, 1)
		charge := line.Charges[0]
		assert.Equal(t, "12.00", charge.Amount.String())
		assert.Equal(t, "Testing", charge.Reason)

		assert.Len(t, line.Discounts, 1)
		discount := line.Discounts[0]
		assert.Equal(t, "12.00", discount.Amount.String())
		assert.Equal(t, "Damage", discount.Reason)

		assert.Len(t, line.Item.Identities, 4)
		assert.Equal(t, "1234567890128", line.Item.Identities[0].Code)
		assert.Equal(t, "0088", line.Item.Identities[0].Label)
		assert.Equal(t, "12344321", line.Item.Identities[1].Code)
		assert.Equal(t, "ZZZ", line.Item.Identities[1].Label)
		assert.Equal(t, "65434568", line.Item.Identities[2].Code)
		assert.Equal(t, "STI", line.Item.Identities[2].Label)

		// Check the second line
		line = lines[1]
		assert.Equal(t, "2", line.Index)
		assert.Equal(t, "Pepsi Max 24x33cl", line.Item.Name)
		assert.Equal(t, "1", line.Quantity.String())
		assert.Equal(t, "CT", line.Item.Unit)
		assert.Equal(t, "19.00", line.Item.Price.String())
		assert.Equal(t, "19.00", line.Sum.String())
		assert.Equal(t, cbc.Code("VAT"), line.Taxes[0].Category)
		assert.Equal(t, cbc.Key("standard"), line.Taxes[0].Rate)
		assert.Equal(t, "25%", line.Taxes[0].Percent.String())
		assert.Equal(t, "19.00", line.Total.String())
	})
}
