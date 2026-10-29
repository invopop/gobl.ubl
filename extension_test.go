package ubl_test

import (
	"encoding/xml"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/xmldsig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExtension(t *testing.T) {
	ext := ubl.NewExtension()

	require.NotNil(t, ext)
	assert.Nil(t, ext.ExtensionURI)
	require.NotNil(t, ext.ExtensionContent)
	require.NotNil(t, ext.ExtensionContent.UBLDocumentSignatures)
	assert.Nil(t, ext.ExtensionContent.UBLDocumentSignatures.SignatureInformation)
}

func TestNewExtensionReturnsDistinctInstances(t *testing.T) {
	a := ubl.NewExtension()
	b := ubl.NewExtension()

	require.NotSame(t, a, b)
	require.NotSame(t, a.ExtensionContent, b.ExtensionContent)
	require.NotSame(t, a.ExtensionContent.UBLDocumentSignatures, b.ExtensionContent.UBLDocumentSignatures)
}

func TestAddExtension(t *testing.T) {
	t.Run("initializes nil Extensions", func(t *testing.T) {
		inv := &ubl.Invoice{}
		assert.Nil(t, inv.Extensions)

		ext := ubl.NewExtension()
		inv.AddExtension(ext)

		require.NotNil(t, inv.Extensions)
		require.Len(t, inv.Extensions.Extension, 1)
		assert.Equal(t, *ext, inv.Extensions.Extension[0])
	})

	t.Run("appends to existing Extensions", func(t *testing.T) {
		uri := "urn:existing"
		inv := &ubl.Invoice{
			Extensions: &ubl.Extensions{
				Extension: []ubl.Extension{
					{ExtensionURI: &uri},
				},
			},
		}

		inv.AddExtension(ubl.NewExtension())

		require.Len(t, inv.Extensions.Extension, 2)
		require.NotNil(t, inv.Extensions.Extension[0].ExtensionURI)
		assert.Equal(t, "urn:existing", *inv.Extensions.Extension[0].ExtensionURI)
		assert.Nil(t, inv.Extensions.Extension[1].ExtensionURI)
	})

	t.Run("multiple calls accumulate", func(t *testing.T) {
		inv := &ubl.Invoice{}
		inv.AddExtension(ubl.NewExtension())
		inv.AddExtension(ubl.NewExtension())
		inv.AddExtension(ubl.NewExtension())

		require.NotNil(t, inv.Extensions)
		assert.Len(t, inv.Extensions.Extension, 3)
	})

	t.Run("stores a copy of the extension", func(t *testing.T) {
		inv := &ubl.Invoice{}
		ext := ubl.NewExtension()
		inv.AddExtension(ext)

		uri := "urn:mutated"
		ext.ExtensionURI = &uri

		require.Len(t, inv.Extensions.Extension, 1)
		assert.Nil(t, inv.Extensions.Extension[0].ExtensionURI,
			"AddExtension should copy the extension by value, decoupling later mutations")
	})
}

func TestExtensionsXML(t *testing.T) {
	uri := "urn:test:extension"
	exts := &ubl.Extensions{
		Extension: []ubl.Extension{
			{
				ExtensionURI: &uri,
				ExtensionContent: &ubl.ExtensionContent{
					UBLDocumentSignatures: &ubl.DocumentSignatures{
						SIGNamespace: "urn:sig",
						SACNamespace: "urn:sac",
						SBCNamespace: "urn:sbc",
					},
				},
			},
		},
	}

	out, err := xml.Marshal(exts)
	require.NoError(t, err)

	xmlStr := string(out)
	assert.Contains(t, xmlStr, "<ext:UBLExtension>")
	assert.Contains(t, xmlStr, "<ext:ExtensionURI>urn:test:extension</ext:ExtensionURI>")
	assert.Contains(t, xmlStr, "<ext:ExtensionContent>")
	assert.Contains(t, xmlStr, "<sig:UBLDocumentSignatures")
}

func TestExtensionXMLOmitsEmpty(t *testing.T) {
	ext := ubl.Extension{}

	out, err := xml.Marshal(ext)
	require.NoError(t, err)

	xmlStr := string(out)
	assert.NotContains(t, xmlStr, "<ext:ExtensionURI>")
	assert.NotContains(t, xmlStr, "<ext:ExtensionContent>")
}

// signatureXML is the signature extension as gobl.ubl v0.99.0 writes it.
// Signed documents depend on these bytes.
const signatureXML = `<Extensions><ext:UBLExtension><ext:ExtensionURI>urn:oasis:names:specification:ubl:dsig:enveloped:xades</ext:ExtensionURI><ext:ExtensionContent><sig:UBLDocumentSignatures xmlns:sig="urn:oasis:names:specification:ubl:schema:xsd:CommonSignatureComponents-2" xmlns:sac="urn:oasis:names:specification:ubl:schema:xsd:SignatureAggregateComponents-2" xmlns:sbc="urn:oasis:names:specification:ubl:schema:xsd:SignatureBasicComponents-2"><sac:SignatureInformation><cbc:ID>urn:oasis:names:specification:ubl:signature:1</cbc:ID><sbc:ReferencedSignatureID>urn:oasis:names:specification:ubl:signature:Invoice</sbc:ReferencedSignatureID><ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#" Id="signature"><ds:SignedInfo><ds:Reference URI=""><ds:DigestValue>ZGlnZXN0</ds:DigestValue></ds:Reference></ds:SignedInfo><ds:SignatureValue Id="">c2lnbmF0dXJl</ds:SignatureValue></ds:Signature></sac:SignatureInformation></sig:UBLDocumentSignatures></ext:ExtensionContent></ext:UBLExtension></Extensions>`

// signatureExtension builds a signature extension with fixed values.
func signatureExtension() *ubl.Extension {
	uri := ubl.SignatureMethod
	sig := &xmldsig.Signature{
		DSigNamespace: xmldsig.NamespaceDSig,
		ID:            "signature",
		SignedInfo: &xmldsig.SignedInfo{
			Reference: []*xmldsig.Reference{{URI: "", DigestValue: "ZGlnZXN0"}},
		},
		Value: &xmldsig.Value{Value: "c2lnbmF0dXJl"},
	}
	ext := ubl.NewExtension()
	ext.ExtensionURI = &uri
	ext.ExtensionContent.UBLDocumentSignatures.SIGNamespace = ubl.NamespaceSIG
	ext.ExtensionContent.UBLDocumentSignatures.SACNamespace = ubl.NamespaceSAC
	ext.ExtensionContent.UBLDocumentSignatures.SBCNamespace = ubl.NamespaceSBC
	ext.ExtensionContent.UBLDocumentSignatures.SignatureInformation = &ubl.SignatureInformation{
		ID:                    ubl.SignatureInformationID,
		ReferencedSignatureID: ubl.ReferenceSignatureID,
		Signature:             sig,
	}
	return ext
}

func TestSignatureExtensionXML(t *testing.T) {
	inv := new(ubl.Invoice)
	inv.AddExtension(signatureExtension())
	out, err := xml.Marshal(inv.Extensions)
	require.NoError(t, err)
	assert.Equal(t, signatureXML, string(out))
}
