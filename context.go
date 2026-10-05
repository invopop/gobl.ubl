package ubl

import (
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/schema"
)

// Peppol Billing Profile IDs
const (
	PeppolBillingProfileIDDefault = "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0"
)

// VESIDMapping maps document types to their corresponding VESID values.
type VESIDMapping struct {
	// Invoice is the VESID for invoices
	Invoice string
	// CreditNote is the VESID for credit notes
	CreditNote string
	// Status is the VESID for status documents (UBL ApplicationResponses)
	Status string
}

// Context is used to ensure that the generated UBL document
// uses a specific CustomizationID and ProfileID when generating
// the output document, and to apply the layers of the specification it
// represents on top of the base conversion.
type Context struct {
	// Key identifies the context in the GOBL convert register, with one layer
	// for each specification it builds on, e.g. "ubl+peppol".
	Key cbc.Key
	// Name of the context.
	Name i18n.String
	// Countries where the context applies. Empty means no restriction.
	Countries []l10n.Code
	// Schemas of the GOBL documents the context converts, in both directions.
	Schemas []schema.ID
	// CustomizationID identifies specific characteristics in the
	// document which need to be present for local differences.
	CustomizationID string
	// ProfileID determines the business process context or scenario
	// for the exchange of the document.
	ProfileID string
	// OutputCustomizationID optionally specifies a different CustomizationID
	// to use in the actual generated UBL XML document. If empty, CustomizationID
	// is used. This allows the context to be identified by one ID externally while
	// generating different values in the XML output.
	OutputCustomizationID string
	// Addons contains the list of Addons required for this CustomizationID
	// and ProfileID.
	Addons []cbc.Key
	// VESIDs contains the VESID (Validation Exchange Specification ID) mappings
	// for different document types and scenarios within this context.
	VESIDs VESIDMapping
	// Match optionally identifies the context's documents before the
	// CustomizationID and ProfileID are compared.
	Match func(customizationID, profileID string) bool
	// Fallback optionally claims documents that no context matched.
	Fallback func(customizationID, profileID string) bool
	// Layers add the behavior of the context's specifications, applied in
	// order after the base conversion.
	Layers []*Layer
}

// Is checks if two contexts are the same.
func (c *Context) Is(c2 Context) bool {
	return c.CustomizationID == c2.CustomizationID && c.ProfileID == c2.ProfileID
}

// GetVESID returns the appropriate VESID based on the invoice type.
func (c *Context) GetVESID(inv *bill.Invoice) string {
	if inv.Type.In(bill.InvoiceTypeCreditNote) {
		return c.VESIDs.CreditNote
	}
	return c.VESIDs.Invoice
}

// FindContext looks up a registered context by CustomizationID and optionally
// ProfileID. Returns nil if no matching context is found.
//
// The lookup logic works as follows:
//  1. Contexts whose Match function claims the document
//  2. Tries to match on the full CustomizationID (for external identification)
//  3. If not found, tries to match on OutputCustomizationID (for parsing incoming documents)
//  4. Contexts whose Fallback function claims the document
func FindContext(customizationID string, profileID string) *Context {
	for _, ctx := range contexts {
		if ctx.Match != nil && ctx.Match(customizationID, profileID) {
			return &ctx
		}
	}

	// First pass: try to match on full CustomizationID
	for _, ctx := range contexts {
		if ctx.CustomizationID == customizationID {
			// If context has a ProfileID and one was provided, they must match
			if ctx.ProfileID != "" && profileID != "" && ctx.ProfileID != profileID {
				continue
			}
			return &ctx
		}
	}

	// Second pass: try to match on OutputCustomizationID (for parsing incoming documents)
	for _, ctx := range contexts {
		if ctx.OutputCustomizationID != "" && ctx.OutputCustomizationID == customizationID {
			return &ctx
		}
	}

	for _, ctx := range contexts {
		if ctx.Fallback != nil && ctx.Fallback(customizationID, profileID) {
			return &ctx
		}
	}

	return nil
}

// RegisterContexts makes the contexts available to FindContext, and registers
// them with the GOBL convert register. Packages that implement regional
// contexts call it from their init function.
func RegisterContexts(ctxs ...Context) {
	registerContexts(false, ctxs)
}

