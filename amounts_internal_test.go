package ubl

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRescaleToCurrency(t *testing.T) {
	tests := []struct {
		name     string
		amount   num.Amount
		currency string
		expected string
	}{
		{
			name:     "rounds down to the currency's precision",
			amount:   num.MakeAmount(67273, 4), // 6.7273
			currency: "EUR",
			expected: "6.73",
		},
		{
			name:     "pads up to the currency's precision",
			amount:   num.MakeAmount(6, 0),
			currency: "EUR",
			expected: "6.00",
		},
		{
			name:     "currency without subunits",
			amount:   num.MakeAmount(12345, 2), // 123.45
			currency: "JPY",
			expected: "123",
		},
		{
			name:     "three decimal currency",
			amount:   num.MakeAmount(12345, 2), // 123.45
			currency: "BHD",
			expected: "123.450",
		},
		{
			name:     "unknown currency keeps the amount as it stands",
			amount:   num.MakeAmount(67273, 4),
			currency: "ZZZ",
			expected: "6.7273",
		},
		{
			name:     "empty currency keeps the amount as it stands",
			amount:   num.MakeAmount(67273, 4),
			currency: "",
			expected: "6.7273",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, rescaleToCurrency(tt.amount, tt.currency))
		})
	}
}

func TestNewAmount(t *testing.T) {
	t.Run("rounds and carries the currency", func(t *testing.T) {
		a := newAmount(num.MakeAmount(508760, 4), "EUR") // 50.8760
		assert.Equal(t, "50.88", a.Value)
		require.NotNil(t, a.CurrencyID)
		assert.Equal(t, "EUR", *a.CurrencyID)
	})

	t.Run("each amount owns its currency attribute", func(t *testing.T) {
		first := newAmount(num.MakeAmount(100, 2), "EUR")
		second := newAmount(num.MakeAmount(200, 2), "USD")
		assert.Equal(t, "EUR", *first.CurrencyID)
		assert.Equal(t, "USD", *second.CurrencyID)
	})

	t.Run("pointer variant", func(t *testing.T) {
		a := newAmountPtr(num.MakeAmount(1, 0), "EUR")
		require.NotNil(t, a)
		assert.Equal(t, "1.00", a.Value)
	})

	t.Run("unit amounts keep their precision", func(t *testing.T) {
		// BT-146 is exempt from the BR-DEC rules, so the extra decimals a
		// tax-exclusive unit price needs must survive.
		a := newUnitAmount(num.MakeAmount(81818, 4), "EUR") // 8.1818
		assert.Equal(t, "8.1818", a.Value)
		assert.Equal(t, "EUR", *a.CurrencyID)
	})
}

