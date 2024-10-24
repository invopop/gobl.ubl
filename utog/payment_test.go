package utog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPayment(t *testing.T) {
	t.Run("ubl-example2.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("ubl-example2.xml")
		require.NoError(t, err)

		conversor := NewConversor()
		err = conversor.NewInvoice(doc)
		require.NoError(t, err)

		payment := conversor.GetInvoice().Payment
		require.NotNil(t, payment)

		require.NotNil(t, payment.Payee)
		assert.Equal(t, "Ebeneser Scrooge AS", payment.Payee.Name)
		assert.Equal(t, "NO", payment.Payee.TaxID.Country)
		assert.Equal(t, "989823401", payment.Payee.TaxID.Code)

		assert.Equal(t, "2013-06-10", payment.Terms.DueDates[0].Date.Format("2006-01-02"))
		assert.Equal(t, "30", payment.Instructions.Key)
		assert.Equal(t, "NO9386011117947", payment.Instructions.CreditTransfer[0].IBAN)
		assert.Equal(t, "DNBANOKK", payment.Instructions.CreditTransfer[0].BIC)
	})

	t.Run("ubl-example5.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("ubl-example5.xml")
		require.NoError(t, err)

		conversor := NewConversor()
		err = conversor.NewInvoice(doc)
		require.NoError(t, err)

		payment := conversor.GetInvoice().Payment
		require.NotNil(t, payment)

		assert.Equal(t, "Dagobert Duck", payment.Payee.Name)

		assert.Equal(t, "50% prepaid, 50% within one month", payment.Terms.Notes)
		assert.Equal(t, "49", payment.Instructions.Key)
		assert.Equal(t, "Payref1", payment.Instructions.Detail)
		assert.Equal(t, "123456", payment.Instructions.Mandate.ID)
		assert.Equal(t, "DK1212341234123412", payment.Instructions.Mandate.Account.IBAN)
	})

	t.Run("ubl-example7.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("ubl-example7.xml")
		require.NoError(t, err)

		conversor := NewConversor()
		err = conversor.NewInvoice(doc)
		require.NoError(t, err)

		payment := conversor.GetInvoice().Payment
		require.NotNil(t, payment)

		assert.Equal(t, "2013-04-10", payment.DueDate)
		assert.Equal(t, "30", payment.Terms)
		assert.Equal(t, "SE:BANKGIRO", payment.Details[0].Type)
		assert.Equal(t, "5896-7771", payment.Details[0].Number)
	})

	t.Run("ubl-example8.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("ubl-example8.xml")
		require.NoError(t, err)

		conversor := NewConversor()
		err = conversor.NewInvoice(doc)
		require.NoError(t, err)

		payment := conversor.GetInvoice().Payment
		require.NotNil(t, payment)

		assert.Equal(t, "2014-11-25", payment.DueDate)
		assert.Equal(t, "15", payment.Terms)
		assert.Equal(t, "NL:IBAN", payment.Details[0].Type)
		assert.Equal(t, "NL78RABO0106741292", payment.Details[0].Number)
		assert.Equal(t, "NL:BIC", payment.Details[1].Type)
		assert.Equal(t, "RABONL2U", payment.Details[1].Number)
	})

}
