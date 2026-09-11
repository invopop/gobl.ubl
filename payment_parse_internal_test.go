package ubl

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoblCard(t *testing.T) {
	ptr := func(s string) *string { return &s }

	tests := []struct {
		name       string
		card       CardAccount
		wantLast4  string
		wantHolder string
	}{
		{
			name:      "a full PAN is reduced to its last four digits",
			card:      CardAccount{PrimaryAccountNumberID: ptr("4111111111111234")},
			wantLast4: "1234",
		},
		{
			name:      "a PAN that is already four digits is kept",
			card:      CardAccount{PrimaryAccountNumberID: ptr("1234")},
			wantLast4: "1234",
		},
		{
			name:      "a shorter PAN is kept as it stands",
			card:      CardAccount{PrimaryAccountNumberID: ptr("123")},
			wantLast4: "123",
		},
		{
			name:       "holder name",
			card:       CardAccount{PrimaryAccountNumberID: ptr("4111111111111234"), HolderName: ptr("Jane Sample")},
			wantLast4:  "1234",
			wantHolder: "Jane Sample",
		},
		{
			name: "no card details at all",
			card: CardAccount{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := goblCard(&PaymentMeans{CardAccount: &tt.card})
			require.NotNil(t, got)
			assert.Equal(t, tt.wantLast4, got.Last4)
			assert.Equal(t, tt.wantHolder, got.Holder)
		})
	}
}

func TestGoblPaymentMeansCode(t *testing.T) {
	tests := []struct {
		code string
		want cbc.Key
	}{
		{"10", pay.MeansKeyCash},
		{"30", pay.MeansKeyCreditTransfer},
		{"48", pay.MeansKeyCard},
		{"49", pay.MeansKeyDirectDebit},
		{"59", pay.MeansKeyDirectDebit.With(pay.MeansKeySEPA)},
		{"", pay.MeansKeyAny},
		{"not-a-code", pay.MeansKeyAny},
		{"97", pay.MeansKeyAny},
	}

	for _, tt := range tests {
		t.Run("code "+tt.code, func(t *testing.T) {
			assert.Equal(t, tt.want, goblPaymentMeansCode(tt.code))
		})
	}
}

func TestGoblInvoiceDirectDebit(t *testing.T) {
	sepa := func(code string) *org.Identity {
		return &org.Identity{Label: "SEPA", Code: cbc.Code(code)}
	}
	account := "DE89370400440532013000"

	t.Run("mandate reference and payer account", func(t *testing.T) {
		dd := goblInvoiceDirectDebit(&bill.Invoice{}, &PaymentMeans{
			PaymentMandate: &PaymentMandate{
				ID:                    &IDType{Value: "MANDATE-1"},
				PayerFinancialAccount: &FinancialAccount{ID: &account},
			},
		})
		require.NotNil(t, dd)
		assert.Equal(t, "MANDATE-1", dd.Ref)
		assert.Equal(t, account, dd.Account)
		assert.Empty(t, dd.Creditor)
	})

	t.Run("an empty mandate yields no details", func(t *testing.T) {
		dd := goblInvoiceDirectDebit(&bill.Invoice{}, &PaymentMeans{PaymentMandate: &PaymentMandate{}})
		require.NotNil(t, dd)
		assert.Empty(t, dd.Ref)
		assert.Empty(t, dd.Account)
	})

	t.Run("the creditor comes from the supplier's SEPA identity", func(t *testing.T) {
		inv := &bill.Invoice{Supplier: &org.Party{
			Identities: []*org.Identity{{Label: "OTHER", Code: "ignored"}, sepa("CREDITOR-1")},
		}}
		dd := goblInvoiceDirectDebit(inv, &PaymentMeans{PaymentMandate: &PaymentMandate{}})
		assert.Equal(t, "CREDITOR-1", dd.Creditor)
	})

	t.Run("a payee SEPA identity takes precedence over the supplier", func(t *testing.T) {
		inv := &bill.Invoice{
			Supplier: &org.Party{Identities: []*org.Identity{sepa("SUPPLIER")}},
			Payment: &bill.PaymentDetails{
				Payee: &org.Party{Identities: []*org.Identity{sepa("PAYEE")}},
			},
		}
		dd := goblInvoiceDirectDebit(inv, &PaymentMeans{PaymentMandate: &PaymentMandate{}})
		assert.Equal(t, "PAYEE", dd.Creditor)
	})

	t.Run("a payee without a SEPA identity leaves the supplier's", func(t *testing.T) {
		inv := &bill.Invoice{
			Supplier: &org.Party{Identities: []*org.Identity{sepa("SUPPLIER")}},
			Payment: &bill.PaymentDetails{
				Payee: &org.Party{Identities: []*org.Identity{{Label: "OTHER", Code: "x"}}},
			},
		}
		dd := goblInvoiceDirectDebit(inv, &PaymentMeans{PaymentMandate: &PaymentMandate{}})
		assert.Equal(t, "SUPPLIER", dd.Creditor)
	})
}
