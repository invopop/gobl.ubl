package ubl_test

import (
	"testing"

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
