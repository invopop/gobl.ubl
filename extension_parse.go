package ubl

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"maps"
	"slices"

	"github.com/invopop/xmlctx"
)

// xmlnsPrefix is the prefix of namespace declarations. The decoder gives it
// as the space of the attribute that declares a prefix, and as the name of
// the attribute that declares the default namespace.
const xmlnsPrefix = "xmlns"

// extensionTypes maps the name of an extension content element to the type
// it decodes into.
var extensionTypes = map[xml.Name]func() ExtensionData{}

// RegisterExtensionData registers the type that extension content decodes
// into when its element has the name: Space is the element's namespace and
// Local its local name. Call it from init; a later registration for the same
// name replaces the earlier one.
func RegisterExtensionData(name xml.Name, fn func() ExtensionData) {
	extensionTypes[name] = fn
}

// RawExtensionData keeps extension content no type is registered for.
type RawExtensionData struct {
	// XML is the content as decoded, with each element declaring its namespace.
	XML []byte
}

// ExtensionNamespaces implements ExtensionData. Raw content declares its own
// namespaces, so it needs none.
func (*RawExtensionData) ExtensionNamespaces() map[string]string {
	return nil
}

// MarshalXML writes the kept content again, token by token.
func (r *RawExtensionData) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	d := xml.NewDecoder(bytes.NewReader(r.XML))
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := e.EncodeToken(literalDeclarations(tok)); err != nil {
			return err
		}
	}
}

// UnmarshalXML decodes the content into the type registered for the name of
// its element. Content with no registered type, or that is not a single
// element, is kept as RawExtensionData.
func (ec *ExtensionContent) UnmarshalXML(d *xml.Decoder, _ xml.StartElement) error {
	toks, single, err := readExtensionContent(d)
	if err != nil {
		return err
	}
	if len(toks) == 0 {
		return nil
	}
	if single {
		start := toks[0].(xml.StartElement)
		if fn, ok := extensionTypes[start.Name]; ok {
			data, err := decodeExtensionData(fn(), toks)
			if err != nil {
				return fmt.Errorf("decoding extension content %s (%s): %w", start.Name.Local, start.Name.Space, err)
			}
			ec.Data = data
			return nil
		}
	}
	raw, err := encodeTokens(toks)
	if err != nil {
		return err
	}
	ec.Data = &RawExtensionData{XML: raw}
	return nil
}

// readExtensionContent reads the tokens up to the end of the extension
// content, leaving out comments, processing instructions, directives and
// whitespace-only text. It also reports whether the tokens are a single
// element with no text around it.
func readExtensionContent(d *xml.Decoder) ([]xml.Token, bool, error) {
	var toks []xml.Token
	depth, elements, text := 0, 0, false
	for {
		tok, err := d.Token()
		if err != nil {
			return nil, false, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if depth == 0 {
				elements++
			}
			depth++
		case xml.EndElement:
			if depth == 0 {
				return toks, elements == 1 && !text, nil
			}
			depth--
		case xml.CharData:
			if len(bytes.Trim(t, " \t\r\n")) == 0 {
				continue
			}
			if depth == 0 {
				text = true
			}
		default: // comments, processing instructions and directives
			continue
		}
		toks = append(toks, xml.CopyToken(tok))
	}
}

// decodeExtensionData decodes the element in toks into data. It declares on
// the element the prefixes that data's xml tags use, because the document
// may declare them on an ancestor.
func decodeExtensionData(data ExtensionData, toks []xml.Token) (ExtensionData, error) {
	ns := data.ExtensionNamespaces()
	toks[0] = declarePrefixes(toks[0].(xml.StartElement), ns)
	raw, err := encodeTokens(toks)
	if err != nil {
		return nil, err
	}
	if err := xmlctx.Unmarshal(raw, data, xmlctx.WithNamespaces(ns)); err != nil {
		return nil, err
	}
	return data, nil
}

// declarePrefixes adds to start a declaration for each prefix in ns that it
// does not declare itself.
func declarePrefixes(start xml.StartElement, ns map[string]string) xml.StartElement {
	start = start.Copy()
	for _, p := range slices.Sorted(maps.Keys(ns)) {
		declared := slices.ContainsFunc(start.Attr, func(a xml.Attr) bool {
			return a.Name.Space == xmlnsPrefix && a.Name.Local == p
		})
		if !declared {
			start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: xmlnsPrefix + ":" + p}, Value: ns[p]})
		}
	}
	return start
}

// encodeTokens writes the tokens as XML.
func encodeTokens(toks []xml.Token) ([]byte, error) {
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	for _, tok := range toks {
		if err := enc.EncodeToken(literalDeclarations(tok)); err != nil {
			return nil, err
		}
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// literalDeclarations rewrites the namespace declarations of a start element
// as literal attributes, which the encoder writes as they are; it would
// otherwise write them as _xmlns attributes. It drops the default namespace
// declaration, as the encoder declares the namespace of each element.
func literalDeclarations(tok xml.Token) xml.Token {
	start, ok := tok.(xml.StartElement)
	if !ok {
		return tok
	}
	attrs := make([]xml.Attr, 0, len(start.Attr))
	for _, a := range start.Attr {
		switch {
		case a.Name.Space == xmlnsPrefix:
			attrs = append(attrs, xml.Attr{Name: xml.Name{Local: xmlnsPrefix + ":" + a.Name.Local}, Value: a.Value})
		case a.Name.Space == "" && a.Name.Local == xmlnsPrefix:
			// the encoder declares the element's namespace
		default:
			attrs = append(attrs, a)
		}
	}
	start.Attr = attrs
	return start
}
