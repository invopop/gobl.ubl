package ubl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrdering(t *testing.T) {
	t.Run("invoice-de-de.json", func(t *testing.T) {
		doc, err := testInvoiceFrom("invoice-de-de.json")
		require.NoError(t, err)

		assert.Equal(t, "PO4711", doc.BuyerReference)
		assert.Equal(t, "2013-03-10", *doc.InvoicePeriod[0].StartDate)
		assert.Equal(t, "2013-04-10", *doc.InvoicePeriod[0].EndDate)
		assert.Equal(t, "2013-05", doc.ContractDocumentReference[0].ID.Value)
		assert.Equal(t, "PO4711", doc.OrderReference.ID)
		assert.Equal(t, "3544", doc.ReceiptDocumentReference[0].ID.Value)
		assert.Equal(t, "5433", doc.DespatchDocumentReference[0].ID.Value)

	})

	t.Run("invoice-minimal.json", func(t *testing.T) {
		doc, err := testInvoiceFrom("invoice-minimal.json")
		require.NoError(t, err)

		assert.Equal(t, "", doc.BuyerReference)
		assert.NotNil(t, doc.OrderReference)
		assert.Equal(t, "NA", doc.OrderReference.ID)
	})

}
