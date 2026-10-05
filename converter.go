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

var convertContexts = []*convert.Context{
	{
		Key:    KeyUBL,
		Name:   i18n.NewString("UBL"),
		MIME:   mimeXML,
		Syntax: "ubl",
		Import: []schema.ID{invoiceSchema, statusSchema},
	},
	newConvertContext(ContextEN16931, "UBL EN 16931", invoiceSchema),
	newConvertContext(ContextPeppol, "UBL Peppol BIS Billing 3", invoiceSchema),
	newConvertContext(ContextPeppolSelfBilled, "UBL Peppol BIS Self-Billing 3", invoiceSchema),
	newConvertContext(ContextPeppolInvoiceResponse, "UBL Peppol Invoice Response 3", statusSchema),
}

func newConvertContext(c Context, name string, s schema.ID) *convert.Context {
	return &convert.Context{
		Key:    c.Key,
		Name:   i18n.NewString(name),
		MIME:   mimeXML,
		Syntax: "ubl",
		Addons: c.Addons,
		Import: []schema.ID{s},
		Export: []schema.ID{s},
	}
}

func init() {
	convert.Register(converter{})
}

// converter implements convert.Converter for UBL documents.
type converter struct{}

func (converter) Contexts() []*convert.Context {
	return convertContexts
}

func (converter) Detect(in *convert.Input) cbc.Key {
	dc := ReadDocumentContext(in)
	if dc.Err != nil {
		return cbc.KeyEmpty
	}
	switch dc.Namespace {
	case NamespaceUBLInvoice, NamespaceUBLCreditNote, NamespaceUBLApplicationResponse:
	default:
		return cbc.KeyEmpty
	}
	if ctx := FindContext(dc.CustomizationID, dc.ProfileID); ctx != nil {
		// Regional contexts have no key here, and are detected by the
		// packages that register them.
		return ctx.Key
	}
	return KeyUBL
}

// Import parses the data and converts it, determining the context from the
// document in the same way as Detect.
func (converter) Import(_ cbc.Key, data []byte) (*gobl.Envelope, error) {
	doc, err := Parse(data)
	if err != nil {
		return nil, err
	}
	switch d := doc.(type) {
	case *Invoice:
		return d.Convert()
	case *ApplicationResponse:
		return d.Convert()
	}
	return nil, ErrUnsupportedDocumentType
}

func (converter) Accepts(_ cbc.Key, _ *gobl.Envelope) bool {
	return true
}

func (converter) Export(key cbc.Key, env *gobl.Envelope) ([]byte, error) {
	ctx := contextForKey(key)
	if ctx == nil {
		return nil, ErrUnsupportedDocumentType
	}
	doc, err := Convert(env, WithContext(*ctx))
	if err != nil {
		return nil, err
	}
	return Bytes(doc)
}

func contextForKey(key cbc.Key) *Context {
	for _, ctx := range contexts {
		if ctx.Key == key {
			return &ctx
		}
	}
	return nil
}

type documentContextKey struct{}

// DocumentContext holds the identifiers at the start of a UBL document that
// determine its context.
type DocumentContext struct {
	// Namespace of the root element.
	Namespace string
	// CustomizationID declares the specification the document follows.
	CustomizationID string
	// ProfileID declares the business process.
	ProfileID string
	// Err is set if the identifiers could not be read.
	Err error
}

// ReadDocumentContext provides the root namespace and the specification and
// business process identifiers of the input, reading them only once for all
// the converters that ask.
func ReadDocumentContext(in *convert.Input) *DocumentContext {
	if v, ok := in.Get(documentContextKey{}); ok {
		return v.(*DocumentContext)
	}
	dc := readDocumentContext(in.Data)
	in.Set(documentContextKey{}, dc)
	return dc
}

// readDocumentContext reads the root element and the header elements that
// precede the document ID, stopping at the first other element.
func readDocumentContext(data []byte) *DocumentContext {
	dc := new(DocumentContext)
	d := xml.NewDecoder(bytes.NewReader(data))
	root := false
	for {
		tk, err := d.Token()
		if err == io.EOF {
			if !root {
				dc.Err = ErrUnknownDocumentType
			}
			return dc
		}
		if err != nil {
			dc.Err = err
			return dc
		}
		se, ok := tk.(xml.StartElement)
		if !ok {
			continue
		}
		if !root {
			dc.Namespace = se.Name.Space
			root = true
			continue
		}
		switch se.Name.Local {
		case "UBLExtensions", "UBLVersionID":
			err = d.Skip()
		case "CustomizationID":
			err = d.DecodeElement(&dc.CustomizationID, &se)
		case "ProfileID":
			err = d.DecodeElement(&dc.ProfileID, &se)
		default:
			return dc
		}
		if err != nil {
			dc.Err = err
			return dc
		}
	}
}
