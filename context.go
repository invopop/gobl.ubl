package ubl

import (
	"github.com/invopop/gobl/addons/de/xrechnung"
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/addons/fr/facturx"
	"github.com/invopop/gobl/cbc"
)

// Peppol Billing Profile IDs
const (
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
	// OutputCustomizationID optionally specifies a different CustomizationID
	// to use in the actual generated UBL XML document. If empty, CustomizationID
	// is used. This allows the context to be identified by one ID externally while
	// generating different values in the XML output.
	OutputCustomizationID string
	// Addons contains the list of Addons required for this CustomizationID
	// and ProfileID.
	Addons []cbc.Key
}

// Is checks if two contexts are the same.
func (c *Context) Is(c2 Context) bool {
	return c.CustomizationID == c2.CustomizationID && c.ProfileID == c2.ProfileID
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
	CustomizationID:       "urn:cen.eu:en16931:2017#compliant#urn:peppol:france:billing:cius:1.0",
	ProfileID:             "urn:peppol:france:billing:regulated",
	OutputCustomizationID: "urn:cen.eu:en16931:2017",
	Addons:                []cbc.Key{en16931.V2017},
}

// ContextPeppolFranceExtended defines the context for France UBL Invoice Extended.
var ContextPeppolFranceExtended = Context{
	CustomizationID:       "urn:cen.eu:en16931:2017#conformant#urn:peppol:france:billing:extended:1.0",
	ProfileID:             "urn:peppol:france:billing:regulated",
	OutputCustomizationID: "urn:cen.eu:en16931:2017#conformant#urn.cpro.gouv.fr:1p0:extended-ctc-fr",
	Addons:                []cbc.Key{facturx.V1},
}

// contexts is used internally for reverse lookups during parsing.
// When adding new contexts, remember to add them here AND as exported variables above.
var contexts = []Context{ContextEN16931, ContextPeppol, ContextXRechnung, ContextPeppolFranceCIUS, ContextPeppolFranceExtended}
