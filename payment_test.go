package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPayment(t *testing.T) {
	t.Run("self-billed-invoice", func(t *testing.T) {
		doc := testInvoiceFrom(t, "peppol-self-billed/self-billed-invoice.json")

		// PayeeParty should have PartyName (BR-17)
		assert.Equal(t, "Ebeneser Scrooge AS", doc.PayeeParty.PartyName.Name)
		assert.Equal(t, "2013-07-20", doc.DueDate)

		assert.Equal(t, "30", doc.PaymentMeans[0].PaymentMeansCode.Value)
		assert.Equal(t, "0003434323213231", *doc.PaymentMeans[0].PaymentID)
		assert.NotEmpty(t, doc.PaymentMeans[0].PayeeFinancialAccount)
		assert.Equal(t, "NO9386011117947", *doc.PaymentMeans[0].PayeeFinancialAccount.ID)
		assert.Equal(t, "DNBANOKK", *doc.PaymentMeans[0].PayeeFinancialAccount.FinancialInstitutionBranch.ID)
	})

	t.Run("credit transfer with account number", func(t *testing.T) {
		doc := testInvoiceFrom(t, "invoice-account-number.json")

		// Verify the account number was set in the UBL financial account ID
		assert.NotEmpty(t, doc.PaymentMeans[0].PayeeFinancialAccount)
		assert.Equal(t, "123456789", *doc.PaymentMeans[0].PayeeFinancialAccount.ID)
		assert.Equal(t, "Test Bank Account", *doc.PaymentMeans[0].PayeeFinancialAccount.Name)
		assert.Equal(t, "DNBANOKK", *doc.PaymentMeans[0].PayeeFinancialAccount.FinancialInstitutionBranch.ID)
	})

	t.Run("direct debit without mandate reference omits the mandate ID", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Payment.Instructions.CreditTransfer = nil
		inv.Payment.Instructions.DirectDebit = &pay.DirectDebit{Account: "0667"}

		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)
		require.NotEmpty(t, doc.PaymentMeans)

		// No reference means the mandate has no ID.
		require.NotNil(t, doc.PaymentMeans[0].PaymentMandate)
		assert.Nil(t, doc.PaymentMeans[0].PaymentMandate.ID)
		assert.Equal(t, "0667", *doc.PaymentMeans[0].PaymentMandate.PayerFinancialAccount.ID)
	})

	t.Run("direct debit with mandate reference includes the mandate", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Payment.Instructions.CreditTransfer = nil
		inv.Payment.Instructions.DirectDebit = &pay.DirectDebit{Ref: "MANDATE-123", Account: "0667"}

		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)
		require.NotEmpty(t, doc.PaymentMeans)
		require.NotNil(t, doc.PaymentMeans[0].PaymentMandate)
		require.NotNil(t, doc.PaymentMeans[0].PaymentMandate.ID)
		assert.Equal(t, "MANDATE-123", doc.PaymentMeans[0].PaymentMandate.ID.Value)
	})

	t.Run("direct debit places account inside mandate (BT-91)", func(t *testing.T) {
		env := loadTestEnvelope(t, "xrechnung/invoice-xr-minimal.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Payment = &bill.PaymentDetails{
			Instructions: &pay.Instructions{
				Key: pay.MeansKeyDirectDebit,
				DirectDebit: &pay.DirectDebit{
					Ref:      "MAND-2024-001",
					Creditor: "DE98ZZZ09999999999",
					Account:  "DE89370400440532013000",
				},
				Ext: tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyPaymentMeans: "49"}),
			},
		}

		doc, err := ubl.ConvertInvoice(env, ubl.WithContext(ubl.ContextXRechnung))
		require.NoError(t, err)

		pm := doc.PaymentMeans[0]
		assert.Equal(t, "49", pm.PaymentMeansCode.Value)
		assert.NotNil(t, pm.PaymentMandate)
		require.NotNil(t, pm.PaymentMandate.ID)
		assert.Equal(t, "MAND-2024-001", pm.PaymentMandate.ID.Value)
		assert.NotNil(t, pm.PaymentMandate.PayerFinancialAccount)
		assert.Equal(t, "DE89370400440532013000", *pm.PaymentMandate.PayerFinancialAccount.ID)
		assert.Nil(t, pm.PayerFinancialAccount, "BT-91 must not appear at PaymentMeans level (UBL-CR-680)")
	})

	t.Run("card payment includes the network ID required by the schema", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Payment.Instructions.CreditTransfer = nil
		inv.Payment.Instructions.Card = &pay.Card{Last4: "0312"}

		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)
		require.NotEmpty(t, doc.PaymentMeans)

		card := doc.PaymentMeans[0].CardAccount
		require.NotNil(t, card)
		require.NotNil(t, card.PrimaryAccountNumberID)
		assert.Equal(t, "0312", *card.PrimaryAccountNumberID)
		// cbc:NetworkID is mandatory in UBL, but has no EN 16931 business term.
		require.NotNil(t, card.NetworkID)
		assert.Equal(t, "NA", *card.NetworkID)
		assert.Nil(t, card.HolderName)

		data, err := ubl.Bytes(doc)
		require.NoError(t, err)
		assert.Contains(t, string(data), "<cac:CardAccount>\n      <cbc:PrimaryAccountNumberID>0312</cbc:PrimaryAccountNumberID>\n      <cbc:NetworkID>NA</cbc:NetworkID>\n    </cac:CardAccount>")
	})

	t.Run("document type extension", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")

		inv, ok := env.Extract().(*bill.Invoice)
		assert.True(t, ok)

		inv.Payment.Instructions.Ext = tax.MakeExtensions()

		_, err := ubl.ConvertInvoice(env)
		assert.ErrorContains(t, err, "instructions: (ext: (untdid-payment-means: required.).).")
	})

	t.Run("payee inbox maps to EndpointID", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-complete.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		if inv.Payment == nil {
			inv.Payment = &bill.PaymentDetails{}
		}
		inv.Payment.Payee = &org.Party{
			Name: "Test Payee",
			Inboxes: []*org.Inbox{
				{Email: "payee@example.com"},
			},
		}
		require.NoError(t, env.Calculate())

		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)

		require.NotNil(t, doc.PayeeParty)
		require.NotNil(t, doc.PayeeParty.EndpointID)
		assert.Equal(t, "EM", doc.PayeeParty.EndpointID.SchemeID)
		assert.Equal(t, "payee@example.com", doc.PayeeParty.EndpointID.Value)

		data, err := ubl.Bytes(doc)
		require.NoError(t, err)

		parsed, err := ubl.Parse(data)
		require.NoError(t, err)
		out, ok := parsed.(*ubl.Invoice)
		require.True(t, ok)
		outEnv, err := out.Convert()
		require.NoError(t, err)
		outInv, ok := outEnv.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.NotNil(t, outInv.Payment)
		require.NotNil(t, outInv.Payment.Payee)
		require.NotEmpty(t, outInv.Payment.Payee.Inboxes)
		assert.Equal(t, "payee@example.com", outInv.Payment.Payee.Inboxes[0].Email)
	})
}

