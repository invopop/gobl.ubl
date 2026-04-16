package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
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
		assert.Equal(t, "380", out.InvoiceTypeCode.Value)

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

	t.Run("tax point conversion", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-complete.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		tests := []struct {
			name string
			key  cbc.Key
			code string
		}{
			{"issue", tax.PointIssue, "3"},
			{"delivery", tax.PointDelivery, "35"},
			{"payment", tax.PointPayment, "432"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				inv.Tax.Point = tt.key
				out, err := ubl.ConvertInvoice(env)
				require.NoError(t, err)
				require.Len(t, out.InvoicePeriod, 1)
				assert.Equal(t, tt.code, out.InvoicePeriod[0].DescriptionCode)
			})
		}

		t.Run("unknown key ignored", func(t *testing.T) {
			inv.Tax.Point = "unknown"
			out, err := ubl.ConvertInvoice(env)
			require.NoError(t, err)
			// Period still present from ordering data, but no DescriptionCode
			if len(out.InvoicePeriod) > 0 {
				assert.Empty(t, out.InvoicePeriod[0].DescriptionCode)
			}
		})

		t.Run("nil tax", func(t *testing.T) {
			inv.Tax = nil
			out, err := ubl.ConvertInvoice(env)
			// Tax is required for document type, so this will error
			assert.Error(t, err)
			assert.Nil(t, out)
		})
	})

	t.Run("tax point round trip", func(t *testing.T) {
		tests := []struct {
			name string
			key  cbc.Key
			code string
		}{
			{"issue", tax.PointIssue, "3"},
			{"delivery", tax.PointDelivery, "35"},
			{"payment", tax.PointPayment, "432"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				env, err := loadTestEnvelope("invoice-complete.json")
				require.NoError(t, err)

				inv, ok := env.Extract().(*bill.Invoice)
				require.True(t, ok)

				inv.Tax.Point = tt.key
				out, err := ubl.ConvertInvoice(env)
				require.NoError(t, err)
				require.Len(t, out.InvoicePeriod, 1)
				assert.Equal(t, tt.code, out.InvoicePeriod[0].DescriptionCode)

				// Parse back and verify round-trip
				parsed, err := out.Convert()
				require.NoError(t, err)
				parsedInv, ok := parsed.Extract().(*bill.Invoice)
				require.True(t, ok)
				assert.Equal(t, tt.key, parsedInv.Tax.Point)
			})
		}
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
