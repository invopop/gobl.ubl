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

// Format defines a UBL format: the CustomizationID and ProfileID its
// documents carry, and the functions that adjust the base import and export
// for the specification it represents.
type Format struct {
	// Key identifies the format in the GOBL convert register, with one layer
	// for each specification it builds on, e.g. "ubl+peppol".
	Key cbc.Key
	// Name of the format.
	Name i18n.String
	// Countries where the format applies. Empty means no restriction.
	Countries []l10n.Code
	// Schemas of the GOBL documents the format converts, in both directions.
	Schemas []schema.ID
	// CustomizationID identifies specific characteristics in the
	// document which need to be present for local differences.
	CustomizationID string
	// ProfileID determines the business process context or scenario
	// for the exchange of the document.
	ProfileID string
	// OutputCustomizationID optionally specifies a different CustomizationID
	// to use in the actual generated UBL XML document. If empty, CustomizationID
	// is used. This allows the format to be identified by one ID externally while
	// generating different values in the XML output.
	OutputCustomizationID string
	// Addons contains the list of Addons required for this CustomizationID
	// and ProfileID.
	Addons []cbc.Key
	// VESIDs contains the VESID (Validation Exchange Specification ID) mappings
	// for different document types and scenarios within this format.
	VESIDs VESIDMapping
	// Match optionally identifies the format's documents before the
	// CustomizationID and ProfileID are compared.
	Match func(customizationID, profileID string) bool
	// Fallback optionally claims documents that no format matched.
	Fallback func(customizationID, profileID string) bool
	// ExportFuncs adjust the UBL document exported from GOBL, in order, after
	// the base export.
	ExportFuncs []ExportFunc
	// ImportFuncs adjust the GOBL envelope imported from UBL, in order, after
	// the base import and before the document is calculated.
	ImportFuncs []ImportFunc
}

// Is checks if two formats are the same.
func (f *Format) Is(f2 Format) bool {
	return f.CustomizationID == f2.CustomizationID && f.ProfileID == f2.ProfileID
}

// GetVESID returns the appropriate VESID based on the invoice type.
func (f *Format) GetVESID(inv *bill.Invoice) string {
	if inv.Type.In(bill.InvoiceTypeCreditNote) {
		return f.VESIDs.CreditNote
	}
	return f.VESIDs.Invoice
}

// FindFormat looks up a registered format by CustomizationID and optionally
// ProfileID. Returns nil if no matching format is found.
//
// The lookup logic works as follows:
//  1. Formats whose Match function claims the document
//  2. Tries to match on the full CustomizationID (for external identification)
//  3. If not found, tries to match on OutputCustomizationID (for parsing incoming documents)
//  4. Formats whose Fallback function claims the document
func FindFormat(customizationID string, profileID string) *Format {
	for _, f := range formats {
		if f.Match != nil && f.Match(customizationID, profileID) {
			return &f
		}
	}

	// First pass: try to match on full CustomizationID
	for _, f := range formats {
		if f.CustomizationID == customizationID {
			// If format has a ProfileID and one was provided, they must match
			if f.ProfileID != "" && profileID != "" && f.ProfileID != profileID {
				continue
			}
			return &f
		}
	}

	// Second pass: try to match on OutputCustomizationID (for parsing incoming documents)
	for _, f := range formats {
		if f.OutputCustomizationID != "" && f.OutputCustomizationID == customizationID {
			return &f
		}
	}

	for _, f := range formats {
		if f.Fallback != nil && f.Fallback(customizationID, profileID) {
			return &f
		}
	}

	return nil
}

// RegisterFormats makes the formats available to FindFormat, and registers
// them with the GOBL convert register. Packages that implement regional
// formats call it from their init function.
func RegisterFormats(fs ...Format) {
	registerFormats(false, fs)
}

func registerFormats(fallback bool, fs []Format) {
	formats = append(formats, fs...)
	convert.Register(&converter{formats: fs, fallback: fallback})
}

type options struct {
	format Format
	from   cbc.URI
	to     cbc.URI
}

// Option is used to define configuration options to use during
// conversion processes.
type Option func(*options)

// WithFormat sets the format to use for the configuration
// and business profile.
func WithFormat(f Format) Option {
	return func(o *options) {
		o.format = f
	}
}

// WithRouting supplies the transport addresses the receiving app was routed
// with (the Peppol SBD From / To — who sent the document, who received it).
// They are recorded verbatim on the parsed envelope's Head.From / Head.To,
// regardless of the document type, so a received document is never mislabelled
// with GOBL's document-derived, outgoing-direction guess. See Import.
func WithRouting(from, to cbc.URI) Option {
	return func(o *options) {
		o.from = from
		o.to = to
	}
}

// FormatEN16931 is the default format for basic UBL documents.
var FormatEN16931 = Format{
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

// FormatPeppol defines the default Peppol format.
var FormatPeppol = Format{
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
	ExportFuncs: []ExportFunc{exportPeppolBilling},
}

// FormatPeppolSelfBilled defines the Peppol self-billed format.
var FormatPeppolSelfBilled = Format{
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

// FormatPeppolInvoiceResponse defines the Peppol BIS Invoice Response format.
// It is its own format (separate from the billing FormatPeppol) because the
// Invoice Response declares a different CustomizationID, which is what
// FindFormat matches a parsed document against.
var FormatPeppolInvoiceResponse = Format{
	Key:             "ubl+peppol+invoice-response",
	Name:            i18n.NewString("UBL Peppol Invoice Response 3"),
	Schemas:         []schema.ID{statusSchema},
	CustomizationID: "urn:fdc:peppol.eu:poacc:trns:invoice_response:3",
	ProfileID:       "urn:fdc:peppol.eu:poacc:bis:invoice_response:3",
	VESIDs: VESIDMapping{
		Status: "eu.peppol.bis3:invoice-message-response:2026.5",
	},
	ExportFuncs: []ExportFunc{exportPeppolInvoiceResponse},
	ImportFuncs: []ImportFunc{importPeppolInvoiceResponse},
}

// formats holds every registered format for lookups during parsing.
var formats []Format

func init() {
	registerFormats(true, []Format{
		FormatEN16931,
		FormatPeppol,
		FormatPeppolSelfBilled,
		FormatPeppolInvoiceResponse,
	})
}
