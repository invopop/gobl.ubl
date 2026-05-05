package ubl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewOrdering(t *testing.T) {
	t.Run("invoice-minimal.json", func(t *testing.T) {
		doc := testInvoiceFrom(t, "invoice-minimal.json")

		assert.Equal(t, "", doc.BuyerReference)
		assert.NotNil(t, doc.OrderReference)
		assert.Equal(t, "NA", doc.OrderReference.ID)
	})

}
