package ubl

import (
	"github.com/invopop/gobl/addons/de/xrechnung"
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/addons/fr/facturx"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
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
}

// Context is used to ensure that the generated UBL document
// uses a specific CustomizationID and ProfileID when generating
// the output document.
type Context struct {
	// CustomizationID identifies specific characteristics in the
	// document which need to be present for local differences.
	CustomizationID string
	// ProfileID determines the business process context or scenario
	// for the exchange of the document
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

// FindContext looks up a context by CustomizationID and optionally ProfileID.
// Returns nil if no matching context is found.
//
// The lookup logic works as follows:
// 1. First tries to match on the full CustomizationID (for external identification)
// 2. If not found, tries to match on OutputCustomizationID (for parsing incoming documents)
// 3. For contexts with a ProfileID, checks if it matches (if provided)
func FindContext(customizationID string, profileID string) *Context {
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

	// Second pass: try to match on OutputCustomizationID (for parsing where Profile may not be added))
	for _, ctx := range contexts {
		if ctx.OutputCustomizationID != "" && ctx.OutputCustomizationID == customizationID {
			return &ctx
		}
	}

	return nil
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
	CustomizationID: "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0",
	ProfileID:       PeppolBillingProfileIDDefault,
	Addons:          []cbc.Key{en16931.V2017},
	VESIDs: VESIDMapping{
		Invoice:    "eu.peppol.bis3:invoice:2025.5",
		CreditNote: "eu.peppol.bis3:creditnote:2025.5",
	},
}

// ContextPeppolSelfBilled defines the Peppol self-billed context.
var ContextPeppolSelfBilled = Context{
	CustomizationID: "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:selfbilling:3.0",
	ProfileID:       "urn:fdc:peppol.eu:2017:poacc:selfbilling:01:1.0",
	Addons:          []cbc.Key{en16931.V2017},
	VESIDs: VESIDMapping{
		Invoice:    "eu.peppol.bis3:invoice-self-billing:2025.3",
		CreditNote: "eu.peppol.bis3:creditnote-self-billing:2025.3",
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
	CustomizationID:       "urn:cen.eu:en16931:2017#compliant#urn:peppol:france:billing:cius:1.0",
	ProfileID:             "urn:peppol:france:billing:regulated",
	OutputCustomizationID: "urn:cen.eu:en16931:2017",
	Addons:                []cbc.Key{en16931.V2017},
	VESIDs: VESIDMapping{
		Invoice:    "fr.ctc:ubl-invoice:1.2",
		CreditNote: "fr.ctc:ubl-creditnote:1.2",
	},
}

// ContextPeppolFranceExtended defines the context for France UBL Invoice Extended.
var ContextPeppolFranceExtended = Context{
	CustomizationID:       "urn:cen.eu:en16931:2017#conformant#urn:peppol:france:billing:extended:1.0",
	ProfileID:             "urn:peppol:france:billing:regulated",
	OutputCustomizationID: "urn:cen.eu:en16931:2017#conformant#urn.cpro.gouv.fr:1p0:extended-ctc-fr",
	Addons:                []cbc.Key{facturx.V1},
	VESIDs: VESIDMapping{
		Invoice:    "fr.ctc:ubl-invoice:1.2",
		CreditNote: "fr.ctc:ubl-creditnote:1.2",
	},
}

// ContextOIOUBL defines the context for OIOUBL (Nemhandel) UBL documents.
var ContextOIOUBL = Context{
	CustomizationID: "urn:fdc:oioubl.dk:trns:billing:invoice:3.0",
	ProfileID:       "urn:fdc:oioubl.dk:bis:billing_with_response:3",
	Addons:          []cbc.Key{en16931.V2017},
}

// ContextOIOUBL21 defines the context for legacy OIOUBL 2.1 documents.
var ContextOIOUBL21 = Context{
	CustomizationID: "OIOUBL-2.1",
	ProfileID:       "urn:www.nesubl.eu:profiles:profile5:ver2.0",
	Addons:          []cbc.Key{en16931.V2017},
}

// IsOIOUBL reports whether a context is any supported OIOUBL variant.
func (c *Context) IsOIOUBL() bool {
	return c.Is(ContextOIOUBL) || c.Is(ContextOIOUBL21)
}

// contexts is used internally for reverse lookups during parsing.
// When adding new contexts, remember to add them here AND as exported variables above.
var contexts = []Context{ContextEN16931, ContextPeppol, ContextPeppolSelfBilled, ContextXRechnung, ContextPeppolFranceCIUS, ContextPeppolFranceExtended, ContextOIOUBL, ContextOIOUBL21}
