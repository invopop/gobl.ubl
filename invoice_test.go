package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvoiceHeaders(t *testing.T) {
	t.Run("document type extension", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-complete.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		assert.True(t, ok)

		out, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)

		assert.NoError(t, err)
		assert.Equal(t, "380", out.InvoiceTypeCode)

		inv.Tax = nil
		_, err = ubl.ConvertInvoice(env)
		assert.ErrorContains(t, err, "tax: (ext: (untdid-document-type: required.).).")

		inv.Tax = &bill.Tax{
			Ext: tax.Extensions{},
		}
		_, err = ubl.ConvertInvoice(env)
		assert.ErrorContains(t, err, "ext: (untdid-document-type: required.).")
	})

	t.Run("format date", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-complete.json")
		require.NoError(t, err)

		out, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)
		assert.Equal(t, "2024-02-13", out.IssueDate)
	})

	t.Run("invoice number", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-complete.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		assert.True(t, ok)
		out, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)
		assert.Equal(t, "SAMPLE-001", out.ID)

		inv.Series = ""
		out, err = ubl.ConvertInvoice(env)
		require.NoError(t, err)
		assert.Equal(t, "001", out.ID)
	})
}
