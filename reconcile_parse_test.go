package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseDeclaredTotals covers the reconciliation of a converted document
// against the amounts the sender declared. The invoice line net amount (BT-131)
// is mandatory and is what the document totals are summed from, so it decides
// whenever the terms it is derived from disagree with it.
func TestParseDeclaredTotals(t *testing.T) {
	// A multiplier backed by a base amount (BT-137) that reproduces the
	// declared amount: the line reconciles and keeps its calculated totals.
	t.Run("line allowance with a declared base", func(t *testing.T) {
		e := parseXMLInvoice(t, "en16931/line-allowance-base.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)
		assert.False(t, inv.HasTags(tax.TagBypass))

		require.Len(t, inv.Lines, 1)
		// 200 x 10.00, less 10% of the declared base of 1000.00
		assert.Equal(t, "2000.00", inv.Lines[0].Sum.String())
		assert.Equal(t, "1900.00", inv.Lines[0].Total.String())
		assert.Equal(t, "1900.00", inv.Totals.Sum.String())
		assert.Equal(t, "2280.00", inv.Totals.Payable.String())
	})

	// A line whose price and quantity cannot reproduce its declared amount
	// under any reading: the document keeps the sender's own figures.
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

// TestLineAllowanceBaseRoundTrip guards PEPPOL-EN16931-R040, which requires a
// line allowance amount to equal its base amount times its percentage. The
// basis the sender calculated on has to survive the round trip; emitting the
// line sum in its place breaks the rule on a document that arrived valid.
func TestLineAllowanceBaseRoundTrip(t *testing.T) {
	e := parseXMLInvoice(t, "en16931/line-allowance-base.xml")

	inv, ok := e.Extract().(*bill.Invoice)
	require.True(t, ok)
	require.Len(t, inv.Lines, 1)
	require.Len(t, inv.Lines[0].Discounts, 1)

	d := inv.Lines[0].Discounts[0]
	require.NotNil(t, d.Base)
	// The declared basis, not the line sum of 2000.00 it was applied to.
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
