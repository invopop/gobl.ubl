package ubl_test

import (
	"encoding/xml"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
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

func TestExtensionContentRaw(t *testing.T) {
	t.Run("written verbatim", func(t *testing.T) {
		const content = `<x:Data xmlns:x="urn:test:x"><x:Flag>yes</x:Flag></x:Data>`
		exts := &ubl.Extensions{
			Extension: []ubl.Extension{
				{ExtensionContent: &ubl.ExtensionContent{Raw: []byte(content)}},
			},
		}
		out, err := xml.Marshal(exts)
		require.NoError(t, err)
		assert.Equal(t, "<Extensions><ext:UBLExtension><ext:ExtensionContent>"+content+"</ext:ExtensionContent></ext:UBLExtension></Extensions>", string(out))
	})

	t.Run("read on decode", func(t *testing.T) {
		data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
	xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"
	xmlns:ext="urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2"
	xmlns:x="urn:test:x">
	<ext:UBLExtensions>
		<ext:UBLExtension>
			<ext:ExtensionContent><x:Data><x:Flag>root</x:Flag></x:Data></ext:ExtensionContent>
		</ext:UBLExtension>
		<ext:UBLExtension>
			<ext:ExtensionContent><y:Data xmlns:y="urn:test:x"><y:Flag>local</y:Flag></y:Data></ext:ExtensionContent>
		</ext:UBLExtension>
	</ext:UBLExtensions>
	<cbc:ID>1</cbc:ID>
</Invoice>`)
		doc, err := ubl.Decode(data)
		require.NoError(t, err)
		inv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)
		require.NotNil(t, inv.Extensions)
		require.Len(t, inv.Extensions.Extension, 2)

		for i, want := range []string{"root", "local"} {
			ec := inv.Extensions.Extension[i].ExtensionContent
			require.NotNil(t, ec)
			var x struct {
				XMLName xml.Name `xml:"urn:test:x Data"`
				Flag    string   `xml:"urn:test:x Flag"`
			}
			require.NoError(t, xml.Unmarshal(ec.Raw, &x), string(ec.Raw))
			assert.Equal(t, want, x.Flag)
		}
	})
}
