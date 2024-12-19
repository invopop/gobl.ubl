package gtou

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvoiceHeaders(t *testing.T) {
	t.Run("document type extension", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-de-de.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		assert.True(t, ok)

		code, err := getTypeCode(inv)
		assert.NoError(t, err)
		assert.Equal(t, "380", code)

		inv.Tax = nil
		_, err = getTypeCode(inv)
		assert.ErrorContains(t, err, "tax: (ext: (untdid-document-type: required.).).")

		inv.Tax = &bill.Tax{
			Ext: tax.Extensions{},
		}
		_, err = getTypeCode(inv)
		assert.ErrorContains(t, err, "ext: (untdid-document-type: required.).")
	})

	t.Run("format date", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-de-de.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		assert.True(t, ok)

		date := formatDate(inv.IssueDate)
		assert.Equal(t, "2024-02-13", date)
	})

	t.Run("invoice number", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-de-de.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		assert.True(t, ok)

		number := invoiceNumber(inv.Series, inv.Code)
		assert.Equal(t, "SAMPLE-001", number)

		inv.Series = ""
		number = invoiceNumber(inv.Series, inv.Code)
		assert.Equal(t, "001", number)
	})
}
