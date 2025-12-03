// Package ubl helps convert GOBL into UBL documents and vice versa.
package ubl

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/addons/de/xrechnung"
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/addons/fr/facturx"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	nbio "github.com/nbio/xml"
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
	// Addons contains the list of Addons required for this CustomizationID
	// and ProfileID.
	Addons []cbc.Key
}

// When adding new contexts, remember to add them to both the exported
// variable definitions below AND the contexts slice.

// ContextEN16931 is the default context for basic UBL documents.
var ContextEN16931 = Context{
	CustomizationID: "urn:cen.eu:en16931:2017",
	Addons:          []cbc.Key{en16931.V2017},
}

// ContextPeppol defines the default Peppol context.
var ContextPeppol = Context{
	CustomizationID: "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0",
	ProfileID:       PeppolBillingProfileIDDefault,
	Addons:          []cbc.Key{en16931.V2017},
}

// ContextXRechnung defines the main context to use for XRechnung UBL documents.
var ContextXRechnung = Context{
	CustomizationID: "urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0",
	ProfileID:       PeppolBillingProfileIDDefault,
	Addons:          []cbc.Key{xrechnung.V3},
}

// ContextPeppolFranceCIUS defines the context for France UBL Invoice CIUS.
var ContextPeppolFranceCIUS = Context{
	CustomizationID: "urn:cen.eu:en16931:2017#compliant#urn:peppol:france:billing:cius:1.0",
	ProfileID:       "urn:peppol:france:billing:regulated",
	Addons:          []cbc.Key{facturx.V1},
}

// ContextPeppolFranceExtended defines the context for France UBL Invoice Extended.
var ContextPeppolFranceExtended = Context{
	CustomizationID: "urn:cen.eu:en16931:2017#conformant#urn:peppol:france:billing:extended:1.0",
	ProfileID:       "urn:peppol:france:billing:regulated",
	Addons:          []cbc.Key{facturx.V1},
}

// contexts is used internally for reverse lookups during parsing.
// When adding new contexts, remember to add them here AND as exported variables above.
var contexts = []Context{ContextEN16931, ContextPeppol, ContextXRechnung, ContextPeppolFranceCIUS, ContextPeppolFranceExtended}

// Is checks if two contexts are the same.
func (c *Context) Is(c2 Context) bool {
	return c.CustomizationID == c2.CustomizationID && c.ProfileID == c2.ProfileID
}

// FindContext looks up a context by CustomizationID and optionally ProfileID.
// Returns nil if no matching context is found.
func FindContext(customizationID string, profileID string) *Context {
	for _, ctx := range contexts {
		if ctx.CustomizationID == customizationID {
			// If profileID is specified, it must match too
			if ctx.ProfileID != "" && ctx.ProfileID != profileID {
				continue
			}
			return &ctx
		}
	}
	return nil
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
	o := new(options)

	switch ns {
	case NamespaceUBLInvoice, NamespaceUBLCreditNote:
		in := new(Invoice)
		if err := nbio.Unmarshal(ublDoc, in); err != nil {
			return nil, err
		}

		ctx := FindContext(in.CustomizationID, in.ProfileID)
		if ctx != nil {
			o.context = *ctx
		}

		if res, err = goblInvoice(in, o); err != nil {
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
		// Check addons
		missingAddons := make([]cbc.Key, 0)
		for _, ao := range o.context.Addons {
			if !ao.In(doc.GetAddons()...) {
				missingAddons = append(missingAddons, ao)
			}
		}

		// only build if we have missing addons
		if len(missingAddons) > 0 {
			doc.SetAddons(append(doc.GetAddons(), missingAddons...)...)
			if err := doc.Calculate(); err != nil {
				return nil, fmt.Errorf("gobl invoice missing addon %v: %w", missingAddons, err)
			}
			if err := doc.Validate(); err != nil {
				return nil, fmt.Errorf("gobl invoice missing addon %v: %w", missingAddons, err)
			}
		}

		// Removes included taxes as they are not supported in UBL
		if err := doc.RemoveIncludedTaxes(); err != nil {
			return nil, fmt.Errorf("cannot convert invoice with included taxes: %w", err)
		}

		return ublInvoice(doc, o)
	default:
		return nil, ErrUnsupportedDocumentType
	}
}

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
