package gtou

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCharges(t *testing.T) {
	t.Run("invoice-complete.json", func(t *testing.T) {
		doc, err := newDocumentFrom("invoice-complete.json")
		require.NoError(t, err)

		assert.Len(t, doc.AllowanceCharge, 2)

		assert.True(t, doc.AllowanceCharge[0].ChargeIndicator)
		assert.Equal(t, "11.00", doc.AllowanceCharge[0].Amount.Value)
		assert.Equal(t, "Freight", *doc.AllowanceCharge[0].AllowanceChargeReason)

		assert.False(t, doc.AllowanceCharge[1].ChargeIndicator)
		assert.Equal(t, "88", *doc.AllowanceCharge[1].AllowanceChargeReasonCode)
		assert.Equal(t, "10.00", doc.AllowanceCharge[1].Amount.Value)
		assert.Equal(t, "Promotion discount", *doc.AllowanceCharge[1].AllowanceChargeReason)
	})
}
