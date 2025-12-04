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
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
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
	// SelfBilledCustomizationID is an optional alternative CustomizationID
	// to use when the invoice has the self-billed tag. Only relevant for
	// contexts that support self-billing (e.g., Peppol).
	SelfBilledCustomizationID string
	// SelfBilledProfileID is an optional alternative ProfileID to use
	// when the invoice has the self-billed tag.
	SelfBilledProfileID string
	// Addons contains the list of Addons required for this CustomizationID
	// and ProfileID.
	Addons []cbc.Key
	// VESIDs contains the VESID (Validation Exchange Specification ID) mappings
	// for different document types and scenarios within this context.
	VESIDs VESIDMapping
}

// VESIDMapping maps document types and self-billing status to their
// corresponding VESID values.
type VESIDMapping struct {
	// Invoice is the VESID for standard invoices
	Invoice string
	// InvoiceSelfBilled is the VESID for self-billed invoices (optional)
	InvoiceSelfBilled string
	// CreditNote is the VESID for credit notes
	CreditNote string
	// CreditNoteSelfBilled is the VESID for self-billed credit notes (optional)
	CreditNoteSelfBilled string
}

// When adding new contexts, remember to add them to both the exported
// variable definitions below AND the contexts slice.

// ContextEN16931 is the default context for basic UBL documents.
var ContextEN16931 = Context{
	CustomizationID: "urn:cen.eu:en16931:2017",
	Addons:          []cbc.Key{en16931.V2017},
	VESIDs: VESIDMapping{
		Invoice:    "eu.cen.en16931:ubl:1.3.14-2",
		CreditNote: "eu.cen.en16931:ubl-creditnote:1.3.15",
	},
}

// ContextPeppol defines the default Peppol context.
var ContextPeppol = Context{
	CustomizationID:           "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0",
	SelfBilledCustomizationID: "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:selfbilling:3.0",
	ProfileID:                 PeppolBillingProfileIDDefault,
	Addons:                    []cbc.Key{en16931.V2017},
	VESIDs: VESIDMapping{
		Invoice:              "eu.peppol.bis3:invoice:2025.5",
		InvoiceSelfBilled:    "eu.peppol.bis3:invoice-self-billing:2025.3",
		CreditNote:           "eu.peppol.bis3:creditnote:2025.5",
		CreditNoteSelfBilled: "eu.peppol.bis3:creditnote-self-billing:2025.3",
	},
}

// ContextXRechnung defines the main context to use for XRechnung UBL documents.
var ContextXRechnung = Context{
	CustomizationID: "urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0",
	ProfileID:       PeppolBillingProfileIDDefault,
	Addons:          []cbc.Key{xrechnung.V3},
	VESIDs: VESIDMapping{
		Invoice:    "de.xrechnung:ubl-invoice:3.0.2",
		CreditNote: "de.xrechnung:ubl-creditnote:3.0.2",
	},
}

// ContextPeppolFranceCIUS defines the context for France UBL Invoice CIUS.
var ContextPeppolFranceCIUS = Context{
	CustomizationID: "urn:cen.eu:en16931:2017#compliant#urn:peppol:france:billing:cius:1.0",
	ProfileID:       "urn:peppol:france:billing:regulated",
	Addons:          []cbc.Key{en16931.V2017},
	VESIDs: VESIDMapping{
		Invoice:    "fr.ctc:ubl-invoice:1.2",
		CreditNote: "fr.ctc:ubl-creditnote:1.2",
	},
}

// ContextPeppolFranceExtended defines the context for France UBL Invoice Extended.
var ContextPeppolFranceExtended = Context{
	CustomizationID: "urn:cen.eu:en16931:2017#conformant#urn:peppol:france:billing:extended:1.0",
	ProfileID:       "urn:peppol:france:billing:regulated",
	Addons:          []cbc.Key{en16931.V2017},
	VESIDs: VESIDMapping{
		Invoice:    "fr.ctc:ubl-invoice:1.2",
		CreditNote: "fr.ctc:ubl-creditnote:1.2",
	},
}

