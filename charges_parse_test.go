package ubl_test

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCharges(t *testing.T) {
	t.Run("ubl-example2.xml", func(t *testing.T) {
		e, err := testParseInvoice("ubl-example2.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)
		charges := inv.Charges
		discounts := inv.Discounts

		// Check if there's exactly one charge
		require.Len(t, charges, 1)
		require.Len(t, discounts, 1)

		// Check the values of the charge
		charge := charges[0]
		assert.Equal(t, "Freight", charge.Reason)
		assert.Equal(t, "100.00", charge.Amount.String())

		// Check the tax category of the charge
		require.NotNil(t, charge.Taxes)
		assert.Equal(t, cbc.Code("VAT"), charge.Taxes[0].Category)
		assert.Equal(t, "25%", charge.Taxes[0].Percent.String())

		// Check the values of the discount
		discount := discounts[0]
		assert.Equal(t, "Promotion discount", discount.Reason)
		assert.Equal(t, "88", discount.Ext[untdid.ExtKeyAllowance].String())
		assert.Equal(t, "100.00", discount.Amount.String())

		// Check the tax category of the discount
		require.NotNil(t, discount.Taxes)
		assert.Equal(t, cbc.Code("VAT"), discount.Taxes[0].Category)
		assert.Equal(t, "25%", discount.Taxes[0].Percent.String())
	})

	t.Run("ubl-example5.xml", func(t *testing.T) {
		e, err := testParseInvoice("ubl-example5.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		charges := inv.Charges
		discounts := inv.Discounts

		// Check if there's exactly one charge and one discount
		require.Len(t, charges, 1)
		require.Len(t, discounts, 1)

		// Check the values of the charge
		charge := charges[0]
		assert.Equal(t, "Packaging", charge.Reason)
		assert.Equal(t, "ABL", charge.Ext[untdid.ExtKeyCharge].String())
		assert.Equal(t, "150.00", charge.Amount.String())

		// Check the tax category of the charge
		require.NotNil(t, charge.Taxes)
		assert.Equal(t, cbc.Code("VAT"), charge.Taxes[0].Category)
		assert.Equal(t, "25%", charge.Taxes[0].Percent.String())

		// Check the values of the discount
		discount := discounts[0]
		assert.Equal(t, "Loyal customer", discount.Reason)
		assert.Equal(t, "100", discount.Ext[untdid.ExtKeyAllowance].String())
		assert.Equal(t, "150.00", discount.Amount.String())

		// Check the tax category of the discount
		require.NotNil(t, discount.Taxes)
		assert.Equal(t, cbc.Code("VAT"), discount.Taxes[0].Category)
		assert.Equal(t, "25%", discount.Taxes[0].Percent.String())
	})

}
