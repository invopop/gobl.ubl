package gtou

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLines(t *testing.T) {
	t.Run("invoice-without-buyers-tax-id.json", func(t *testing.T) {
		doc, err := newDocumentFrom("invoice-without-buyers-tax-id.json")
		require.NoError(t, err)

		assert.NotNil(t, doc.InvoiceLine)
		assert.Len(t, doc.InvoiceLine, 1)
		assert.Equal(t, "1", doc.InvoiceLine[0].ID)
		assert.Equal(t, "1800.00", doc.InvoiceLine[0].LineExtensionAmount.Value)
		assert.Equal(t, "Development services", doc.InvoiceLine[0].Item.Name)
		assert.Equal(t, "HUR", doc.InvoiceLine[0].InvoicedQuantity.UnitCode)
		assert.Equal(t, "VAT", doc.InvoiceLine[0].Item.ClassifiedTaxCategory.TaxScheme.ID)
		assert.Equal(t, "19", *doc.InvoiceLine[0].Item.ClassifiedTaxCategory.Percent)
		assert.True(t, doc.InvoiceLine[0].AllowanceCharge[0].ChargeIndicator)
		assert.Equal(t, "Testing", *doc.InvoiceLine[0].AllowanceCharge[0].AllowanceChargeReason)
		assert.Equal(t, "12.00", doc.InvoiceLine[0].AllowanceCharge[0].Amount.Value)
		assert.False(t, doc.InvoiceLine[0].AllowanceCharge[1].ChargeIndicator)
		assert.Equal(t, "Damage", *doc.InvoiceLine[0].AllowanceCharge[1].AllowanceChargeReason)
		assert.Equal(t, "12.00", doc.InvoiceLine[0].AllowanceCharge[1].Amount.Value)

	})

}
