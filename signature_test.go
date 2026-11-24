package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignatureConstants(t *testing.T) {
	assert.Equal(t, "urn:oasis:names:specification:ubl:signature:1", ubl.SignatureInformationID)
	assert.Equal(t, "urn:oasis:names:specification:ubl:signature:Invoice", ubl.ReferenceSignatureID)
	assert.Equal(t, "urn:oasis:names:specification:ubl:dsig:enveloped:xades", ubl.SignatureMethod)
}

func TestAddSignatureReference(t *testing.T) {
	t.Run("appends to empty slice", func(t *testing.T) {
		inv := &ubl.Invoice{}
		inv.AddSignatureReference(ubl.SignatureMethod, ubl.ReferenceSignatureID)

		require.Len(t, inv.Signature, 1)
		assert.Equal(t, ubl.ReferenceSignatureID, inv.Signature[0].ID)
		require.NotNil(t, inv.Signature[0].SignatureMethod)
		assert.Equal(t, ubl.SignatureMethod, *inv.Signature[0].SignatureMethod)
	})

	t.Run("appends to existing slice", func(t *testing.T) {
		existingMethod := "existing-method"
		inv := &ubl.Invoice{
			Signature: []ubl.Signature{
				{ID: "existing-id", SignatureMethod: &existingMethod},
			},
		}
		inv.AddSignatureReference(ubl.SignatureMethod, ubl.ReferenceSignatureID)

		require.Len(t, inv.Signature, 2)
		assert.Equal(t, "existing-id", inv.Signature[0].ID)
		assert.Equal(t, ubl.ReferenceSignatureID, inv.Signature[1].ID)
		require.NotNil(t, inv.Signature[1].SignatureMethod)
		assert.Equal(t, ubl.SignatureMethod, *inv.Signature[1].SignatureMethod)
	})

	t.Run("preserves custom arguments", func(t *testing.T) {
		inv := &ubl.Invoice{}
		inv.AddSignatureReference("custom-method", "custom-ref")

		require.Len(t, inv.Signature, 1)
		assert.Equal(t, "custom-ref", inv.Signature[0].ID)
		require.NotNil(t, inv.Signature[0].SignatureMethod)
		assert.Equal(t, "custom-method", *inv.Signature[0].SignatureMethod)
	})

	t.Run("multiple calls accumulate", func(t *testing.T) {
		inv := &ubl.Invoice{}
		inv.AddSignatureReference("m1", "r1")
		inv.AddSignatureReference("m2", "r2")
		inv.AddSignatureReference("m3", "r3")

		require.Len(t, inv.Signature, 3)
		assert.Equal(t, "r1", inv.Signature[0].ID)
		assert.Equal(t, "r2", inv.Signature[1].ID)
		assert.Equal(t, "r3", inv.Signature[2].ID)
	})
}