func registerContexts(fallback bool, ctxs []Context) {
	contexts = append(contexts, ctxs...)
	convert.Register(&converter{contexts: ctxs, fallback: fallback})
}

type options struct {
	context Context
	from    cbc.URI
	to      cbc.URI
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

// WithRouting supplies the transport addresses the receiving app was routed
// with (the Peppol SBD From / To — who sent the document, who received it).
// They are recorded verbatim on the parsed envelope's Head.From / Head.To,
// regardless of the document type, so a received document is never mislabelled
// with GOBL's document-derived, outgoing-direction guess. See Invoice.Convert.
func WithRouting(from, to cbc.URI) Option {
	return func(o *options) {
		o.from = from
		o.to = to
	}
}

// ContextEN16931 is the default context for basic UBL documents.
var ContextEN16931 = Context{
	Key:             "ubl+en16931",
	Name:            i18n.NewString("UBL EN 16931"),
	Schemas:         []schema.ID{invoiceSchema},
	CustomizationID: "urn:cen.eu:en16931:2017",
	Addons:          []cbc.Key{en16931.V2017},
	VESIDs: VESIDMapping{
		Invoice:    "eu.cen.en16931:ubl:1.3.16",
		CreditNote: "eu.cen.en16931:ubl-creditnote:1.3.16",
	},
}

// ContextPeppol defines the default Peppol context.
var ContextPeppol = Context{
	Key:             "ubl+peppol",
	Name:            i18n.NewString("UBL Peppol BIS Billing 3"),
	Schemas:         []schema.ID{invoiceSchema},
	CustomizationID: "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0",
	ProfileID:       PeppolBillingProfileIDDefault,
	Addons:          []cbc.Key{en16931.V2017},
	VESIDs: VESIDMapping{
		Invoice:    "eu.peppol.bis3:invoice:2026.5",
		CreditNote: "eu.peppol.bis3:creditnote:2026.5",
	},
	Layers: []*Layer{LayerPeppolBilling},
}

// ContextPeppolSelfBilled defines the Peppol self-billed context.
var ContextPeppolSelfBilled = Context{
	Key:             "ubl+peppol+self-billing",
	Name:            i18n.NewString("UBL Peppol BIS Self-Billing 3"),
	Schemas:         []schema.ID{invoiceSchema},
	CustomizationID: "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:selfbilling:3.0",
	ProfileID:       "urn:fdc:peppol.eu:2017:poacc:selfbilling:01:1.0",
	Addons:          []cbc.Key{en16931.V2017},
	// The Peppol self-billing rule sets are versioned independently from the
	// regular invoice/credit-note ones. phive-rules keeps only a rolling window
	// of releases, so the older ":2025.3" sets were dropped in phive-rules 4.3.x;
	// ":2026.3" is the oldest still published and validates the same documents.
	VESIDs: VESIDMapping{
		Invoice:    "eu.peppol.bis3:invoice-self-billing:2026.5",
		CreditNote: "eu.peppol.bis3:creditnote-self-billing:2026.5",
	},
}

// ContextPeppolInvoiceResponse defines the Peppol BIS Invoice Response context.
// It is its own context (separate from the billing ContextPeppol) because the
// Invoice Response declares a different CustomizationID, which is what
// FindContext matches a parsed document against.
var ContextPeppolInvoiceResponse = Context{
	Key:             "ubl+peppol+invoice-response",
	Name:            i18n.NewString("UBL Peppol Invoice Response 3"),
	Schemas:         []schema.ID{statusSchema},
	CustomizationID: "urn:fdc:peppol.eu:poacc:trns:invoice_response:3",
	ProfileID:       "urn:fdc:peppol.eu:poacc:bis:invoice_response:3",
	VESIDs: VESIDMapping{
		Status: "eu.peppol.bis3:invoice-message-response:2026.5",
	},
	Layers: []*Layer{LayerPeppolInvoiceResponse},
}

// contexts holds every registered context for lookups during parsing.
var contexts []Context

func init() {
	registerContexts(true, []Context{
		ContextEN16931,
		ContextPeppol,
		ContextPeppolSelfBilled,
		ContextPeppolInvoiceResponse,
	})
}