// contexts is used internally for reverse lookups during parsing.
// When adding new contexts, remember to add them here AND as exported variables above.
var contexts = []Context{ContextEN16931, ContextPeppol, ContextXRechnung, ContextPeppolFranceCIUS, ContextPeppolFranceExtended}

// Is checks if two contexts are the same.
func (c *Context) Is(c2 Context) bool {
	return c.CustomizationID == c2.CustomizationID && c.ProfileID == c2.ProfileID
}

// GetCustomizationID returns the appropriate CustomizationID based on
// whether the invoice is self-billed or not.
func (c *Context) GetCustomizationID(inv *bill.Invoice) string {
	if c.SelfBilledCustomizationID != "" && inv.HasTags(tax.TagSelfBilled) {
		return c.SelfBilledCustomizationID
	}
	return c.CustomizationID
}

// GetProfileID returns the appropriate ProfileID based on
// whether the invoice is self-billed or not.
func (c *Context) GetProfileID(inv *bill.Invoice) string {
	if c.SelfBilledProfileID != "" && inv.HasTags(tax.TagSelfBilled) {
		return c.SelfBilledProfileID
	}
	return c.ProfileID
}

// GetVESID returns the appropriate VESID based on the invoice type
// and whether it's self-billed or not.
func (c *Context) GetVESID(inv *bill.Invoice) string {
	isSelfBilled := inv.HasTags(tax.TagSelfBilled)
	isCreditNote := inv.Type.In(bill.InvoiceTypeCreditNote)

	switch {
	case isCreditNote && isSelfBilled && c.VESIDs.CreditNoteSelfBilled != "":
		return c.VESIDs.CreditNoteSelfBilled
	case isCreditNote && c.VESIDs.CreditNote != "":
		return c.VESIDs.CreditNote
	case isSelfBilled && c.VESIDs.InvoiceSelfBilled != "":
		return c.VESIDs.InvoiceSelfBilled
	case c.VESIDs.Invoice != "":
		return c.VESIDs.Invoice
	default:
		return ""
	}
}

// FindContext looks up a context by CustomizationID and optionally ProfileID.
// Returns nil if no matching context is found. This method also handles
// self-billed CustomizationIDs and ProfileIDs.
func FindContext(customizationID string, profileID string) *Context {
	for i := range contexts {
		ctx := &contexts[i]

		// Check if CustomizationID matches (standard or self-billed)
		isStandard := ctx.CustomizationID == customizationID
		isSelfBilled := ctx.SelfBilledCustomizationID == customizationID

		if !isStandard && !isSelfBilled {
			continue
		}

		// If no profileID provided or context has no profileID, it's a match
		if profileID == "" || ctx.ProfileID == "" {
			return ctx
		}

		// Check ProfileID match based on which CustomizationID matched
		if isSelfBilled && ctx.SelfBilledProfileID != "" {
			if ctx.SelfBilledProfileID == profileID {
				return ctx
			}
		} else if ctx.ProfileID == profileID {
			return ctx
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

	doc, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, ErrUnsupportedDocumentType
	}

	// Check and add missing addons
	if err := ensureAddons(doc, o.context.Addons); err != nil {
		return nil, err
	}

	// Removes included taxes as they are not supported in UBL
	if err := doc.RemoveIncludedTaxes(); err != nil {
		return nil, fmt.Errorf("cannot convert invoice with included taxes: %w", err)
	}

	return ublInvoice(doc, o)
}

// ensureAddons checks if the invoice has all required addons and adds missing ones
func ensureAddons(inv *bill.Invoice, required []cbc.Key) error {
	if len(required) == 0 {
		return nil
	}

	var missing []cbc.Key
	existing := inv.GetAddons()
	for _, addon := range required {
		if !addon.In(existing...) {
			missing = append(missing, addon)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	inv.SetAddons(append(existing, missing...)...)
	if err := inv.Calculate(); err != nil {
		return fmt.Errorf("gobl invoice missing addon %v: %w", missing, err)
	}
	if err := inv.Validate(); err != nil {
		return fmt.Errorf("gobl invoice missing addon %v: %w", missing, err)
	}
	return nil
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
