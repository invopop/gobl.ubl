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