func TestPaymentPayer(t *testing.T) {
	const fixture = "france-extended/invoice-payer.json"

	t.Run("french extended maps the payer to the payment mandate", func(t *testing.T) {
		doc, err := testInvoiceFromContext(fixture, ubl.ContextPeppolFranceExtended)
		require.NoError(t, err)

		require.NotEmpty(t, doc.PaymentMeans)
		mandate := doc.PaymentMeans[0].PaymentMandate
		require.NotNil(t, mandate)
		payer := mandate.PayerParty
		require.NotNil(t, payer)
		assert.Equal(t, "Payeur SA", payer.PartyName.Name)
		require.NotEmpty(t, payer.PartyIdentification)
		assert.Equal(t, "39183804200003", payer.PartyIdentification[0].ID.Value)
		assert.Equal(t, "0009", *payer.PartyIdentification[0].ID.SchemeID)
		require.NotNil(t, payer.PartyLegalEntity)
		assert.Equal(t, "391838042", payer.PartyLegalEntity.CompanyID.Value)
		assert.Equal(t, "0002", *payer.PartyLegalEntity.CompanyID.SchemeID)

		// The payee travels alongside the payer (BG-10).
		require.NotNil(t, doc.PayeeParty)
		assert.Equal(t, "Bénéficiaire SARL", doc.PayeeParty.PartyName.Name)
	})

	t.Run("payer without payment instructions synthesizes the payment means", func(t *testing.T) {
		env := loadTestEnvelope(t, fixture)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		inv.Payment.Instructions = nil

		doc, err := ubl.ConvertInvoice(env, ubl.WithContext(ubl.ContextPeppolFranceExtended))
		require.NoError(t, err)
		require.NotEmpty(t, doc.PaymentMeans)
		assert.Equal(t, "1", doc.PaymentMeans[0].PaymentMeansCode.Value)
		require.NotNil(t, doc.PaymentMeans[0].PaymentMandate)
		assert.Nil(t, doc.PaymentMeans[0].PaymentMandate.ID)
		assert.NotNil(t, doc.PaymentMeans[0].PaymentMandate.PayerParty)
	})

	t.Run("payer is ignored outside the french extended context", func(t *testing.T) {
		doc, err := testInvoiceFromContext(fixture, ubl.ContextPeppol)
		require.NoError(t, err)

		require.NotEmpty(t, doc.PaymentMeans)
		assert.Nil(t, doc.PaymentMeans[0].PaymentMandate)
	})

	t.Run("parse restores the payer without inventing a direct debit", func(t *testing.T) {
		doc, err := testInvoiceFromContext(fixture, ubl.ContextPeppolFranceExtended)
		require.NoError(t, err)
		data, err := ubl.Bytes(doc)
		require.NoError(t, err)

		parsed, err := ubl.Parse(data)
		require.NoError(t, err)
		in, ok := parsed.(*ubl.Invoice)
		require.True(t, ok)
		env, err := in.Convert()
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		require.NotNil(t, inv.Payment)
		require.NotNil(t, inv.Payment.Payer)
		assert.Equal(t, "Payeur SA", inv.Payment.Payer.Name)
		require.NotNil(t, inv.Payment.Payee)
		assert.Equal(t, "Bénéficiaire SARL", inv.Payment.Payee.Name)
		require.NotNil(t, inv.Payment.Instructions)
		assert.Nil(t, inv.Payment.Instructions.DirectDebit)
	})
}
