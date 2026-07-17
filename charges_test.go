package ubl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCharges(t *testing.T) {
	t.Run("invoice-complete.json", func(t *testing.T) {
		doc := testInvoiceFrom(t, "invoice-complete.json")

		assert.Len(t, doc.AllowanceCharge, 2)

		assert.True(t, doc.AllowanceCharge[0].ChargeIndicator)
		assert.Equal(t, "11.00", doc.AllowanceCharge[0].Amount.Value)
		assert.Equal(t, "Freight", *doc.AllowanceCharge[0].AllowanceChargeReason)

		assert.False(t, doc.AllowanceCharge[1].ChargeIndicator)
		assert.Equal(t, "88", *doc.AllowanceCharge[1].AllowanceChargeReasonCode)
		assert.Equal(t, "10.00", doc.AllowanceCharge[1].Amount.Value)
		assert.Equal(t, "Promotion discount", *doc.AllowanceCharge[1].AllowanceChargeReason)
	})
}

// TestChargeBaseOverridesInvoiceSum covers 826af66: a charge/discount with
// its own explicit Base must use that Base for cbc:BaseAmount, not the
// invoice sum -- e.g. OIOUBL folds excise duty into the line sum, so a
// percentage charge computed against the pre-duty amount would otherwise be
// wrongly reported against the post-duty invoice sum.
func TestChargeBaseOverridesInvoiceSum(t *testing.T) {
	doc := testInvoiceFrom(t, "invoice-charge-explicit-base.json")

	require.Len(t, doc.AllowanceCharge, 2)

	charge := doc.AllowanceCharge[0]
	assert.True(t, charge.ChargeIndicator)
	assert.Equal(t, "Document charge on a different base", *charge.AllowanceChargeReason)
	require.NotNil(t, charge.BaseAmount)
	assert.Equal(t, "300.00", charge.BaseAmount.Value, "should use the charge's own base, not the invoice sum")
	assert.Equal(t, "30.00", charge.Amount.Value)

	discount := doc.AllowanceCharge[1]
	assert.False(t, discount.ChargeIndicator)
	assert.Equal(t, "Document discount on a different base", *discount.AllowanceChargeReason)
	require.NotNil(t, discount.BaseAmount)
	assert.Equal(t, "500.00", discount.BaseAmount.Value, "should use the discount's own base, not the invoice sum")
	assert.Equal(t, "25.00", discount.Amount.Value)
}
