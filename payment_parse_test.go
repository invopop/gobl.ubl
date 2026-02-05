package ubl_test

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePayment(t *testing.T) {
	t.Run("general example", func(t *testing.T) {
		e, err := testParseInvoice("en16931/ubl-example2.xml")
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
		e, err := testParseInvoice("en16931/ubl-example2.xml")
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
		e, err := testParseInvoice("en16931/ubl-example5.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		payment := inv.Payment
		require.NotNil(t, payment)

		require.NotNil(t, payment.Payee)
		assert.Equal(t, "Dagobert Duck", payment.Payee.Name)
		// Both PartyLegalEntity.CompanyID and PartyIdentification.ID are parsed as identities
		require.Len(t, payment.Payee.Identities, 2)
		assert.Equal(t, cbc.Code("DK16356608"), payment.Payee.Identities[0].Code)
		assert.Equal(t, org.IdentityScopeLegal, payment.Payee.Identities[0].Scope)
		assert.Equal(t, cbc.Code("DK16356608"), payment.Payee.Identities[1].Code)
	})
}

func TestParsePaymentInstructions(t *testing.T) {
	t.Run("instructions with credit transfer", func(t *testing.T) {
		e, err := testParseInvoice("en16931/ubl-example2.xml")
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
		e, err := testParseInvoice("en16931/ubl-example5.xml")
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
		e, err := testParseInvoice("en16931/ubl-example7.xml")
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
		e, err := testParseInvoice("en16931/ubl-example8.xml")
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
		e, err := testParseInvoice("en16931/ubl-example2.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		payment := inv.Payment
		require.NotNil(t, payment)
		require.NotNil(t, payment.Terms)

		assert.Equal(t, "2 % discount if paid within 2 days\n            Penalty percentage 10% from due date", payment.Terms.Notes)
	})

	t.Run("only due date present", func(t *testing.T) {
		e, err := testParseInvoice("en16931/ubl-example10.xml")
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
		e, err := testParseInvoice("en16931/ubl-example2.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		payment := inv.Payment
		require.NotNil(t, payment)
		require.NotNil(t, payment.Advances)

		assert.Equal(t, "1000.00", payment.Advances[0].Amount.String())
	})
}

func TestPaymentRoundTrip(t *testing.T) {
	t.Run("account number round trip", func(t *testing.T) {
		// Load the test envelope with account number (not IBAN)
		env, err := loadTestEnvelope("invoice-account-number.json")
		require.NoError(t, err)

		originalInv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Convert to UBL
		doc, err := testInvoiceFrom("invoice-account-number.json")
		require.NoError(t, err)

		// Convert back to GOBL
		resultEnv, err := doc.Convert()
		require.NoError(t, err)

		resultInv, ok := resultEnv.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Check that the account number is preserved (not moved to IBAN)
		require.NotNil(t, resultInv.Payment)
		require.NotNil(t, resultInv.Payment.Instructions)
		require.NotNil(t, resultInv.Payment.Instructions.CreditTransfer)
		require.Len(t, resultInv.Payment.Instructions.CreditTransfer, 1)

		// The original has number, not IBAN
		assert.Equal(t, "123456789", originalInv.Payment.Instructions.CreditTransfer[0].Number)
		assert.Equal(t, "", originalInv.Payment.Instructions.CreditTransfer[0].IBAN)

		// After round-trip, the number should still be in the number field, not IBAN
		assert.Equal(t, "123456789", resultInv.Payment.Instructions.CreditTransfer[0].Number, "Account number should be preserved in Number field")
		assert.Equal(t, "", resultInv.Payment.Instructions.CreditTransfer[0].IBAN, "IBAN should be empty when Number was used")
	})
}
