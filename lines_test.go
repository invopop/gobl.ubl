package ubl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLines(t *testing.T) {
	t.Run("invoice-without-buyers-tax-id.json", func(t *testing.T) {
		doc, err := testInvoiceFrom("invoice-without-buyers-tax-id.json")
		require.NoError(t, err)

		assert.NotNil(t, doc.InvoiceLines)
		assert.Len(t, doc.InvoiceLines, 1)
		assert.Equal(t, "1", doc.InvoiceLines[0].ID)
		assert.Equal(t, "1800.00", doc.InvoiceLines[0].LineExtensionAmount.Value)
		assert.Equal(t, "Development services", doc.InvoiceLines[0].Item.Name)
		assert.Equal(t, "HUR", doc.InvoiceLines[0].InvoicedQuantity.UnitCode)
		assert.Equal(t, "VAT", doc.InvoiceLines[0].Item.ClassifiedTaxCategory.TaxScheme.ID)
		assert.Equal(t, "19", *doc.InvoiceLines[0].Item.ClassifiedTaxCategory.Percent)
		assert.True(t, doc.InvoiceLines[0].AllowanceCharge[0].ChargeIndicator)
		assert.Equal(t, "Testing", *doc.InvoiceLines[0].AllowanceCharge[0].AllowanceChargeReason)
		assert.Equal(t, "12.00", doc.InvoiceLines[0].AllowanceCharge[0].Amount.Value)
		assert.False(t, doc.InvoiceLines[0].AllowanceCharge[1].ChargeIndicator)
		assert.Equal(t, "Damage", *doc.InvoiceLines[0].AllowanceCharge[1].AllowanceChargeReason)
		assert.Equal(t, "12.00", doc.InvoiceLines[0].AllowanceCharge[1].Amount.Value)
		assert.Equal(t, "0088", *doc.InvoiceLines[0].Item.StandardItemIdentification.ID.SchemeID)
		assert.Equal(t, "1234567890128", doc.InvoiceLines[0].Item.StandardItemIdentification.ID.Value)
	})

	t.Run("invoice-with-line-order.json", func(t *testing.T) {
		doc, err := testInvoiceFrom("invoice-with-line-order.json")
		require.NoError(t, err)

		assert.NotNil(t, doc.InvoiceLines)
		assert.Len(t, doc.InvoiceLines, 1)
		assert.Equal(t, "1", doc.InvoiceLines[0].ID)
		assert.NotNil(t, doc.InvoiceLines[0].OrderLineReference)
		assert.Equal(t, "123", doc.InvoiceLines[0].OrderLineReference.LineID)
		assert.Equal(t, "DEVSERV001", doc.InvoiceLines[0].Item.SellersItemIdentification.ID.Value)

		// First identity with extension maps to StandardItemIdentification
		assert.NotNil(t, doc.InvoiceLines[0].Item.StandardItemIdentification)
		assert.Equal(t, "0088", *doc.InvoiceLines[0].Item.StandardItemIdentification.ID.SchemeID)
		assert.Equal(t, "1234567890128", doc.InvoiceLines[0].Item.StandardItemIdentification.ID.Value)

		// First identity without extension maps to BuyersItemIdentification
		assert.NotNil(t, doc.InvoiceLines[0].Item.BuyersItemIdentification)
		assert.Nil(t, doc.InvoiceLines[0].Item.BuyersItemIdentification.ID.SchemeID)
		assert.Equal(t, "1234567890128", doc.InvoiceLines[0].Item.BuyersItemIdentification.ID.Value)
	})

	t.Run("invoice-zero-quantity.json", func(t *testing.T) {
		doc, err := testInvoiceFrom("invoice-zero-quantity.json")
		require.NoError(t, err)

		assert.NotNil(t, doc.InvoiceLines)
		assert.Len(t, doc.InvoiceLines, 1)
		assert.Equal(t, "1", doc.InvoiceLines[0].ID)
		assert.Equal(t, "0.00", doc.InvoiceLines[0].LineExtensionAmount.Value)
		assert.Equal(t, "Development services", doc.InvoiceLines[0].Item.Name)

		// Quantity should always be set, even when zero (mandatory field)
		assert.NotNil(t, doc.InvoiceLines[0].InvoicedQuantity)
		assert.Equal(t, "0", doc.InvoiceLines[0].InvoicedQuantity.Value)
		assert.Equal(t, "HUR", doc.InvoiceLines[0].InvoicedQuantity.UnitCode)
	})

	// BR-DEC-24 / UBL-DT-01: when prices_include=VAT, RemoveIncludedTaxes
	// leaves discount amounts with extra precision (1.76 / 1.21 = 1.4545).
	// The converter must round line allowance amounts to 2 decimals to satisfy
	// EN16931 / Peppol BIS 3.0, even though the item net price (BT-146) is
	// allowed to keep its higher precision.
	t.Run("invoice-prices-include-vat.json", func(t *testing.T) {
		doc, err := testInvoiceFrom("invoice-prices-include-vat.json")
		require.NoError(t, err)

		require.Len(t, doc.InvoiceLines, 2)

		l1 := doc.InvoiceLines[0]
		require.Len(t, l1.AllowanceCharge, 1)
		assert.False(t, l1.AllowanceCharge[0].ChargeIndicator)
		assert.Equal(t, "1.45", l1.AllowanceCharge[0].Amount.Value)
		assert.Equal(t, "8.1818", l1.Price.PriceAmount.Value)

		l2 := doc.InvoiceLines[1]
		require.Len(t, l2.AllowanceCharge, 1)
		assert.Equal(t, "11.02", l2.AllowanceCharge[0].Amount.Value)
		assert.Equal(t, "61.9008", l2.Price.PriceAmount.Value)
	})

}
