package ubl_test

import (
	"encoding/xml"
	"strings"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/xmldsig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The types below stand for what a converter defines for its own extension:
// gobl.ubl has no content types.

// documentSignatures is the content of a signature extension.
type documentSignatures struct {
	XMLName              xml.Name               `xml:"sig:UBLDocumentSignatures"`
	SIGNamespace         string                 `xml:"xmlns:sig,attr,omitempty"`
	SACNamespace         string                 `xml:"xmlns:sac,attr,omitempty"`
	SBCNamespace         string                 `xml:"xmlns:sbc,attr,omitempty"`
	SignatureInformation []signatureInformation `xml:"sac:SignatureInformation"`
}

type signatureInformation struct {
	ID                    string             `xml:"cbc:ID"`
	ReferencedSignatureID string             `xml:"sbc:ReferencedSignatureID"`
	Signature             *xmldsig.Signature `xml:"ds:Signature"`
}

// signatureNamespaces maps the prefixes of the signature types' xml tags.
var signatureNamespaces = map[string]string{
	"sig": ubl.NamespaceSIG,
	"sac": ubl.NamespaceSAC,
	"sbc": ubl.NamespaceSBC,
	"cbc": ubl.NamespaceCBC,
	"ds":  xmldsig.NamespaceDSig,
}

// testData is the content of a made-up extension.
type testData struct {
	XMLName   xml.Name `xml:"ex:Data"`
	Namespace string   `xml:"xmlns:ex,attr,omitempty"`
	Code      string   `xml:"ex:Code,omitempty"`
	Note      string   `xml:"cbc:Note,omitempty"`
}

const (
	testURI       = "urn:example:extension"
	testNamespace = "urn:example:ext"
)

var testNamespaces = map[string]string{"ex": testNamespace, "cbc": ubl.NamespaceCBC}

// document is a root with the declarations the encoder writes on an invoice,
// so that extensions can be encoded and decoded again in place.
type document struct {
	XMLName    xml.Name        `xml:"Invoice"`
	UBL        string          `xml:"xmlns,attr"`
	CAC        string          `xml:"xmlns:cac,attr"`
	CBC        string          `xml:"xmlns:cbc,attr"`
	EXT        string          `xml:"xmlns:ext,attr"`
	Extensions *ubl.Extensions `xml:"ext:UBLExtensions"`
}

func encodeExtensions(t *testing.T, exts *ubl.Extensions, indent bool) string {
	t.Helper()
	doc := document{
		UBL:        ubl.NamespaceUBLInvoice,
		CAC:        ubl.NamespaceCAC,
		CBC:        ubl.NamespaceCBC,
		EXT:        ubl.NamespaceEXT,
		Extensions: exts,
	}
	var out []byte
	var err error
	if indent {
		out, err = xml.MarshalIndent(doc, "", "  ")
	} else {
		out, err = xml.Marshal(doc)
	}
	require.NoError(t, err)
	return string(out)
}

func decodeInvoice(t *testing.T, data string) *ubl.Invoice {
	t.Helper()
	doc, err := ubl.Decode([]byte(data))
	require.NoError(t, err)
	inv, ok := doc.(*ubl.Invoice)
	require.True(t, ok)
	return inv
}

// invoiceXML returns an invoice with one extension holding the content. The
// attrs are written on the root, on ext:UBLExtensions, on ext:UBLExtension
// and on ext:ExtensionContent, in that order.
func invoiceXML(content string, attrs ...string) string {
	at := func(i int) string {
		if i < len(attrs) && attrs[i] != "" {
			return " " + attrs[i]
		}
		return ""
	}
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<Invoice xmlns="` + ubl.NamespaceUBLInvoice + `" xmlns:cac="` + ubl.NamespaceCAC +
		`" xmlns:cbc="` + ubl.NamespaceCBC + `" xmlns:ext="` + ubl.NamespaceEXT + `"` + at(0) + `>` +
		`<ext:UBLExtensions` + at(1) + `><ext:UBLExtension` + at(2) + `>` +
		`<ext:ExtensionURI>` + testURI + `</ext:ExtensionURI>` +
		`<ext:ExtensionContent` + at(3) + `>` + content + `</ext:ExtensionContent>` +
		`</ext:UBLExtension></ext:UBLExtensions><cbc:ID>INV-1</cbc:ID></Invoice>`
}

// assertExtensionPrefixes checks that every element and attribute of the
// document has its prefix declared.
func assertExtensionPrefixes(t *testing.T, data string) {
	t.Helper()
	d := xml.NewDecoder(strings.NewReader(data))
	for {
		tok, err := d.Token()
		if err != nil {
			require.ErrorContains(t, err, "EOF")
			return
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		assert.Contains(t, start.Name.Space, ":", "element %s", start.Name.Local)
		for _, a := range start.Attr {
			if a.Name.Space != "" && a.Name.Space != "xmlns" {
				assert.Contains(t, a.Name.Space, ":", "attribute %s of %s", a.Name.Local, start.Name.Local)
			}
		}
	}
}

func TestNewExtension(t *testing.T) {
	t.Run("with a URI", func(t *testing.T) {
		d := &testData{Code: "A1"}
		ext := ubl.NewExtension(testURI, d)
		require.NotNil(t, ext.ExtensionURI)
		assert.Equal(t, testURI, *ext.ExtensionURI)
		require.NotNil(t, ext.ExtensionContent)
		assert.Same(t, d, ext.ExtensionContent.Value)
	})
	t.Run("without a URI", func(t *testing.T) {
		ext := ubl.NewExtension("", &testData{})
		assert.Nil(t, ext.ExtensionURI)
	})
}

func TestAddExtension(t *testing.T) {
	t.Run("initializes nil Extensions", func(t *testing.T) {
		inv := &ubl.Invoice{}
		ext := ubl.NewExtension(testURI, &testData{})
		inv.AddExtension(ext)

		require.NotNil(t, inv.Extensions)
		require.Len(t, inv.Extensions.Extension, 1)
		assert.Equal(t, *ext, inv.Extensions.Extension[0])
	})

	t.Run("appends to existing Extensions", func(t *testing.T) {
		uri := "urn:existing"
		inv := &ubl.Invoice{
			Extensions: &ubl.Extensions{Extension: []ubl.Extension{{ExtensionURI: &uri}}},
		}
		inv.AddExtension(ubl.NewExtension(testURI, &testData{}))

		require.Len(t, inv.Extensions.Extension, 2)
		assert.Equal(t, "urn:existing", *inv.Extensions.Extension[0].ExtensionURI)
		assert.Equal(t, testURI, *inv.Extensions.Extension[1].ExtensionURI)
	})

	t.Run("stores a copy of the extension", func(t *testing.T) {
		inv := &ubl.Invoice{}
		ext := ubl.NewExtension(testURI, &testData{})
		inv.AddExtension(ext)
		ext.ExtensionURI = nil

		require.NotNil(t, inv.Extensions.Extension[0].ExtensionURI,
			"AddExtension should copy the extension by value, decoupling later mutations")
	})
}

func TestFindExtension(t *testing.T) {
	t.Run("no extensions", func(t *testing.T) {
		assert.Nil(t, new(ubl.Invoice).FindExtension(testURI))
	})

	inv := new(ubl.Invoice)
	inv.AddExtension(ubl.NewExtension("", &testData{Code: "none"}))
	inv.AddExtension(ubl.NewExtension(testURI, &testData{Code: "first"}))
	inv.AddExtension(ubl.NewExtension(testURI, &testData{Code: "second"}))

	t.Run("first with the URI", func(t *testing.T) {
		ext := inv.FindExtension(testURI)
		require.NotNil(t, ext)
		assert.Same(t, &inv.Extensions.Extension[1], ext)
	})
	t.Run("empty URI finds the extension with none", func(t *testing.T) {
		ext := inv.FindExtension("")
		require.NotNil(t, ext)
		assert.Same(t, &inv.Extensions.Extension[0], ext)
	})
	t.Run("unknown URI", func(t *testing.T) {
		assert.Nil(t, inv.FindExtension("urn:example:other"))
	})
}

func TestExtensionXMLOmitsEmpty(t *testing.T) {
	out, err := xml.Marshal(ubl.Extension{})
	require.NoError(t, err)

	assert.NotContains(t, string(out), "<ext:ExtensionURI>")
	assert.NotContains(t, string(out), "<ext:ExtensionContent>")
}

// signatureXML is the signature extension as gobl.ubl v0.99.0 writes it.
// Signed documents depend on these bytes.
const signatureXML = `<Extensions><ext:UBLExtension><ext:ExtensionURI>urn:oasis:names:specification:ubl:dsig:enveloped:xades</ext:ExtensionURI><ext:ExtensionContent><sig:UBLDocumentSignatures xmlns:sig="urn:oasis:names:specification:ubl:schema:xsd:CommonSignatureComponents-2" xmlns:sac="urn:oasis:names:specification:ubl:schema:xsd:SignatureAggregateComponents-2" xmlns:sbc="urn:oasis:names:specification:ubl:schema:xsd:SignatureBasicComponents-2"><sac:SignatureInformation><cbc:ID>urn:oasis:names:specification:ubl:signature:1</cbc:ID><sbc:ReferencedSignatureID>urn:oasis:names:specification:ubl:signature:Invoice</sbc:ReferencedSignatureID><ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#" Id="signature"><ds:SignedInfo><ds:Reference URI=""><ds:DigestValue>ZGlnZXN0</ds:DigestValue></ds:Reference></ds:SignedInfo><ds:SignatureValue Id="">c2lnbmF0dXJl</ds:SignatureValue></ds:Signature></sac:SignatureInformation></sig:UBLDocumentSignatures></ext:ExtensionContent></ext:UBLExtension></Extensions>`

// signatureValue builds signature content with fixed values.
func signatureValue() *documentSignatures {
	return &documentSignatures{
		SIGNamespace: ubl.NamespaceSIG,
		SACNamespace: ubl.NamespaceSAC,
		SBCNamespace: ubl.NamespaceSBC,
		SignatureInformation: []signatureInformation{{
			ID:                    ubl.SignatureInformationID,
			ReferencedSignatureID: ubl.ReferenceSignatureID,
			Signature: &xmldsig.Signature{
				DSigNamespace: xmldsig.NamespaceDSig,
				ID:            "signature",
				SignedInfo: &xmldsig.SignedInfo{
					Reference: []*xmldsig.Reference{{URI: "", DigestValue: "ZGlnZXN0"}},
				},
				Value: &xmldsig.Value{Value: "c2lnbmF0dXJl"},
			},
		}},
	}
}

func TestSignatureExtensionXML(t *testing.T) {
	inv := new(ubl.Invoice)
	inv.AddExtension(ubl.NewExtension(ubl.SignatureMethod, signatureValue()))
	out, err := xml.Marshal(inv.Extensions)
	require.NoError(t, err)
	assert.Equal(t, signatureXML, string(out))
}

// signatureDeclarations are the declarations signatureContent needs besides
// the ones of the root.
const signatureDeclarations = `xmlns:sig="` + ubl.NamespaceSIG + `" xmlns:sac="` + ubl.NamespaceSAC +
	`" xmlns:sbc="` + ubl.NamespaceSBC + `" xmlns:ds="` + xmldsig.NamespaceDSig + `"`

// signatureContent returns signature content with two information blocks. It
// uses cbc from the root, and ext inside a text value.
func signatureContent(attrs string) string {
	if attrs != "" {
		attrs = " " + attrs
	}
	return `
      <sig:UBLDocumentSignatures` + attrs + `>
        <!-- stamp -->
        <sac:SignatureInformation>
          <cbc:ID>urn:oasis:names:specification:ubl:signature:1</cbc:ID>
          <sbc:ReferencedSignatureID>urn:oasis:names:specification:ubl:signature:Invoice</sbc:ReferencedSignatureID>
          <ds:Signature Id="signature"><ds:SignedInfo><ds:Reference URI=""><ds:Transforms><ds:Transform Algorithm="http://www.w3.org/TR/1999/REC-xpath-19991116"><ds:XPath>not(//ancestor-or-self::ext:UBLExtensions)</ds:XPath></ds:Transform></ds:Transforms><ds:DigestValue>ZGlnZXN0</ds:DigestValue></ds:Reference></ds:SignedInfo><ds:SignatureValue>c2lnbmF0dXJl</ds:SignatureValue></ds:Signature>
        </sac:SignatureInformation>
        <sac:SignatureInformation><cbc:ID>second</cbc:ID></sac:SignatureInformation>
      </sig:UBLDocumentSignatures>
    `
}

// signaturePlacements lists where a document may declare the prefixes of its
// signature content, with the start tag the content element is written with.
var signaturePlacements = []struct {
	name    string
	content string
	attrs   []string
	start   string
}{
	{
		name:    "on the content",
		content: signatureContent(signatureDeclarations),
		start:   `<ext:ExtensionContent>`,
	},
	{
		name:    "on the root",
		content: signatureContent(""),
		attrs:   []string{signatureDeclarations},
		start:   sortedSignatureStart,
	},
	{
		name:    "on ext:UBLExtensions",
		content: signatureContent(""),
		attrs:   []string{"", signatureDeclarations},
		start:   sortedSignatureStart,
	},
	{
		name:    "on ext:UBLExtension",
		content: signatureContent(""),
		attrs:   []string{"", "", signatureDeclarations},
		start:   sortedSignatureStart,
	},
	{
		name:    "on ext:ExtensionContent",
		content: signatureContent(""),
		attrs:   []string{"", "", "", signatureDeclarations},
		start:   sortedSignatureStart,
	},
}

// sortedSignatureStart is the content element with the signature
// declarations, which are written sorted by prefix.
const sortedSignatureStart = `<ext:ExtensionContent xmlns:ds="` + xmldsig.NamespaceDSig + `" xmlns:sac="` + ubl.NamespaceSAC +
	`" xmlns:sbc="` + ubl.NamespaceSBC + `" xmlns:sig="` + ubl.NamespaceSIG + `">`

func assertSignature(t *testing.T, ec *ubl.ExtensionContent) {
	t.Helper()
	sigs, err := ubl.DecodeExtension[documentSignatures](ec, signatureNamespaces)
	require.NoError(t, err)
	assert.Equal(t, xml.Name{Space: ubl.NamespaceSIG, Local: "UBLDocumentSignatures"}, sigs.XMLName)
	require.Len(t, sigs.SignatureInformation, 2)
	info := sigs.SignatureInformation[0]
	assert.Equal(t, ubl.SignatureInformationID, info.ID)
	assert.Equal(t, ubl.ReferenceSignatureID, info.ReferencedSignatureID)
	require.NotNil(t, info.Signature)
	assert.Equal(t, "ZGlnZXN0", info.Signature.SignedInfo.Reference[0].DigestValue)
	assert.Equal(t, "c2lnbmF0dXJl", info.Signature.Value.Value)
	assert.Equal(t, "second", sigs.SignatureInformation[1].ID)
}

func TestDecodeExtensionSignature(t *testing.T) {
	for _, p := range signaturePlacements {
		t.Run("prefixes declared "+p.name, func(t *testing.T) {
			inv := decodeInvoice(t, invoiceXML(p.content, p.attrs...))
			ext := inv.FindExtension(testURI)
			require.NotNil(t, ext)
			assert.Equal(t, p.content, string(ext.ExtensionContent.Inner), "source bytes")
			assertSignature(t, ext.ExtensionContent)
		})
	}

	t.Run("another prefix for a root namespace", func(t *testing.T) {
		content := strings.ReplaceAll(signatureContent(signatureDeclarations), "cbc:", "b:")
		inv := decodeInvoice(t, invoiceXML(content, `xmlns:b="`+ubl.NamespaceCBC+`"`))
		ext := inv.FindExtension(testURI)
		assertSignature(t, ext.ExtensionContent)

		out := encodeExtensions(t, inv.Extensions, false)
		assert.Contains(t, out, `<ext:ExtensionContent xmlns:b="`+ubl.NamespaceCBC+`">`+content+`</ext:ExtensionContent>`)
		assertExtensionPrefixes(t, out)
	})
}

func TestExtensionContentRoundTrip(t *testing.T) {
	for _, p := range signaturePlacements {
		for _, indent := range []bool{false, true} {
			name := "prefixes declared " + p.name
			if indent {
				name += ", indented"
			}
			t.Run(name, func(t *testing.T) {
				inv := decodeInvoice(t, invoiceXML(p.content, p.attrs...))

				out := encodeExtensions(t, inv.Extensions, indent)
				assert.Contains(t, out, p.start+p.content+`</ext:ExtensionContent>`, "source bytes are written back")
				assertExtensionPrefixes(t, out)

				again := decodeInvoice(t, out)
				ext := again.FindExtension(testURI)
				require.NotNil(t, ext)
				assert.Equal(t, p.content, string(ext.ExtensionContent.Inner))
				assertSignature(t, ext.ExtensionContent)
				assert.Equal(t, out, encodeExtensions(t, again.Extensions, indent), "stable after one round")
			})
		}
	}
}

func TestExtensionContentSourceBytes(t *testing.T) {
	tests := map[string]string{
		"element":                `<a:x xmlns:a="urn:example:a"><a:y>1</a:y></a:x>`,
		"comment":                `<!-- a note --><a:x xmlns:a="urn:example:a"/>`,
		"processing instruction": `<?target data?><a:x xmlns:a="urn:example:a"/>`,
		"cdata":                  `<a:x xmlns:a="urn:example:a"><![CDATA[1 < 2 & 3]]></a:x>`,
		"two elements":           `<a:x xmlns:a="urn:example:a"/><b:y xmlns:b="urn:example:b"/>`,
		"text around":            `before<a:x xmlns:a="urn:example:a"/>after`,
		"text only":              `just text`,
		"prefix in a value":      `<a:x xmlns:a="urn:example:a" xmlns:xsi="` + ubl.NamespaceXSI + `" xsi:type="a:T">cbc:ID</a:x>`,
		"no namespace":           `<x xmlns=""><y/></x>`,
		"default namespace":      `<Custom><Tag>1</Tag></Custom>`,
		"references":             `<a:x xmlns:a="urn:example:a" v="&quot;&#x9;">&amp;&#10;&lt;&#39;</a:x>`,
		"whitespace":             "\n      <a:x xmlns:a=\"urn:example:a\">\n\t<a:y> 1 </a:y>\r\n</a:x>\n    ",
		"whitespace only":        "\n    ",
		"empty":                  ``,
	}
	for name, content := range tests {
		for _, indent := range []bool{false, true} {
			if indent {
				name += ", indented"
			}
			t.Run(name, func(t *testing.T) {
				inv := decodeInvoice(t, invoiceXML(content))
				ext := inv.FindExtension(testURI)
				require.NotNil(t, ext)
				assert.Equal(t, content, string(ext.ExtensionContent.Inner))
				assert.Nil(t, ext.ExtensionContent.Value)

				out := encodeExtensions(t, inv.Extensions, indent)
				assert.Contains(t, out, `<ext:ExtensionContent>`+content+`</ext:ExtensionContent>`)

				again := decodeInvoice(t, out)
				assert.Equal(t, content, string(again.FindExtension(testURI).ExtensionContent.Inner))
				assert.Equal(t, out, encodeExtensions(t, again.Extensions, indent))
			})
		}
	}

	t.Run("empty element", func(t *testing.T) {
		data := strings.Replace(invoiceXML(""), `<ext:ExtensionContent></ext:ExtensionContent>`, `<ext:ExtensionContent/>`, 1)
		inv := decodeInvoice(t, data)
		ec := inv.FindExtension(testURI).ExtensionContent
		require.NotNil(t, ec)
		assert.Empty(t, ec.Inner)
	})

	t.Run("each extension keeps its own content and scope", func(t *testing.T) {
		data := `<Invoice xmlns="` + ubl.NamespaceUBLInvoice + `" xmlns:cbc="` + ubl.NamespaceCBC + `" xmlns:ext="` + ubl.NamespaceEXT + `">` +
			`<cbc:ID>INV-1</cbc:ID>` +
			`<ext:UBLExtensions>` +
			`<ext:UBLExtension xmlns:ex="urn:example:first"><ext:ExtensionContent><ex:Data/></ext:ExtensionContent></ext:UBLExtension>` +
			`<ext:UBLExtension><ext:ExtensionURI>urn:example:none</ext:ExtensionURI></ext:UBLExtension>` +
			`<cbc:Note>not an extension</cbc:Note>` +
			`<ext:UBLExtension xmlns:ex="urn:example:third"><ext:ExtensionURI>` + testURI + `</ext:ExtensionURI><ext:ExtensionContent><ex:Data/></ext:ExtensionContent></ext:UBLExtension>` +
			`</ext:UBLExtensions></Invoice>`
		inv := decodeInvoice(t, data)
		require.Len(t, inv.Extensions.Extension, 3)
		assert.Nil(t, inv.Extensions.Extension[1].ExtensionContent)

		first, err := ubl.DecodeExtension[testData](inv.FindExtension("").ExtensionContent, map[string]string{"ex": "urn:example:first"})
		require.NoError(t, err)
		assert.Equal(t, "urn:example:first", first.XMLName.Space)
		third, err := ubl.DecodeExtension[testData](inv.FindExtension(testURI).ExtensionContent, map[string]string{"ex": "urn:example:third"})
		require.NoError(t, err)
		assert.Equal(t, "urn:example:third", third.XMLName.Space)

		out := encodeExtensions(t, inv.Extensions, false)
		assert.Contains(t, out, `<ext:ExtensionContent xmlns:ex="urn:example:first"><ex:Data/></ext:ExtensionContent>`)
		assert.Contains(t, out, `<ext:ExtensionContent xmlns:ex="urn:example:third"><ex:Data/></ext:ExtensionContent>`)
	})

	t.Run("root prefix declared below the root", func(t *testing.T) {
		// a root that does not declare xsi is not written with it
		xsi := `xmlns:xsi="` + ubl.NamespaceXSI + `"`
		content := `<a:x xmlns:a="urn:example:a" xsi:type="a:T"/>`
		inv := decodeInvoice(t, invoiceXML(content, "", "", xsi))
		out := encodeExtensions(t, inv.Extensions, false)
		assert.Contains(t, out, `<ext:ExtensionContent `+xsi+`>`+content+`</ext:ExtensionContent>`)
		assertExtensionPrefixes(t, out)
	})

	t.Run("apostrophe reference", func(t *testing.T) {
		content := `<a:x xmlns:a="urn:example:a" v='it&#39;s'>it&#39;s</a:x>`
		inv := decodeInvoice(t, invoiceXML(content))
		for _, encode := range []func(ubl.Document) ([]byte, error){ubl.Encode, ubl.EncodeCompact} {
			out, err := encode(inv)
			require.NoError(t, err)
			assert.Contains(t, string(out), `<ext:ExtensionContent>`+content+`</ext:ExtensionContent>`)
		}

		// a value is written by the encoder, so its apostrophes are plain
		inv.FindExtension(testURI).ExtensionContent.Value = &testData{Namespace: testNamespace, Code: "it's"}
		out, err := ubl.EncodeCompact(inv)
		require.NoError(t, err)
		assert.Contains(t, string(out), `<ex:Code>it's</ex:Code>`)
	})

	t.Run("root with a prefix", func(t *testing.T) {
		// the content is in no namespace, and the encoder writes a root with
		// a default one
		data := `<ubl:Invoice xmlns:ubl="` + ubl.NamespaceUBLInvoice + `" xmlns:cbc="` + ubl.NamespaceCBC + `" xmlns:ext="` + ubl.NamespaceEXT + `">` +
			`<ext:UBLExtensions><ext:UBLExtension><ext:ExtensionContent><Custom/></ext:ExtensionContent></ext:UBLExtension></ext:UBLExtensions>` +
			`<cbc:ID>INV-1</cbc:ID></ubl:Invoice>`
		inv := decodeInvoice(t, data)
		out := encodeExtensions(t, inv.Extensions, false)
		assert.Contains(t, out, `<ext:ExtensionContent xmlns="" xmlns:ubl="`+ubl.NamespaceUBLInvoice+`"><Custom/></ext:ExtensionContent>`)

		again := decodeInvoice(t, out)
		assert.Equal(t, out, encodeExtensions(t, again.Extensions, false))
	})

	t.Run("malformed content", func(t *testing.T) {
		_, err := ubl.Decode([]byte(invoiceXML(`<a></b>`)))
		assert.Error(t, err)
	})

	t.Run("decoding again replaces the content", func(t *testing.T) {
		ec := new(ubl.ExtensionContent)
		require.NoError(t, xml.Unmarshal([]byte(`<ExtensionContent xmlns:ex="`+testNamespace+`"><ex:Data/></ExtensionContent>`), ec))
		assert.Equal(t, `<ex:Data/>`, string(ec.Inner))
		assert.Equal(t, map[string]string{"ex": testNamespace}, ec.Namespaces)

		ec.Value = &testData{}
		require.NoError(t, xml.Unmarshal([]byte(`<ExtensionContent/>`), ec))
		assert.Empty(t, ec.Inner)
		assert.Empty(t, ec.Namespaces)
		assert.Nil(t, ec.Value)
	})
}

func TestExtensionContentXML(t *testing.T) {
	t.Run("value", func(t *testing.T) {
		ext := ubl.NewExtension(testURI, &testData{Namespace: testNamespace, Code: "A1"})
		out, err := xml.Marshal(ext)
		require.NoError(t, err)
		assert.Equal(t, `<Extension><ext:ExtensionURI>`+testURI+`</ext:ExtensionURI>`+
			`<ext:ExtensionContent><ex:Data xmlns:ex="`+testNamespace+`"><ex:Code>A1</ex:Code></ex:Data></ext:ExtensionContent></Extension>`, string(out))
	})

	t.Run("value is written instead of the source bytes", func(t *testing.T) {
		inv := decodeInvoice(t, invoiceXML(`<ex:Data xmlns:ex="`+testNamespace+`"><ex:Code>old</ex:Code></ex:Data>`))
		ec := inv.FindExtension(testURI).ExtensionContent
		ec.Value = &testData{Namespace: testNamespace, Code: "new"}

		out := encodeExtensions(t, inv.Extensions, false)
		assert.Contains(t, out, `<ex:Code>new</ex:Code>`)
		assert.NotContains(t, out, `old`)
	})

	t.Run("source bytes set by hand", func(t *testing.T) {
		ec := &ubl.ExtensionContent{
			Inner:      []byte(`<ex:Data><ex:Code>A1</ex:Code></ex:Data>`),
			Namespaces: map[string]string{"ex": testNamespace, "": "urn:example:default"},
		}
		out, err := xml.Marshal(ec)
		require.NoError(t, err)
		assert.Equal(t, `<ExtensionContent xmlns="urn:example:default" xmlns:ex="`+testNamespace+`"><ex:Data><ex:Code>A1</ex:Code></ex:Data></ExtensionContent>`, string(out))
	})

	t.Run("value that cannot be written", func(t *testing.T) {
		_, err := xml.Marshal(ubl.NewExtension(testURI, make(chan int)))
		assert.Error(t, err)
	})
}

func TestDecodeExtension(t *testing.T) {
	t.Run("no content", func(t *testing.T) {
		_, err := ubl.DecodeExtension[testData](nil, testNamespaces)
		assert.ErrorIs(t, err, ubl.ErrNoExtensionContent)
	})

	t.Run("content with no element", func(t *testing.T) {
		for _, content := range []string{``, "\n  ", `<!-- nothing -->`, `text`} {
			inv := decodeInvoice(t, invoiceXML(content))
			_, err := ubl.DecodeExtension[testData](inv.FindExtension(testURI).ExtensionContent, testNamespaces)
			assert.ErrorIs(t, err, ubl.ErrNoExtensionContent, "content %q", content)
		}
	})

	t.Run("source bytes set by hand", func(t *testing.T) {
		ec := &ubl.ExtensionContent{
			Inner:      []byte(`<ex:Data><ex:Code>A1</ex:Code><cbc:Note>n</cbc:Note></ex:Data>`),
			Namespaces: map[string]string{"ex": testNamespace},
		}
		d, err := ubl.DecodeExtension[testData](ec, testNamespaces)
		require.NoError(t, err)
		assert.Equal(t, "A1", d.Code)
		assert.Equal(t, "n", d.Note, "cbc is a prefix of the root")
	})

	t.Run("malformed source bytes", func(t *testing.T) {
		ec := &ubl.ExtensionContent{Inner: []byte(`<ex:Data>`)}
		_, err := ubl.DecodeExtension[testData](ec, testNamespaces)
		require.Error(t, err)
		assert.NotErrorIs(t, err, ubl.ErrNoExtensionContent)
	})

	t.Run("value", func(t *testing.T) {
		v := &testData{Namespace: testNamespace, Code: "A1", Note: "n"}
		ext := ubl.NewExtension(testURI, v)
		d, err := ubl.DecodeExtension[testData](ext.ExtensionContent, testNamespaces)
		require.NoError(t, err)
		assert.NotSame(t, v, d)
		assert.Equal(t, "A1", d.Code)
		assert.Equal(t, "n", d.Note)
	})

	t.Run("value that cannot be written", func(t *testing.T) {
		ext := ubl.NewExtension(testURI, make(chan int))
		_, err := ubl.DecodeExtension[testData](ext.ExtensionContent, testNamespaces)
		require.Error(t, err)
		assert.NotErrorIs(t, err, ubl.ErrNoExtensionContent)
	})
}
