package ubl_test

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePayment(t *testing.T) {
	t.Run("general example", func(t *testing.T) {
		e, err := testParseInvoice("ubl-example2.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		payment := inv.Payment
		require.NotNil(t, payment)

		// Check Payee
		require.NotNil(t, payment.Payee)
		assert.Equal(t, "Ebeneser Scrooge AS", payment.Payee.Name)
		require.Len(t, payment.Payee.Identities, 2)
		assert.Equal(t, cbc.Code("989823401"), payment.Payee.Identities[0].Code)

		// Check Instructions
		require.NotNil(t, payment.Instructions)
		assert.Equal(t, cbc.Key("credit-transfer"), payment.Instructions.Key)
		assert.Equal(t, cbc.Code("0003434323213231"), payment.Instructions.Ref)
		require.NotNil(t, payment.Instructions.CreditTransfer)
		require.Len(t, payment.Instructions.CreditTransfer, 1)
		assert.Equal(t, "NO9386011117947", payment.Instructions.CreditTransfer[0].IBAN)
		assert.Equal(t, "DNBANOKK", payment.Instructions.CreditTransfer[0].BIC)

		// Check Terms
		require.NotNil(t, payment.Terms)
		assert.Equal(t, "2 % discount if paid within 2 days\n            Penalty percentage 10% from due date", payment.Terms.Notes)
	})
}

func TestParsePaymentPayee(t *testing.T) {
	t.Run("payee with two identities", func(t *testing.T) {
		e, err := testParseInvoice("ubl-example2.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		payment := inv.Payment
		require.NotNil(t, payment)

		require.NotNil(t, payment.Payee)
		assert.Equal(t, "Ebeneser Scrooge AS", payment.Payee.Name)

		require.Len(t, payment.Payee.Identities, 2)
		assert.Equal(t, cbc.Code("989823401"), payment.Payee.Identities[0].Code)
		assert.Equal(t, "0088", payment.Payee.Identities[1].Ext[iso.ExtKeySchemeID].String())
		assert.Equal(t, cbc.Code("2298740918237"), payment.Payee.Identities[1].Code)
	})

	t.Run("payee with one identity", func(t *testing.T) {
		e, err := testParseInvoice("ubl-example5.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		payment := inv.Payment
		require.NotNil(t, payment)

		require.NotNil(t, payment.Payee)
		assert.Equal(t, "Dagobert Duck", payment.Payee.Name)
		require.Len(t, payment.Payee.Identities, 1)
		assert.Equal(t, cbc.Code("DK16356608"), payment.Payee.Identities[0].Code)
	})
}

func TestParsePaymentInstructions(t *testing.T) {
	t.Run("instructions with credit transfer", func(t *testing.T) {
		e, err := testParseInvoice("ubl-example2.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		payment := inv.Payment
		require.NotNil(t, payment)

		require.NotNil(t, payment.Instructions)
		assert.Equal(t, cbc.Key("credit-transfer"), payment.Instructions.Key)
		assert.Equal(t, cbc.Code("0003434323213231"), payment.Instructions.Ref)
		require.NotNil(t, payment.Instructions.CreditTransfer)
		require.Len(t, payment.Instructions.CreditTransfer, 1)
		assert.Equal(t, "NO9386011117947", payment.Instructions.CreditTransfer[0].IBAN)
		assert.Equal(t, "DNBANOKK", payment.Instructions.CreditTransfer[0].BIC)
	})

	t.Run("instructions with direct debit", func(t *testing.T) {
		e, err := testParseInvoice("ubl-example5.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		payment := inv.Payment
		require.NotNil(t, payment)

		require.NotNil(t, payment.Instructions)
		assert.Equal(t, cbc.Key("direct-debit"), payment.Instructions.Key)
		assert.Equal(t, cbc.Code("Payref1"), payment.Instructions.Ref)
		require.NotNil(t, payment.Instructions.DirectDebit)
		assert.Equal(t, "123456", payment.Instructions.DirectDebit.Ref)
		assert.Equal(t, "DK1212341234123412", payment.Instructions.DirectDebit.Account)
	})

	t.Run("instructions with notes", func(t *testing.T) {
		e, err := testParseInvoice("ubl-example7.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		payment := inv.Payment
		require.NotNil(t, payment)

		require.NotNil(t, payment.Instructions)
		assert.Equal(t, cbc.Key("credit-transfer"), payment.Instructions.Key)
		require.NotNil(t, payment.Instructions.CreditTransfer)
		require.Len(t, payment.Instructions.CreditTransfer, 1)
		assert.Equal(t, "SEXDABCD", payment.Instructions.CreditTransfer[0].BIC)
		assert.Equal(t, "SE1212341234123412", payment.Instructions.CreditTransfer[0].IBAN)
	})

	t.Run("instructions with only IBAN", func(t *testing.T) {
		e, err := testParseInvoice("ubl-example8.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		payment := inv.Payment
		require.NotNil(t, payment)

		require.NotNil(t, payment.Instructions)
		assert.Equal(t, cbc.Key("credit-transfer"), payment.Instructions.Key)
		assert.Equal(t, cbc.Code("1100512149"), payment.Instructions.Ref)
		require.NotNil(t, payment.Instructions.CreditTransfer)
		require.Len(t, payment.Instructions.CreditTransfer, 1)
		assert.Equal(t, "NL28RBOS0420242228", payment.Instructions.CreditTransfer[0].IBAN)
	})
}

func TestParsePaymentTerms(t *testing.T) {
	t.Run("terms with notes", func(t *testing.T) {
		e, err := testParseInvoice("ubl-example2.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		payment := inv.Payment
		require.NotNil(t, payment)
		require.NotNil(t, payment.Terms)

		assert.Equal(t, "2 % discount if paid within 2 days\n            Penalty percentage 10% from due date", payment.Terms.Notes)
	})

	t.Run("only due date present", func(t *testing.T) {
		e, err := testParseInvoice("ubl-example10.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		payment := inv.Payment
		require.NotNil(t, payment)
		require.NotNil(t, payment.Terms)

		require.Len(t, payment.Terms.DueDates, 1)
		assert.NotNil(t, payment.Terms.DueDates[0].Date)
		assert.NotNil(t, payment.Terms.DueDates[0].Percent)
		assert.Equal(t, "100%", payment.Terms.DueDates[0].Percent.String())
	})
}

func TestParsePaymentAdvances(t *testing.T) {
	t.Run("totals with prepaid amount", func(t *testing.T) {
		e, err := testParseInvoice("ubl-example2.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		payment := inv.Payment
		require.NotNil(t, payment)
		require.NotNil(t, payment.Advances)

		assert.Equal(t, "1000.00", payment.Advances[0].Amount.String())
	})
}
