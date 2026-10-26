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
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/schema"
	"github.com/invopop/xmlctx"
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
			Key:         f.Key,
			Name:        f.Name,
			MIME:        mimeXML,
			Syntax:      "ubl",
			Countries:   f.Countries,
			Import:      f.Import,
			Export:      f.Export,
			Attachments: true,
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
// document in the same way as Detect, and provides the files embedded in
// invoices as attachments.
func (c *converter) Import(_ cbc.Key, data []byte) (*convert.ImportResult, error) {
	doc, err := Decode(data)
	if err != nil {
		return nil, err
	}
	env, err := Import(doc)
	if err != nil {
		return nil, err
	}
	res := &convert.ImportResult{Envelope: env}
	if inv, ok := doc.(*Invoice); ok {
		res.Attachments = convertAttachments(inv.ExtractBinaryAttachments())
	}
	return res, nil
}

// Accepts checks the document includes the addons of the specification the
// format exports it with.
func (c *converter) Accepts(key cbc.Key, env *gobl.Envelope) error {
	f := c.format(key)
	if f == nil {
		return ErrUnsupportedDocumentType
	}
	doc := env.Extract()
	s := f.SpecificationFor(doc)
	if len(s.Addons) == 0 {
		return nil
	}
	set := convert.Rules(f.Key, rules.For(doc, convert.RequireAddons("01", s.Addons...)))
	if fs := set.Validate(doc); fs != nil {
		return fs
	}
	return nil
}

// Export exports the envelope in the format, embedding the attachments in
// invoices.
func (c *converter) Export(key cbc.Key, env *gobl.Envelope, opts *convert.ExportOptions) ([]byte, error) {
	f := c.format(key)
	if f == nil {
		return nil, ErrUnsupportedDocumentType
	}
	doc, err := Export(env, WithFormat(*f), WithAttachments(binaryAttachments(opts.Attachments)...))
	if err != nil {
		return nil, err
	}
	return Encode(doc)
}

// binaryAttachments provides the register's attachments as UBL binary
// attachments, identified by their name when they have no ID.
func binaryAttachments(atts []*convert.Attachment) []BinaryAttachment {
	out := make([]BinaryAttachment, len(atts))
	for i, a := range atts {
		id := a.ID
		if id == "" {
			id = a.Name
		}
		out[i] = BinaryAttachment{
			ID:          id,
			Description: a.Description,
			Data:        a.Data,
			MimeCode:    a.MIME,
			Filename:    a.Name,
		}
	}
	return out
}

// convertAttachments provides UBL binary attachments as the register's
// attachments.
func convertAttachments(atts []BinaryAttachment) []*convert.Attachment {
	out := make([]*convert.Attachment, len(atts))
	for i, a := range atts {
		out[i] = &convert.Attachment{
			ID:          a.ID,
			Name:        a.Filename,
			Description: a.Description,
			MIME:        a.MimeCode,
			Data:        a.Data,
		}
	}
	return out
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
	d := xmlctx.NewXMLDecoder(bytes.NewReader(data))
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
