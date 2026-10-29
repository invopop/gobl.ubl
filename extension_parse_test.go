package ubl_test

import (
	"encoding/xml"
	"strings"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	ubl.RegisterExtensionData("urn:example:ext", func() ubl.ExtensionData { return new(testData) })
}

// extensionsXML holds one extension per decode case: registered content
// whose prefixes are declared on the root, the signature, unregistered
// content, two elements in one content, and no content.
const extensionsXML = `<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
  xmlns:ext="urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2"
  xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"
  xmlns:sig="urn:oasis:names:specification:ubl:schema:xsd:CommonSignatureComponents-2"
  xmlns:sac="urn:oasis:names:specification:ubl:schema:xsd:SignatureAggregateComponents-2"
  xmlns:ex="urn:example:ext" xmlns:un="urn:example:unregistered">
  <ext:UBLExtensions>
    <ext:UBLExtension><ext:ExtensionContent>
      <ex:Data><ex:Code>A1</ex:Code><cbc:Note>kept</cbc:Note></ex:Data>
    </ext:ExtensionContent></ext:UBLExtension>
    <ext:UBLExtension><ext:ExtensionContent>
      <sig:UBLDocumentSignatures><sac:SignatureInformation><cbc:ID>urn:oasis:names:specification:ubl:signature:1</cbc:ID></sac:SignatureInformation></sig:UBLDocumentSignatures>
    </ext:ExtensionContent></ext:UBLExtension>
    <ext:UBLExtension><ext:ExtensionContent>
      <un:Other><un:Value>7</un:Value></un:Other>
    </ext:ExtensionContent></ext:UBLExtension>
    <ext:UBLExtension><ext:ExtensionContent>
      <un:First/><un:Second/>
    </ext:ExtensionContent></ext:UBLExtension>
    <ext:UBLExtension><ext:ExtensionContent/></ext:UBLExtension>
  </ext:UBLExtensions>
  <cbc:ID>1</cbc:ID>
</Invoice>`

// decodeInvoice decodes data as an invoice with five extensions.
func decodeInvoice(t *testing.T, data []byte) *ubl.Invoice {
	t.Helper()
	doc, err := ubl.Decode(data)
	require.NoError(t, err)
	inv, ok := doc.(*ubl.Invoice)
	require.True(t, ok)
	require.NotNil(t, inv.Extensions)
	require.Len(t, inv.Extensions.Extension, 5)
	return inv
}

// extensionData returns the content of each extension of the invoice.
func extensionData(t *testing.T, inv *ubl.Invoice) []ubl.ExtensionData {
	t.Helper()
	out := make([]ubl.ExtensionData, len(inv.Extensions.Extension))
	for i, ext := range inv.Extensions.Extension {
		require.NotNil(t, ext.ExtensionContent, "extension %d", i)
		out[i] = ext.ExtensionContent.Data
	}
	return out
}

func TestDecodeExtensionContent(t *testing.T) {
	data := extensionData(t, decodeInvoice(t, []byte(extensionsXML)))

	assert.Equal(t, &testData{
		XMLName:   xml.Name{Space: "urn:example:ext", Local: "Data"},
		Namespace: "urn:example:ext",
		Code:      "A1",
		Note:      "kept",
	}, data[0])

	sigs, ok := data[1].(*ubl.DocumentSignatures)
	require.True(t, ok, "got %T", data[1])
	assert.Equal(t, ubl.NamespaceSIG, sigs.SIGNamespace)
	assert.Equal(t, ubl.NamespaceSAC, sigs.SACNamespace)
	require.NotNil(t, sigs.SignatureInformation)
	assert.Equal(t, ubl.SignatureInformationID, sigs.SignatureInformation.ID)

	raw, ok := data[2].(*ubl.RawExtensionData)
	require.True(t, ok, "got %T", data[2])
	assert.Equal(t, `<Other xmlns="urn:example:unregistered"><Value xmlns="urn:example:unregistered">7</Value></Other>`, string(raw.XML))

	raw, ok = data[3].(*ubl.RawExtensionData)
	require.True(t, ok, "got %T", data[3])
	assert.Equal(t, `<First xmlns="urn:example:unregistered"></First><Second xmlns="urn:example:unregistered"></Second>`, string(raw.XML))

	assert.Nil(t, data[4])
}

func TestDecodeExtensionContentNotSingle(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    ubl.ExtensionData
	}{
		{
			name:    "two registered elements",
			content: `<ex:Data><ex:Code>A1</ex:Code></ex:Data><ex:Data><ex:Code>B2</ex:Code></ex:Data>`,
			want:    &ubl.RawExtensionData{XML: []byte(`<Data xmlns="urn:example:ext"><Code xmlns="urn:example:ext">A1</Code></Data><Data xmlns="urn:example:ext"><Code xmlns="urn:example:ext">B2</Code></Data>`)},
		},
		{
			name:    "registered element and text",
			content: `<ex:Data><ex:Code>A1</ex:Code></ex:Data>note`,
			want:    &ubl.RawExtensionData{XML: []byte(`<Data xmlns="urn:example:ext"><Code xmlns="urn:example:ext">A1</Code></Data>note`)},
		},
		{
			name:    "text only",
			content: `note`,
			want:    &ubl.RawExtensionData{XML: []byte(`note`)},
		},
		{
			name:    "comment only",
			content: "<!-- none -->\n",
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ec := new(ubl.ExtensionContent)
			data := `<ExtensionContent xmlns:ex="urn:example:ext">` + tt.content + `</ExtensionContent>`
			require.NoError(t, xml.Unmarshal([]byte(data), ec))
			assert.Equal(t, tt.want, ec.Data)
		})
	}
}

func TestExtensionContentRoundTrip(t *testing.T) {
	first := decodeInvoice(t, []byte(extensionsXML))
	out, err := ubl.Encode(first)
	require.NoError(t, err)
	second := decodeInvoice(t, out)

	assert.Equal(t, extensionData(t, first), extensionData(t, second))
}

func TestRawExtensionDataXML(t *testing.T) {
	out, err := ubl.Encode(decodeInvoice(t, []byte(extensionsXML)))
	require.NoError(t, err)
	assert.NotContains(t, string(out), "&#x9;")
	assert.NotContains(t, string(out), "&#xA;")
	assert.NotContains(t, string(out), "_xmlns")

	_, err = xml.Marshal(ubl.NewExtension(&ubl.RawExtensionData{XML: []byte("<a>")}))
	assert.Error(t, err)
}

func TestDecodeExtensionContentError(t *testing.T) {
	data := strings.Replace(extensionsXML,
		`<ex:Data><ex:Code>A1</ex:Code><cbc:Note>kept</cbc:Note></ex:Data>`,
		`<ex:Data><ex:Count>x</ex:Count></ex:Data>`, 1)
	require.NotEqual(t, extensionsXML, data)

	_, err := ubl.Decode([]byte(data))
	assert.ErrorContains(t, err, "urn:example:ext")
}
