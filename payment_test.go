package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
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

	t.Run("document type extension", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		assert.True(t, ok)

		inv.Payment.Instructions.Ext = nil

		_, err = ubl.ConvertInvoice(env)
		assert.ErrorContains(t, err, "instructions: (ext: (untdid-payment-means: required.).).")
	})

	t.Run("nemhandel payment mapping", func(t *testing.T) {
		doc, err := testInvoiceFrom("oioubl30-invoice-example.json")
		require.NoError(t, err)

		require.NotEmpty(t, doc.PaymentMeans)
		pm := doc.PaymentMeans[0]
		assert.Equal(t, "31", pm.PaymentMeansCode.Value)
		require.NotNil(t, pm.PaymentChannelCode)
		assert.Equal(t, "IBAN", pm.PaymentChannelCode.Value)
		require.NotNil(t, pm.PayeeFinancialAccount)
		require.NotNil(t, pm.PayeeFinancialAccount.FinancialInstitutionBranch)
		require.NotNil(t, pm.PayeeFinancialAccount.FinancialInstitutionBranch.FinancialInstitution)
		require.NotNil(t, pm.PayeeFinancialAccount.FinancialInstitutionBranch.FinancialInstitution.ID)
		assert.Equal(t, "EXAMGB2L", *pm.PayeeFinancialAccount.FinancialInstitutionBranch.FinancialInstitution.ID)
	})

	t.Run("non OIO keeps payment means untouched", func(t *testing.T) {
		doc, err := testInvoiceFrom("invoice-minimal.json")
		require.NoError(t, err)
		require.NotEmpty(t, doc.PaymentMeans)

		pm := doc.PaymentMeans[0]
		assert.Equal(t, "30", pm.PaymentMeansCode.Value)
		assert.Nil(t, pm.PaymentChannelCode)
		if pm.PayeeFinancialAccount != nil && pm.PayeeFinancialAccount.FinancialInstitutionBranch != nil {
			assert.Nil(t, pm.PayeeFinancialAccount.FinancialInstitutionBranch.FinancialInstitution)
		}
	})

	t.Run("OIO keeps explicit payment-channel", func(t *testing.T) {
		env, err := loadTestEnvelope("oioubl30-invoice-example.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		inv.Payment.Instructions.Meta = cbc.Meta{
			cbc.Key("payment-channel"): "ZZZ",
		}

		doc, err := ubl.ConvertInvoice(env, ubl.WithContext(ubl.ContextOIOUBL))
		require.NoError(t, err)
		require.NotEmpty(t, doc.PaymentMeans)
		require.NotNil(t, doc.PaymentMeans[0].PaymentChannelCode)
		assert.Equal(t, "ZZZ", doc.PaymentMeans[0].PaymentChannelCode.Value)
		assert.Equal(t, "31", doc.PaymentMeans[0].PaymentMeansCode.Value)
	})

	t.Run("oioubl21 applies OIO payment mapping", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		doc, err := ubl.ConvertInvoice(env, ubl.WithContext(ubl.ContextOIOUBL21))
		require.NoError(t, err)
		require.NotEmpty(t, doc.PaymentMeans)
		assert.Equal(t, "31", doc.PaymentMeans[0].PaymentMeansCode.Value)
	})

	t.Run("single due date without date does not panic", func(t *testing.T) {
		env, err := loadTestEnvelope("oioubl30-invoice-example.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		require.NotNil(t, inv.Payment)
		require.NotNil(t, inv.Payment.Terms)
		require.Len(t, inv.Payment.Terms.DueDates, 1)

		inv.Payment.Terms.DueDates[0].Date = nil
		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)
		assert.Empty(t, doc.DueDate)
	})
}
