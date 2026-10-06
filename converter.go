package ubl

import (
	"bytes"
	"encoding/xml"
	"io"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/schema"
)

// KeyUBL identifies UBL documents that do not declare a known specification.
// They can be imported, but not exported.
const KeyUBL cbc.Key = "ubl"

const mimeXML = "application/xml"

var (
	invoiceSchema = schema.Lookup(bill.Invoice{})
	statusSchema  = schema.Lookup(bill.Status{})
)

// converter implements convert.Converter for a set of registered formats.
// The base converter also imports UBL documents that declare no known
// specification.
type converter struct {
	formats  []Format
	fallback bool
}

func (c *converter) Formats() []*convert.Format {
	list := make([]*convert.Format, 0, len(c.formats)+1)
	if c.fallback {
		list = append(list, &convert.Format{
			Key:    KeyUBL,
			Name:   i18n.NewString("UBL"),
			MIME:   mimeXML,
			Syntax: "ubl",
			Import: []schema.ID{invoiceSchema, statusSchema},
		})
	}
	for _, f := range c.formats {
		list = append(list, &convert.Format{
			Key:       f.Key,
			Name:      f.Name,
			MIME:      mimeXML,
			Syntax:    "ubl",
			Countries: f.Countries,
			Addons:    f.Addons,
			Import:    f.Schemas,
			Export:    f.Schemas,
		})
	}
	return list
}

func (c *converter) Detect(in *convert.Input) cbc.Key {
	h := ReadHeader(in)
	if h.Err != nil {
		return cbc.KeyEmpty
	}
	switch h.Namespace {
	case NamespaceUBLInvoice, NamespaceUBLCreditNote, NamespaceUBLApplicationResponse:
	default:
		return cbc.KeyEmpty
	}
	if f := FindFormat(h.CustomizationID, h.ProfileID); f != nil {
		if c.format(f.Key) != nil {
			return f.Key
		}
		return cbc.KeyEmpty
	}
	if c.fallback {
		return KeyUBL
	}
	return cbc.KeyEmpty
}

// Import decodes the data and imports it, determining the format from the
// document in the same way as Detect.
func (c *converter) Import(_ cbc.Key, data []byte) (*gobl.Envelope, error) {
	doc, err := Decode(data)
	if err != nil {
		return nil, err
	}
	return Import(doc)
}

func (c *converter) Accepts(_ cbc.Key, _ *gobl.Envelope) bool {
	return true
}

func (c *converter) Export(key cbc.Key, env *gobl.Envelope) ([]byte, error) {
	f := c.format(key)
	if f == nil {
		return nil, ErrUnsupportedDocumentType
	}
	doc, err := Export(env, WithFormat(*f))
	if err != nil {
		return nil, err
	}
	return Encode(doc)
}

func (c *converter) format(key cbc.Key) *Format {
	for _, f := range c.formats {
		if f.Key == key {
			return &f
		}
	}
	return nil
}

type headerKey struct{}

// Header holds the identifiers at the start of a UBL document that
// determine its format.
type Header struct {
	// Namespace of the root element.
	Namespace string
	// CustomizationID declares the specification the document follows.
	CustomizationID string
	// ProfileID declares the business process.
	ProfileID string
	// Err is set if the identifiers could not be read.
	Err error
}

// ReadHeader provides the root namespace and the specification and
// business process identifiers of the input, reading them only once for all
// the converters that ask.
func ReadHeader(in *convert.Input) *Header {
	if v, ok := in.Get(headerKey{}); ok {
		return v.(*Header)
	}
	h := readHeader(in.Data)
	in.Set(headerKey{}, h)
	return h
}

// readHeader reads the root element and the header elements that
// precede the document ID, stopping at the first other element.
func readHeader(data []byte) *Header {
	h := new(Header)
	d := xml.NewDecoder(bytes.NewReader(data))
	root := false
	for {
		tk, err := d.Token()
		if err == io.EOF {
			if !root {
				h.Err = ErrUnknownDocumentType
			}
			return h
		}
		if err != nil {
			h.Err = err
			return h
		}
		se, ok := tk.(xml.StartElement)
		if !ok {
			continue
		}
		if !root {
			h.Namespace = se.Name.Space
			root = true
			continue
		}
		switch se.Name.Local {
		case "UBLExtensions", "UBLVersionID":
			err = d.Skip()
		case "CustomizationID":
			err = d.DecodeElement(&h.CustomizationID, &se)
		case "ProfileID":
			err = d.DecodeElement(&h.ProfileID, &se)
		default:
			return h
		}
		if err != nil {
			h.Err = err
			return h
		}
	}
}
