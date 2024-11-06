package gtou

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPayment(t *testing.T) {
	t.Run("self-billed-invoice", func(t *testing.T) {
		env, err := LoadTestEnvelope("self-billed-invoice.json")
		require.NoError(t, err)

		inv := env.Extract().(*bill.Invoice)

		converter := NewConverter()
		err = converter.newDocument(inv)
		require.NoError(t, err)

		doc := converter.GetDocument()

		assert.Equal(t, "Ebeneser Scrooge AS", *doc.PayeeParty.PartyLegalEntity.RegistrationName)
		assert.Equal(t, "2013-07-20", doc.DueDate)

		assert.Equal(t, "30", doc.PaymentMeans[0].PaymentMeansCode.Value)
		assert.Equal(t, "0003434323213231", *doc.PaymentMeans[0].PaymentID)
		assert.NotEmpty(t, doc.PaymentMeans[0].PayeeFinancialAccount)
		assert.Equal(t, "NO9386011117947", *doc.PaymentMeans[0].PayeeFinancialAccount.ID)
		assert.Equal(t, "DNBANOKK", *doc.PaymentMeans[0].PayeeFinancialAccount.FinancialInstitutionBranch.ID)

	})

}
