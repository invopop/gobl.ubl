package utog

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define tests for the ParseXMLLines function
func TestGetLines(t *testing.T) {
	t.Run("ubl-example1.xml", func(t *testing.T) {
		e, err := newDocumentFrom("ubl-example1.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		lines := inv.Lines
		assert.NotNil(t, lines)

		assert.Len(t, lines, 20)

		line := lines[0]
		assert.Equal(t, "PATAT FRITES 10MM 10KG", line.Item.Name)
		assert.Equal(t, "2", line.Quantity.String())
		assert.Equal(t, org.Unit("item"), line.Item.Unit)
		assert.Equal(t, "9.95", line.Item.Price.String())
		assert.Equal(t, cbc.Code("VAT"), line.Taxes[0].Category)
		assert.Equal(t, "6%", line.Taxes[0].Percent.String())

		line = lines[19]
		assert.Equal(t, "FRITUUR VET 10 KG RETOUR ", line.Item.Name)
		assert.Equal(t, "6", line.Quantity.String())
		assert.Equal(t, org.Unit("item"), line.Item.Unit)
		assert.Equal(t, "18.33", line.Item.Price.String())
		assert.Equal(t, cbc.Code("VAT"), line.Taxes[0].Category)
		assert.Equal(t, "6%", line.Taxes[0].Percent.String())
	})

	// Line Charges and Discounts
	t.Run("ubl-example2.xml", func(t *testing.T) {
		e, err := newDocumentFrom("ubl-example2.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		lines := inv.Lines
		assert.NotNil(t, lines)
		assert.Len(t, lines, 5)

		// Check the first line
		line := lines[0]
		assert.Equal(t, "Laptop computer", line.Item.Name)
		assert.Equal(t, "2", line.Quantity.String())
		assert.Equal(t, "JB007", line.Item.Ref)
		assert.Equal(t, "Scratch on box", line.Notes[0].Text)
		assert.Equal(t, "Processor: Intel Core 2 Duo SU9400 LV (1.4GHz). RAM: 3MB. Screen 1440x900", line.Item.Description)
		assert.Equal(t, org.Unit("item"), line.Item.Unit)
		assert.Equal(t, l10n.ISOCountryCode("DE"), line.Item.Origin)
		assert.Equal(t, "1273.00", line.Item.Price.String())
		assert.Equal(t, cbc.Code("VAT"), line.Taxes[0].Category)
		assert.Equal(t, "25%", line.Taxes[0].Percent.String())

		assert.Len(t, line.Charges, 1)
		charge := line.Charges[0]
		assert.Equal(t, "12.00", charge.Amount.String())
		assert.Equal(t, "Testing", charge.Reason)

		assert.Len(t, line.Discounts, 1)
		discount := line.Discounts[0]
		assert.Equal(t, "12.00", discount.Amount.String())
		assert.Equal(t, "Damage", discount.Reason)

		assert.Len(t, line.Item.Identities, 3)
		assert.Equal(t, cbc.Code("1234567890128"), line.Item.Identities[0].Code)
		assert.Equal(t, "0088", line.Item.Identities[0].Ext[iso.ExtKeySchemeID].String())
		assert.Equal(t, cbc.Code("12344321"), line.Item.Identities[1].Code)
		assert.Equal(t, "ZZZ", line.Item.Identities[1].Label)
		assert.Equal(t, cbc.Code("65434568"), line.Item.Identities[2].Code)
		assert.Equal(t, "STI", line.Item.Identities[2].Label)

		assert.Len(t, line.Item.Meta, 1)
		assert.Equal(t, "Black", line.Item.Meta[cbc.Key("color")])

		// Check the second line
		line = lines[1]
		assert.Equal(t, "Returned \"Advanced computing\" book", line.Item.Name)
		assert.Equal(t, "-1", line.Quantity.String())
		assert.Equal(t, org.Unit("item"), line.Item.Unit)
		assert.Equal(t, "3.96", line.Item.Price.String())
		assert.Equal(t, cbc.Code("VAT"), line.Taxes[0].Category)
		assert.Equal(t, "15%", line.Taxes[0].Percent.String())
	})
}
