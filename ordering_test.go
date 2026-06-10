package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrdering(t *testing.T) {
	t.Run("invoice-minimal.json", func(t *testing.T) {
		doc := testInvoiceFrom(t, "invoice-minimal.json")

		assert.Equal(t, "", doc.BuyerReference)
		assert.NotNil(t, doc.OrderReference)
		assert.Equal(t, "NA", doc.OrderReference.ID)
	})

	// BT-12 (contract reference) plus the France CTC/Chorus Pro contract type
	// (EXT-FR-FE-01) must both be serialized into ContractDocumentReference.
	t.Run("contract type maps to cbc:DocumentType", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-complete.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.Ordering.Contracts = []*org.DocumentRef{
			{Code: "test nr de marche", Type: "marche"},
		}

		require.NoError(t, env.Calculate())
		doc, err := ubl.ConvertInvoice(env)
		require.NoError(t, err)

		require.Len(t, doc.ContractDocumentReference, 1)
		ref := doc.ContractDocumentReference[0]
		assert.Equal(t, "test nr de marche", ref.ID.Value)
		assert.Equal(t, "marche", ref.DocumentType)
	})
}
