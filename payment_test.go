package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/pay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPayment(t *testing.T) {
	t.Run("self-billed-invoice", func(t *testing.T) {
		doc, err := testInvoiceFrom("peppol-self-billed/self-billed-invoice.json")
		require.NoError(t, err)

		// PayeeParty should have PartyName (BR-17) but not RegistrationName (UBL-CR-275)
		assert.Equal(t, "Ebeneser Scrooge AS", doc.PayeeParty.PartyName.Name)
		assert.Equal(t, "2013-07-20", doc.DueDate)

		assert.Equal(t, "30", doc.PaymentMeans[0].PaymentMeansCode.Value)
		assert.Equal(t, "0003434323213231", *doc.PaymentMeans[0].PaymentID)
		assert.NotEmpty(t, doc.PaymentMeans[0].PayeeFinancialAccount)
		assert.Equal(t, "NO9386011117947", *doc.PaymentMeans[0].PayeeFinancialAccount.ID)
		assert.Equal(t, "DNBANOKK", *doc.PaymentMeans[0].PayeeFinancialAccount.FinancialInstitutionBranch.ID)
	})

	t.Run("credit transfer with account number", func(t *testing.T) {
		doc, err := testInvoiceFrom("invoice-account-number.json")
		require.NoError(t, err)

		// Verify the account number was set in the UBL financial account ID
		assert.NotEmpty(t, doc.PaymentMeans[0].PayeeFinancialAccount)
		assert.Equal(t, "123456789", *doc.PaymentMeans[0].PayeeFinancialAccount.ID)
		assert.Equal(t, "Test Bank Account", *doc.PaymentMeans[0].PayeeFinancialAccount.Name)
		assert.Equal(t, "DNBANOKK", *doc.PaymentMeans[0].PayeeFinancialAccount.FinancialInstitutionBranch.ID)
	})

	t.Run("credit transfer with no account fields omits financial account", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Payment.Instructions.CreditTransfer = []*pay.CreditTransfer{{}}

		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)
		assert.Nil(t, doc.PaymentMeans[0].PayeeFinancialAccount)
	})

	t.Run("card with empty last4 omits PAN", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Payment.Instructions.Card = &pay.Card{Holder: "John Doe"}

		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)
		require.NotNil(t, doc.PaymentMeans[0].CardAccount)
		assert.Nil(t, doc.PaymentMeans[0].CardAccount.PrimaryAccountNumberID)
		assert.Equal(t, "John Doe", *doc.PaymentMeans[0].CardAccount.HolderName)
	})

	t.Run("direct debit with empty ref omits mandate ID", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Payment.Instructions.DirectDebit = &pay.DirectDebit{Account: "DE89370400440532013000"}

		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)
		require.NotNil(t, doc.PaymentMeans[0].PaymentMandate)
		assert.Empty(t, doc.PaymentMeans[0].PaymentMandate.ID.Value)
		assert.Equal(t, "DE89370400440532013000", *doc.PaymentMeans[0].PaymentMandate.PayerFinancialAccount.ID)
	})

	t.Run("instruction detail mapped to payment means name", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Payment.Instructions.Detail = "Bank transfer"

		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)
		require.NotNil(t, doc.PaymentMeans[0].PaymentMeansCode.Name)
		assert.Equal(t, "Bank transfer", *doc.PaymentMeans[0].PaymentMeansCode.Name)
	})

	t.Run("payment terms with empty notes omits payment terms", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Payment.Terms.Notes = ""

		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)
		assert.Empty(t, doc.PaymentTerms)
	})

	t.Run("BT-90 creditor ID on supplier when no payee", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Payment.Instructions.DirectDebit = &pay.DirectDebit{Creditor: "DE98ZZZ09999999999"}

		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)
		require.Nil(t, doc.PayeeParty)
		ids := doc.AccountingSupplierParty.Party.PartyIdentification
		require.NotEmpty(t, ids)
		assert.Equal(t, "DE98ZZZ09999999999", ids[len(ids)-1].ID.Value)
		assert.Equal(t, "SEPA", *ids[len(ids)-1].ID.SchemeID)
	})

	t.Run("document type extension", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		assert.True(t, ok)

		inv.Payment.Instructions.Ext = nil

		_, err = ubl.ConvertInvoice(env)
		assert.ErrorContains(t, err, "instructions: (ext: (untdid-payment-means: required.).).")
	})
}
