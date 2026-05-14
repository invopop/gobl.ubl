package ubl_test

import (
	"encoding/xml"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/xmldsig"
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

func TestDocumentSignaturesXML(t *testing.T) {
	ds := &ubl.DocumentSignatures{
		SIGNamespace: "urn:sig",
		SACNamespace: "urn:sac",
		SBCNamespace: "urn:sbc",
		SignatureInformation: &ubl.SignatureInformation{
			ID:                    ubl.SignatureInformationID,
			ReferencedSignatureID: ubl.ReferenceSignatureID,
			Signature:             &xmldsig.Signature{},
		},
	}

	out, err := xml.Marshal(ds)
	require.NoError(t, err)

	xmlStr := string(out)
	assert.Contains(t, xmlStr, `xmlns:sig="urn:sig"`)
	assert.Contains(t, xmlStr, `xmlns:sac="urn:sac"`)
	assert.Contains(t, xmlStr, `xmlns:sbc="urn:sbc"`)
	assert.Contains(t, xmlStr, "<sac:SignatureInformation>")
	assert.Contains(t, xmlStr, "<cbc:ID>"+ubl.SignatureInformationID+"</cbc:ID>")
	assert.Contains(t, xmlStr, "<sbc:ReferencedSignatureID>"+ubl.ReferenceSignatureID+"</sbc:ReferencedSignatureID>")
	assert.Contains(t, xmlStr, "<ds:Signature")
}

func TestSignatureInformationXMLOmitsNilSignature(t *testing.T) {
	si := &ubl.SignatureInformation{
		ID:                    "id-1",
		ReferencedSignatureID: "ref-1",
	}

	out, err := xml.Marshal(si)
	require.NoError(t, err)

	xmlStr := string(out)
	assert.Contains(t, xmlStr, "<cbc:ID>id-1</cbc:ID>")
	assert.Contains(t, xmlStr, "<sbc:ReferencedSignatureID>ref-1</sbc:ReferencedSignatureID>")
	assert.NotContains(t, xmlStr, "<ds:Signature")
}
