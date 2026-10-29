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
	d := &ubl.DocumentSignatures{}
	ext := ubl.NewExtension(d)

	require.NotNil(t, ext)
	assert.Nil(t, ext.ExtensionURI)
	require.NotNil(t, ext.ExtensionContent)
	assert.Same(t, d, ext.ExtensionContent.Data)
}

func TestAddExtension(t *testing.T) {
	t.Run("initializes nil Extensions", func(t *testing.T) {
		inv := &ubl.Invoice{}
		assert.Nil(t, inv.Extensions)

		ext := ubl.NewExtension(&ubl.DocumentSignatures{})
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

		inv.AddExtension(ubl.NewExtension(&ubl.DocumentSignatures{}))

		require.Len(t, inv.Extensions.Extension, 2)
		require.NotNil(t, inv.Extensions.Extension[0].ExtensionURI)
		assert.Equal(t, "urn:existing", *inv.Extensions.Extension[0].ExtensionURI)
		assert.Nil(t, inv.Extensions.Extension[1].ExtensionURI)
	})

	t.Run("multiple calls accumulate", func(t *testing.T) {
		inv := &ubl.Invoice{}
		inv.AddExtension(ubl.NewExtension(&ubl.DocumentSignatures{}))
		inv.AddExtension(ubl.NewExtension(&ubl.DocumentSignatures{}))
		inv.AddExtension(ubl.NewExtension(&ubl.DocumentSignatures{}))

		require.NotNil(t, inv.Extensions)
		assert.Len(t, inv.Extensions.Extension, 3)
	})

	t.Run("stores a copy of the extension", func(t *testing.T) {
		inv := &ubl.Invoice{}
		ext := ubl.NewExtension(&ubl.DocumentSignatures{})
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
					Data: &ubl.DocumentSignatures{
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
	sigs := &ubl.DocumentSignatures{
		SIGNamespace: ubl.NamespaceSIG,
		SACNamespace: ubl.NamespaceSAC,
		SBCNamespace: ubl.NamespaceSBC,
		SignatureInformation: &ubl.SignatureInformation{
			ID:                    ubl.SignatureInformationID,
			ReferencedSignatureID: ubl.ReferenceSignatureID,
			Signature:             sig,
		},
	}
	ext := ubl.NewExtension(sigs)
	ext.ExtensionURI = &uri
	return ext
}

func TestSignatureExtensionXML(t *testing.T) {
	inv := new(ubl.Invoice)
	inv.AddExtension(signatureExtension())
	out, err := xml.Marshal(inv.Extensions)
	require.NoError(t, err)
	assert.Equal(t, signatureXML, string(out))
}

type testData struct {
	XMLName   xml.Name `xml:"ex:Data"`
	Namespace string   `xml:"xmlns:ex,attr"`
	Code      string   `xml:"ex:Code,omitempty"`
	Count     int      `xml:"ex:Count,omitempty"`
	Note      string   `xml:"cbc:Note,omitempty"`
}

func (*testData) ExtensionNamespaces() map[string]string {
	return map[string]string{"ex": "urn:example:ext", "cbc": ubl.NamespaceCBC}
}

func TestFindExtensionData(t *testing.T) {
	_, ok := ubl.FindExtensionData[*ubl.DocumentSignatures](nil)
	assert.False(t, ok)
	d := &testData{Code: "A1"}
	exts := &ubl.Extensions{Extension: []ubl.Extension{{}, *ubl.NewExtension(d), *signatureExtension()}}
	got, ok := ubl.FindExtensionData[*testData](exts)
	require.True(t, ok)
	assert.Same(t, d, got)
	_, ok = ubl.FindExtensionData[*ubl.DocumentSignatures](&ubl.Extensions{Extension: []ubl.Extension{*ubl.NewExtension(d)}})
	assert.False(t, ok)
}
