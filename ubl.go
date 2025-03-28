// Package ubl helps convert GOBL into UBL documents and vice versa.
package ubl

import (
	"fmt"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
)

const (
	// Peppol Billing Profile ID default value
	PeppolBillingProfileIDDefault = "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0"
)

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

// ParseInvoice parses a raw UBL Invoice and converts to a GOBL envelope
func ParseInvoice(ublDoc []byte) (*gobl.Envelope, error) {
	env := gobl.NewEnvelope()
	inv, err := parseInvoice(ublDoc)
	if err != nil {
		return nil, err
	}
	if err := env.Insert(inv); err != nil {
		return nil, err
	}
	return env, nil
}

// ConvertInvoice takes a GOBL envelope and converts to a UBL Invoice or Credit Note.
func ConvertInvoice(env *gobl.Envelope, opts ...Option) (*Invoice, error) {
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, fmt.Errorf("expected bill.Inboice, got %T", env.Document)
	}
	o := &options{
		context: ContextEN16931,
	}
	for _, opt := range opts {
		opt(o)
	}
	return newInvoice(inv, o)
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