func TestExceedsCurrencyPrecision(t *testing.T) {
	// over builds an amount with one decimal more than EUR allows.
	over := num.MakeAmount(67273, 4) // 6.7273
	fits := num.MakeAmount(673, 2)   // 6.73
	zero := num.MakeAmount(0, 2)     //
	invoice := func(f func(inv *bill.Invoice)) *bill.Invoice {
		inv := &bill.Invoice{
			Currency: currency.EUR,
			Totals:   &bill.Totals{Sum: zero},
			Lines:    []*bill.Line{{Index: 1, Total: &fits, Sum: &fits}},
		}
		if f != nil {
			f(inv)
		}
		return inv
	}

	t.Run("no totals means nothing to round", func(t *testing.T) {
		inv := invoice(nil)
		inv.Totals = nil
		assert.False(t, exceedsCurrencyPrecision(inv))
	})

	t.Run("unknown currency is left alone", func(t *testing.T) {
		inv := invoice(func(inv *bill.Invoice) {
			inv.Currency = currency.Code("ZZZ")
			inv.Lines[0].Total = &over
		})
		assert.False(t, exceedsCurrencyPrecision(inv))
	})

	t.Run("amounts within the currency", func(t *testing.T) {
		assert.False(t, exceedsCurrencyPrecision(invoice(nil)))
	})

	t.Run("line total", func(t *testing.T) {
		assert.True(t, exceedsCurrencyPrecision(invoice(func(inv *bill.Invoice) {
			inv.Lines[0].Total = &over
		})))
	})

	t.Run("line sum", func(t *testing.T) {
		assert.True(t, exceedsCurrencyPrecision(invoice(func(inv *bill.Invoice) {
			inv.Lines[0].Sum = &over
		})))
	})

	t.Run("line discount", func(t *testing.T) {
		assert.True(t, exceedsCurrencyPrecision(invoice(func(inv *bill.Invoice) {
			inv.Lines[0].Discounts = []*bill.LineDiscount{{Amount: over}}
		})))
	})

	t.Run("line charge", func(t *testing.T) {
		assert.True(t, exceedsCurrencyPrecision(invoice(func(inv *bill.Invoice) {
			inv.Lines[0].Charges = []*bill.LineCharge{{Amount: over}}
		})))
	})

	t.Run("document discount", func(t *testing.T) {
		assert.True(t, exceedsCurrencyPrecision(invoice(func(inv *bill.Invoice) {
			inv.Discounts = []*bill.Discount{{Amount: over}}
		})))
	})

	t.Run("document discount base", func(t *testing.T) {
		assert.True(t, exceedsCurrencyPrecision(invoice(func(inv *bill.Invoice) {
			inv.Discounts = []*bill.Discount{{Amount: fits, Base: &over}}
		})))
	})

	t.Run("document charge", func(t *testing.T) {
		assert.True(t, exceedsCurrencyPrecision(invoice(func(inv *bill.Invoice) {
			inv.Charges = []*bill.Charge{{Amount: over}}
		})))
	})

	t.Run("document charge base", func(t *testing.T) {
		assert.True(t, exceedsCurrencyPrecision(invoice(func(inv *bill.Invoice) {
			inv.Charges = []*bill.Charge{{Amount: fits, Base: &over}}
		})))
	})

	t.Run("tax rate base", func(t *testing.T) {
		assert.True(t, exceedsCurrencyPrecision(invoice(func(inv *bill.Invoice) {
			inv.Totals.Taxes = &tax.Total{Categories: []*tax.CategoryTotal{
				{Rates: []*tax.RateTotal{{Base: over, Amount: fits}}},
			}}
		})))
	})

	t.Run("tax rate amount", func(t *testing.T) {
		assert.True(t, exceedsCurrencyPrecision(invoice(func(inv *bill.Invoice) {
			inv.Totals.Taxes = &tax.Total{Categories: []*tax.CategoryTotal{
				{Rates: []*tax.RateTotal{{Base: fits, Amount: over}}},
			}}
		})))
	})
}

func TestNewEndpointID(t *testing.T) {
	tests := []struct {
		name       string
		party      *org.Party
		wantScheme string
		wantValue  string
		wantNil    bool
	}{
		{
			name: "iso6523 endpoint",
			party: &org.Party{Endpoints: []*org.Endpoint{
				{URI: "iso6523-actorid-upis::0088:7300010000001"},
			}},
			wantScheme: "0088",
			wantValue:  "7300010000001",
		},
		{
			name: "mailto endpoint",
			party: &org.Party{Endpoints: []*org.Endpoint{
				{URI: "mailto:billing@example.com"},
			}},
			wantScheme: SchemeIDEmail,
			wantValue:  "billing@example.com",
		},
		{
			name: "endpoint wins over the deprecated inbox",
			party: &org.Party{
				Endpoints: []*org.Endpoint{{URI: "iso6523-actorid-upis::0225:356000000"}},
				Inboxes:   []*org.Inbox{{Scheme: "9957", Code: "356000000"}},
			},
			wantScheme: "0225",
			wantValue:  "356000000",
		},
		{
			name: "an endpoint without a code falls back to the inbox",
			party: &org.Party{
				Endpoints: []*org.Endpoint{{URI: "iso6523-actorid-upis::0088"}},
				Inboxes:   []*org.Inbox{{Scheme: "0151", Code: "99100100100"}},
			},
			wantScheme: "0151",
			wantValue:  "99100100100",
		},
		{
			name: "an unrelated endpoint scheme falls back to the inbox",
			party: &org.Party{
				Endpoints: []*org.Endpoint{{URI: "gobl:acme.example.com"}},
				Inboxes:   []*org.Inbox{{Scheme: "0151", Code: "99100100100"}},
			},
			wantScheme: "0151",
			wantValue:  "99100100100",
		},
		{
			name:       "deprecated inbox with an email",
			party:      &org.Party{Inboxes: []*org.Inbox{{Email: "inbox@example.com"}}},
			wantScheme: SchemeIDEmail,
			wantValue:  "inbox@example.com",
		},
		{
			name: "the first usable inbox is taken",
			party: &org.Party{Inboxes: []*org.Inbox{
				{Key: "peppol"},
				{Scheme: "0088", Code: "7300010000001"},
			}},
			wantScheme: "0088",
			wantValue:  "7300010000001",
		},
		{
			name:    "no electronic address at all",
			party:   &org.Party{Name: "Nobody"},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newEndpointID(tt.party)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.wantScheme, got.SchemeID)
			assert.Equal(t, tt.wantValue, got.Value)
		})
	}
}
