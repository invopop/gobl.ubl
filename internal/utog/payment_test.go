package utog

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPayment(t *testing.T) {
	t.Run("ubl-example2.xml", func(t *testing.T) {
		e, err := newDocumentFrom("ubl-example2.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		payment := inv.Payment
		require.NotNil(t, payment)

		require.NotNil(t, payment.Payee)
		assert.Equal(t, "Ebeneser Scrooge AS", payment.Payee.Name)

		require.Len(t, payment.Payee.Identities, 2)
		assert.Equal(t, "CompanyID", payment.Payee.Identities[0].Label)
		assert.Equal(t, cbc.Code("989823401"), payment.Payee.Identities[0].Code)
		assert.Equal(t, "0088", payment.Payee.Identities[1].Label)
		assert.Equal(t, cbc.Code("2298740918237"), payment.Payee.Identities[1].Code)
		assert.Equal(t, "2 % discount if paid within 2 days\n            Penalty percentage 10% from due date", payment.Terms.Notes)
	})

	t.Run("ubl-example5.xml", func(t *testing.T) {
		e, err := newDocumentFrom("ubl-example5.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		payment := inv.Payment
		require.NotNil(t, payment)

		assert.Equal(t, "Dagobert Duck", payment.Payee.Name)
		assert.Equal(t, cbc.Code("DK16356608"), payment.Payee.Identities[0].Code)
		assert.Equal(t, "CompanyID", payment.Payee.Identities[0].Label)

	})
}
