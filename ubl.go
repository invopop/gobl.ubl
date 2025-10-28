// Package ubl helps convert GOBL into UBL documents and vice versa.
package ubl

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
)

var (
	// ErrUnknownDocumentType is returned when the document type
	// is not recognized during parsing.
	ErrUnknownDocumentType = fmt.Errorf("unknown document type")

	// ErrUnsupportedDocumentType is returned when the document type
	// is not supported for conversion.
	ErrUnsupportedDocumentType = fmt.Errorf("unsupported document type")
)

// Peppol Billing Profile IDs
const (
	PeppolBillingProfileIDDefault = "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0"
)

// Version is the version of UBL documents that will be generated
// by this package.
const Version = "2.1"

// Context is used to ensure that the generated UBL document
// uses a specific CustomizationID and ProfileID when generating
// the output document.
type Context struct {
	// CustomizationID identifies and specific characteristics in the
	// document which need to be present for local differences.
	CustomizationID string
	// ProfileID determines the business process context or scenario
	// for the exchange of the document
	ProfileID string
}

// ContextEN16931 is the default context for basic UBL documents.
var ContextEN16931 = Context{
	CustomizationID: "urn:cen.eu:en16931:2017",
}

// ContextPeppol defines the default Peppol context.
var ContextPeppol = Context{
	CustomizationID: "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0",
	ProfileID:       PeppolBillingProfileIDDefault,
}

// ContextXRechnung defines the main context to use for XRechnung UBL documents.
var ContextXRechnung = Context{
	CustomizationID: "urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0",
	ProfileID:       PeppolBillingProfileIDDefault,
}

// Parse parses a raw UBL document and converts to a GOBL envelope,
// assuming we're dealing with a known document type.
func Parse(ublDoc []byte) (*gobl.Envelope, error) {
	ns, err := extractRootNamespace(ublDoc)
	if err != nil {
		return nil, err
	}
	env := gobl.NewEnvelope()
	var res any
	switch ns {
	case NamespaceUBLInvoice, NamespaceUBLCreditNote:
		if res, err = parseInvoice(ublDoc); err != nil {
			return nil, err
		}
	default:
		return nil, ErrUnknownDocumentType
	}

	// Whatever we get back, try inserting.
	if err := env.Insert(res); err != nil {
		return nil, err
	}

	return env, nil
}

// Convert takes a GOBL envelope and converts to a UBL document of one
// of the supported types.
func Convert(env *gobl.Envelope, opts ...Option) (any, error) {
	o := &options{
		context: ContextEN16931,
	}
	for _, opt := range opts {
		opt(o)
	}
	switch doc := env.Extract().(type) {
	case *bill.Invoice:
		return newInvoice(doc, o)
	default:
		return nil, ErrUnsupportedDocumentType
	}
}

func Extract() {}

// ConvertInvoice is a convenience function that converts a GOBL envelope
// containing an invoice into a UBL Invoice or CreditNote document.
func ConvertInvoice(env *gobl.Envelope, opts ...Option) (*Invoice, error) {
	doc, err := Convert(env, opts...)
	if err != nil {
		return nil, err
	}
	inv, ok := doc.(*Invoice)
	if !ok {
		return nil, fmt.Errorf("expected invoice, got %T", doc)
	}
	return inv, nil
}

func extractRootNamespace(data []byte) (string, error) {
	dc := xml.NewDecoder(bytes.NewReader(data))
	for {
		tk, err := dc.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("error parsing XML: %w", err)
		}
		switch t := tk.(type) {
		case xml.StartElement:
			return t.Name.Space, nil // Extract and return the namespace
		}
	}
	return "", ErrUnknownDocumentType
}

type options struct {
	context Context
}

// Option is used to define configuration options to use during
// conversion processes.
type Option func(*options)

// WithContext sets the context to use for the configuration
// and business profile.
func WithContext(c Context) Option {
	return func(o *options) {
		o.context = c
	}
}
