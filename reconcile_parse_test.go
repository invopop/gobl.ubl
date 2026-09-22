package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseDeclaredTotals covers reconciliation against the declared amounts.
// BT-131 is mandatory and the totals are summed from it, so it decides.
func TestParseDeclaredTotals(t *testing.T) {
	// A multiplier its base (BT-137) reproduces: the line reconciles.
	t.Run("line allowance with a declared base", func(t *testing.T) {
		e := parseXMLInvoice(t, "en16931/line-allowance-base.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)
		assert.False(t, inv.HasTags(tax.TagBypass))

		require.Len(t, inv.Lines, 1)
		// 200 x 10.00, less 10% of the declared 1000.00 base.
		assert.Equal(t, "2000.00", inv.Lines[0].Sum.String())
		assert.Equal(t, "1900.00", inv.Lines[0].Total.String())
		assert.Equal(t, "1900.00", inv.Totals.Sum.String())
		assert.Equal(t, "2280.00", inv.Totals.Payable.String())
	})

	// Reproducible under no reading: the sender's figures are kept.
	t.Run("line totals that cannot be reproduced", func(t *testing.T) {
		e := parseXMLInvoice(t, "en16931/line-totals-mismatch.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)
		assert.True(t, inv.HasTags(tax.TagBypass))

		require.Len(t, inv.Lines, 1)
		assert.Equal(t, "1614.87", inv.Lines[0].Total.String())
		assert.Equal(t, "1614.87", inv.Totals.Sum.String())
		assert.Equal(t, "1937.84", inv.Totals.Payable.String())
	})
}

// TestLineAllowanceBaseRoundTrip guards PEPPOL-EN16931-R040: the amount must
// equal base x percentage, so the sender's base has to survive the round trip.
func TestLineAllowanceBaseRoundTrip(t *testing.T) {
	e := parseXMLInvoice(t, "en16931/line-allowance-base.xml")

	inv, ok := e.Extract().(*bill.Invoice)
	require.True(t, ok)
	require.Len(t, inv.Lines, 1)
	require.Len(t, inv.Lines[0].Discounts, 1)

	d := inv.Lines[0].Discounts[0]
	require.NotNil(t, d.Base)
	// The declared basis, not the 2000.00 line sum.
	assert.Equal(t, "1000.00", d.Base.String())
	require.NotNil(t, d.Percent)
	assert.Equal(t, "10.00%", d.Percent.String())
	assert.Equal(t, "100.00", d.Amount.String())

	out, err := ubl.ConvertInvoice(e)
	require.NoError(t, err)
	require.Len(t, out.InvoiceLines, 1)
	require.Len(t, out.InvoiceLines[0].AllowanceCharge, 1)

	ac := out.InvoiceLines[0].AllowanceCharge[0]
	require.NotNil(t, ac.BaseAmount)
	assert.Equal(t, "1000.00", ac.BaseAmount.Value)
	require.NotNil(t, ac.MultiplierFactorNumeric)
	assert.Equal(t, "10.00", *ac.MultiplierFactorNumeric)
	assert.Equal(t, "100.00", ac.Amount.Value)
}
