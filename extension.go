package ubl

import (
	"bytes"
	"encoding/xml"
	"errors"
	"maps"
	"slices"

	"github.com/invopop/xmlctx"
)

// ErrNoExtensionContent is returned when an extension has no content to
// decode.
var ErrNoExtensionContent = errors.New("no extension content")

// xmlnsPrefix is the prefix of namespace declarations. The decoder gives it
// as the space of an attribute that declares a prefix, and as the name of the
// attribute that declares the default namespace.
const xmlnsPrefix = "xmlns"

// rootNamespaces are the declarations every invoice root is written with, by
// prefix. Extension content can rely on them.
var rootNamespaces = map[string]string{
	"cac":  NamespaceCAC,
	"cbc":  NamespaceCBC,
	"qdt":  NamespaceQDT,
	"udt":  NamespaceUDT,
	"ccts": NamespaceCCTS,
	"xsi":  NamespaceXSI,
	"ext":  NamespaceEXT,
}

// Extensions wraps a list of UBL extensions.
type Extensions struct {
	Extension []Extension `xml:"ext:UBLExtension"`
}

// Extension represents a single UBL extension.
type Extension struct {
	ExtensionURI     *string           `xml:"ext:ExtensionURI"`
	ExtensionContent *ExtensionContent `xml:"ext:ExtensionContent"`
}

// ExtensionContent is the content of a UBL extension: one element of any
// kind. It is carried without being interpreted. Decoding keeps the source
// bytes and encoding writes them back, so the content of a document is not
// changed by reading it. Use DecodeExtension to read the content into a type.
//
// The source bytes may use prefixes that the document declares above the
// content. Only Decode records those declarations: content decoded on its
// own, without its document, has just the ones of its element.
type ExtensionContent struct {
	// Value is the content to write. It must name its element with an
	// XMLName field and declare every prefix it writes that is not one of
	// the document root. When set, it is written instead of Inner.
	Value any
	// Inner is the content as it was read, byte for byte. A document in
	// another encoding is read as UTF-8.
	Inner []byte
	// Namespaces are the declarations of the ext:ExtensionContent element
	// itself, by prefix, with "" for the default namespace.
	Namespaces map[string]string

	// scope holds the declarations of the element's ancestors in the decoded
	// document, which Inner may depend on.
	scope map[string]string
	// carried holds the ones of scope that the element must make itself
	// when written, as the root of a written document does not make them.
	carried map[string]string
}

// extensionXML is how extension content is written and read: a value, or the
// source bytes as they are.
type extensionXML struct {
	Value any    `xml:",any"`
	Inner []byte `xml:",innerxml"`
}

// NewExtension creates an extension that holds the value as its content. An
// empty uri leaves the extension without one.
func NewExtension(uri string, value any) *Extension {
	ext := &Extension{
		ExtensionContent: &ExtensionContent{Value: value},
	}
	if uri != "" {
		ext.ExtensionURI = &uri
	}
	return ext
}

// DecodeExtension decodes the content of an extension into a new T. The ns
// maps the prefixes of T's xml tags to their namespaces. It returns
// ErrNoExtensionContent when the content has no element.
func DecodeExtension[T any](ec *ExtensionContent, ns map[string]string) (*T, error) {
	if ec == nil {
		return nil, ErrNoExtensionContent
	}
	// The content is decoded inside an element with every declaration it had
	// in scope, as its prefixes may be declared anywhere above it.
	var buf bytes.Buffer
	start := xml.StartElement{
		Name: xml.Name{Local: "ExtensionContent"},
		Attr: declarations(rootNamespaces, ec.scope, ec.Namespaces),
	}
	if err := xml.NewEncoder(&buf).EncodeElement(ec.xml(), start); err != nil {
		return nil, err
	}
	out := new(struct {
		Value *T `xml:",any"`
	})
	if err := xmlctx.Unmarshal(buf.Bytes(), out, xmlctx.WithNamespaces(ns)); err != nil {
		return nil, err
	}
	if out.Value == nil {
		return nil, ErrNoExtensionContent
	}
	return out.Value, nil
}

// MarshalXML writes the value, or else the source bytes unchanged.
func (ec ExtensionContent) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Attr = declarations(ec.carried, ec.Namespaces)
	return e.EncodeElement(ec.xml(), start)
}

