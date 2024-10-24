package utog

import (
	"testing"

	"github.com/invopop/gobl/cbc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInstructions(t *testing.T) {
	t.Run("ubl-example2.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("ubl-example2.xml")
		require.NoError(t, err)

		conversor := NewConversor()
		err = conversor.NewInvoice(doc)
		require.NoError(t, err)

		payment := conversor.GetInvoice().Payment
		require.NotNil(t, payment)

		assert.Equal(t, cbc.Key("credit-transfer"), payment.Instructions.Key)
		assert.Equal(t, "NO9386011117947", payment.Instructions.CreditTransfer[0].IBAN)
		assert.Equal(t, "DNBANOKK", payment.Instructions.CreditTransfer[0].BIC)
		assert.Equal(t, "0003434323213231", payment.Instructions.Ref)
		assert.Equal(t, "2 % discount if paid within 2 days\n            Penalty percentage 10% from due date", payment.Terms.Notes)
	})

	t.Run("ubl-example5.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("ubl-example5.xml")
		require.NoError(t, err)

		conversor := NewConversor()
		err = conversor.NewInvoice(doc)
		require.NoError(t, err)

		payment := conversor.GetInvoice().Payment
		require.NotNil(t, payment)

		assert.Equal(t, cbc.Key("direct-debit"), payment.Instructions.Key)
		assert.Equal(t, "Payref1", payment.Instructions.Ref)
		assert.Equal(t, "123456", payment.Instructions.DirectDebit.Ref)
		assert.Equal(t, "DK1212341234123412", payment.Instructions.DirectDebit.Account)
	})

	t.Run("ubl-example7.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("ubl-example7.xml")
		require.NoError(t, err)

		conversor := NewConversor()
		err = conversor.NewInvoice(doc)
		require.NoError(t, err)

		payment := conversor.GetInvoice().Payment
		require.NotNil(t, payment)

		assert.Equal(t, cbc.Key("credit-transfer"), payment.Instructions.Key)
		assert.Equal(t, "SEXDABCD", payment.Instructions.CreditTransfer[0].BIC)
		assert.Equal(t, "SE1212341234123412", payment.Instructions.CreditTransfer[0].IBAN)
		assert.Equal(t, "Payment within 30 days", payment.Terms.Notes)
	})

	t.Run("ubl-example8.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("ubl-example8.xml")
		require.NoError(t, err)

		conversor := NewConversor()
		err = conversor.NewInvoice(doc)
		require.NoError(t, err)

		payment := conversor.GetInvoice().Payment
		require.NotNil(t, payment)

		assert.Equal(t, cbc.Key("credit-transfer"), payment.Instructions.Key)
		assert.Equal(t, "1100512149", payment.Instructions.Ref)
		assert.Equal(t, "NL28RBOS0420242228", payment.Instructions.CreditTransfer[0].IBAN)
		assert.Equal(t, "Enexis brengt wettelijke rente in rekening over te laat betaalde\n            facturen. Kijk voor informatie op www.enexis.nl/rentenota", payment.Terms.Notes)
	})

}
