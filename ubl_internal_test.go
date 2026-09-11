package ubl

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testInvoice builds the smallest invoice that calculates, priced so that the
// line total (3 × 10.555 = 31.665) needs one decimal more than EUR allows.
func testInvoice(t *testing.T) *bill.Invoice {
	t.Helper()

	price := num.MakeAmount(10555, 3)
	inv := &bill.Invoice{
		Regime:    tax.WithRegime("ES"),
		Code:      "TEST-1",
		IssueDate: cal.MakeDate(2024, 1, 15),
		Currency:  currency.EUR,
		Supplier:  &org.Party{Name: "Supplier", TaxID: &tax.Identity{Country: "ES", Code: "B98602642"}},
		Customer:  &org.Party{Name: "Customer", TaxID: &tax.Identity{Country: "ES", Code: "A39200019"}},
		Lines: []*bill.Line{{
			Quantity: num.MakeAmount(3, 0),
			Item:     &org.Item{Name: "Item", Price: &price},
			Taxes:    tax.Set{{Category: "VAT", Percent: num.NewPercentage(210, 3)}},
		}},
	}
	require.NoError(t, inv.Calculate())
	return inv
}

func TestRoundToCurrency(t *testing.T) {
	t.Run("an invoice that already fits is left untouched", func(t *testing.T) {
		inv := testInvoice(t)
		price := num.MakeAmount(1000, 2)
		inv.Lines[0].Item.Price = &price
		require.NoError(t, inv.Calculate())
		require.False(t, exceedsCurrencyPrecision(inv))

		before := inv.Totals.Payable
		require.NoError(t, roundToCurrency(inv))

		assert.Equal(t, before.String(), inv.Totals.Payable.String())
		assert.Nil(t, inv.Totals.Rounding, "no rounding should be invented")
	})

	t.Run("a document without a tax block gets the currency rule", func(t *testing.T) {
		inv := testInvoice(t)
		require.Nil(t, inv.Tax, "the fixture is expected to carry no tax block")
		require.Equal(t, "31.665", inv.Lines[0].Total.String())
		payable := inv.Totals.Payable

		require.NoError(t, roundToCurrency(inv))

		require.NotNil(t, inv.Tax)
		assert.Equal(t, tax.RoundingRuleCurrency, inv.Tax.Rounding)
		assert.Equal(t, "31.67", inv.Lines[0].Total.String(), "BT-131 must fit the currency")
		assert.Equal(t, payable.String(), inv.Totals.Payable.String(), "the amount owed must not move")
	})

	t.Run("an existing rounding total is added to, not replaced", func(t *testing.T) {
		inv := testInvoice(t)

		// A rounding adjustment the document already carried.
		existing := num.MakeAmount(-5, 2)
		inv.Totals.Rounding = &existing
		require.NoError(t, inv.Calculate())
		payable := inv.Totals.Payable

		require.NoError(t, roundToCurrency(inv))

		require.NotNil(t, inv.Totals.Rounding)
		assert.Equal(t, payable.String(), inv.Totals.Payable.String(),
			"the amount owed must survive the rounding")
		// Rounding the line to 31.67 gains a cent, so the -0.05 the document
		// already carried becomes -0.06. Replacing it rather than adding to it
		// would read -0.01 and move the payable total by the 0.05 it dropped.
		assert.Equal(t, "-0.06", inv.Totals.Rounding.String())
	})

	t.Run("a recalculation that cannot resolve its taxes is reported", func(t *testing.T) {
		inv := testInvoice(t)
		// A rate key the regime does not define: the recalculation fails, and
		// the failure must surface rather than yield a half-rounded document.
		inv.Lines[0].Taxes = tax.Set{{Category: "VAT", Rate: "nonsense"}}

		assert.Error(t, roundToCurrency(inv))
	})
}
