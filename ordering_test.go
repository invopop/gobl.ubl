package ubl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrdering(t *testing.T) {
	t.Run("invoice-minimal.json", func(t *testing.T) {
		doc, err := testInvoiceFrom("invoice-minimal.json")
		require.NoError(t, err)

		assert.Equal(t, "", doc.BuyerReference)
		assert.NotNil(t, doc.OrderReference)
		assert.Equal(t, "NA", doc.OrderReference.ID)
	})

}
