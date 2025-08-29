package ubl_test

import (
	"strings"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
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

	t.Run("Allowance-example.xml", func(t *testing.T) {
		e, err := testParseInvoice("Allowance-example.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)
		charges := inv.Charges
		discounts := inv.Discounts

		// Check if there's exactly one charge and no discounts at document level
		require.Len(t, charges, 1)
		require.Len(t, discounts, 1)

		// Check the charge with BaseAmount
		charge := charges[0]
		assert.Equal(t, "Cleaning", charge.Reason)
		assert.Equal(t, "1000", charge.Base.String())
		assert.Equal(t, "20%", charge.Percent.String())
		assert.Equal(t, "200", charge.Amount.String())

		discount := discounts[0]
		assert.Equal(t, "Discount", discount.Reason)
		assert.Equal(t, "200.00", discount.Amount.String())
		assert.Nil(t, discount.Base)

		// First line item should have both charges and discounts
		line1 := inv.Lines[0]
		require.Len(t, line1.Charges, 1)
		require.Len(t, line1.Discounts, 1)

		// Check line charge with BaseAmount
		lineCharge := line1.Charges[0]
		assert.Equal(t, "Cleaning", lineCharge.Reason)
		assert.Equal(t, "100.00", lineCharge.Base.String())
		assert.Equal(t, "1%", lineCharge.Percent.String())
		assert.Equal(t, "1.00", lineCharge.Amount.String())

		// Check line discount with BaseAmount
		lineDiscount := line1.Discounts[0]
		assert.Equal(t, "Discount", lineDiscount.Reason)
		assert.Equal(t, "101.00", lineDiscount.Amount.String())
		assert.Nil(t, lineDiscount.Base)
	})

}

func TestBaseAmountErrorHandling(t *testing.T) {
	t.Run("invalid BaseAmount", func(t *testing.T) {
		// Take the Allowance-example.xml content and modify the BaseAmount value to be invalid
		data, err := testLoadXML("Allowance-example.xml")
		require.NoError(t, err)

		// Replace a valid BaseAmount with an invalid one
		invalidXML := strings.ReplaceAll(string(data), `<cbc:BaseAmount currencyID="EUR">1000</cbc:BaseAmount>`, `<cbc:BaseAmount currencyID="EUR">invalid-amount</cbc:BaseAmount>`)

		// Try to parse the modified XML - should fail due to invalid BaseAmount
		_, err = ubl.Parse([]byte(invalidXML))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid major number")
	})
}