// UnmarshalXML keeps the source bytes of the content and the declarations of
// its element.
func (ec *ExtensionContent) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var src extensionXML
	if err := d.DecodeElement(&src, &start); err != nil {
		return err
	}
	*ec = ExtensionContent{Inner: src.Inner}
	if ns := declared(start.Attr); len(ns) > 0 {
		ec.Namespaces = ns
	}
	return nil
}

func (ec ExtensionContent) xml() extensionXML {
	if ec.Value != nil {
		return extensionXML{Value: ec.Value}
	}
	return extensionXML{Inner: ec.Inner}
}

// declarations returns the namespace declarations of the sets as attributes
// sorted by prefix. A set replaces the declarations of the ones before it.
func declarations(sets ...map[string]string) []xml.Attr {
	ns := make(map[string]string)
	for _, set := range sets {
		maps.Copy(ns, set)
	}
	attrs := make([]xml.Attr, 0, len(ns))
	for _, p := range slices.Sorted(maps.Keys(ns)) {
		name := xmlnsPrefix
		if p != "" {
			name += ":" + p
		}
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: name}, Value: ns[p]})
	}
	return attrs
}

// rootDeclares reports whether an invoice root that has the declaration is
// written with it. Any other declaration of a root is lost when it is written.
func rootDeclares(prefix, uri string) bool {
	if prefix == "" {
		return uri == NamespaceUBLInvoice || uri == NamespaceUBLCreditNote
	}
	return rootNamespaces[prefix] == uri
}

// declared returns the namespace declarations among the attributes, by
// prefix, with "" for the default namespace.
func declared(attrs []xml.Attr) map[string]string {
	ns := make(map[string]string)
	for _, a := range attrs {
		switch {
		case a.Name.Space == xmlnsPrefix:
			ns[a.Name.Local] = a.Value
		case a.Name.Space == "" && a.Name.Local == xmlnsPrefix:
			ns[""] = a.Value
		}
	}
	return ns
}

// AddExtension adds a new extension to the ubl invoice
func (ui *Invoice) AddExtension(extension *Extension) {
	if ui.Extensions == nil {
		ui.Extensions = &Extensions{}
	}
	ui.Extensions.Extension = append(ui.Extensions.Extension, *extension)
}

// setExtensionScopes records, on the content of each extension, the namespace
// declarations of its ancestors: the root, ext:UBLExtensions and
// ext:UBLExtension. The source bytes of the content may use their prefixes.
// It also records which of them the content must carry when written: all but
// the ones the root made and is written with. The data has just been decoded
// into the invoice, so reading it again does not fail.
func (ui *Invoice) setExtensionScopes(data []byte) {
	if ui.Extensions == nil {
		return
	}
	// path holds the declarations in scope on each open ancestor, and names
	// the element to follow at each depth.
	var path []map[string]string
	names := []string{"UBLExtensions", "UBLExtension"}
	exts := ui.Extensions.Extension
	d := xmlctx.NewXMLDecoder(bytes.NewReader(data))
	for tok, err := d.Token(); err == nil; tok, err = d.Token() {
		switch t := tok.(type) {
		case xml.StartElement:
			scope := declared(t.Attr)
			if depth := len(path); depth > 0 {
				if t.Name != (xml.Name{Space: NamespaceEXT, Local: names[depth-1]}) {
					_ = d.Skip()
					continue
				}
				for p, uri := range path[depth-1] {
					if _, ok := scope[p]; !ok {
						scope[p] = uri
					}
				}
			}
			if len(path) < 2 {
				path = append(path, scope)
				continue
			}
			// an ext:UBLExtension
			if _, ok := scope[""]; !ok {
				scope[""] = "" // no default namespace, unlike the root that is written
			}
			if len(exts) > 0 {
				if ec := exts[0].ExtensionContent; ec != nil {
					ec.scope = scope
					ec.carried = make(map[string]string)
					for p, uri := range scope {
						if path[0][p] != uri || !rootDeclares(p, uri) {
							ec.carried[p] = uri
						}
					}
				}
				exts = exts[1:]
			}
			_ = d.Skip()
		case xml.EndElement:
			// Every other element is skipped to its end, and the root
			// cannot end first, so this is the end of ext:UBLExtensions.
			return
		}
	}
}
