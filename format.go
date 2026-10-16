package ubl

import (
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/schema"
	"github.com/invopop/gobl/tax"
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

// Specification defines the identifiers a UBL document carries for one
// specification, and the GOBL documents it applies to.
type Specification struct {
	// CustomizationID identifies the specification.
	CustomizationID string
	// ProfileID identifies the business process.
	ProfileID string
	// OutputCustomizationID optionally specifies a different CustomizationID
	// to write into the generated XML.
	OutputCustomizationID string
	// Addons required by the specification.
	Addons []cbc.Key
	// VESIDs used to validate the specification's documents.
	VESIDs VESIDMapping
	// Applies reports whether the specification applies to the GOBL
	// document. Nil applies to any document.
	Applies func(doc any) bool
}

// Format defines a UBL format: the CustomizationID and ProfileID its
// documents carry, and the functions that adjust the base import and export
// for the specification it represents.
type Format struct {
	// Key identifies the format in the GOBL convert register, with one layer
	// for each specification it builds on, e.g. "ubl+peppol+bis-v3".
	Key cbc.Key
	// Name of the format.
	Name i18n.String
	// Countries where the format applies. Empty means no restriction.
	Countries []l10n.Code
	// Import lists the schemas of the GOBL documents the format's documents
	// are imported into. Formats without any are never detected, such as
	// those whose documents only differ from another format's by their
	// container.
	Import []schema.ID
	// Export lists the schemas of the GOBL documents the format exports.
	Export []schema.ID
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
	// SubLines writes line breakdowns as EXTENDED-CTC-FR sub-invoice lines,
	// and reads them back from the line billing references that name the
	// invoice itself.
	SubLines bool
	// ExportFuncs adjust the UBL document exported from GOBL, in order, after
	// the base export.
	ExportFuncs []ExportFunc
	// ImportFuncs adjust the GOBL envelope imported from UBL, in order, after
	// the base import and before the document is calculated.
	ImportFuncs []ImportFunc
	// Specifications lists the other specifications of the format, for GOBL
	// documents its own identifiers do not cover. On export, the first that
	// applies to the document is used, or else the format's own identifiers.
	Specifications []Specification
}

// Is checks if two formats are the same.
func (f *Format) Is(f2 Format) bool {
	return f.CustomizationID == f2.CustomizationID && f.ProfileID == f2.ProfileID
}

// specifications provides the format's specifications, ending with the one
// defined by its own identifiers.
func (f *Format) specifications() []Specification {
	return append(append([]Specification{}, f.Specifications...), Specification{
		CustomizationID:       f.CustomizationID,
		ProfileID:             f.ProfileID,
		OutputCustomizationID: f.OutputCustomizationID,
		Addons:                f.Addons,
		VESIDs:                f.VESIDs,
	})
}

// SpecificationFor provides the specification the format exports the GOBL
// document with.
func (f *Format) SpecificationFor(doc any) *Specification {
	specs := f.specifications()
	for i := range specs {
		if specs[i].Applies == nil || specs[i].Applies(doc) {
			return &specs[i]
		}
	}
	return nil
}

// VESID provides the VESID to validate the UBL document exported from the
// GOBL document with.
func (f *Format) VESID(doc any) string {
	s := f.SpecificationFor(doc)
	switch d := doc.(type) {
	case *bill.Invoice:
		if d.Type.In(bill.InvoiceTypeCreditNote) {
			return s.VESIDs.CreditNote
		}
		return s.VESIDs.Invoice
	case *bill.Status:
		return s.VESIDs.Status
	}
	return ""
}

// specificationFrom provides the specification a document with the
// identifiers was written with, or the format's own if none declares them.
func (f *Format) specificationFrom(customizationID, profileID string) Specification {
	specs := f.specifications()
	for _, s := range specs {
		if s.matches(customizationID, profileID) {
			return s
		}
	}
	for _, s := range specs {
		if s.OutputCustomizationID != "" && s.OutputCustomizationID == customizationID {
			return s
		}
	}
	return specs[len(specs)-1]
}

// matches checks the CustomizationID and, when both are set, the ProfileID.
func (s Specification) matches(customizationID, profileID string) bool {
	if s.CustomizationID != customizationID {
		return false
	}
	return s.ProfileID == "" || profileID == "" || s.ProfileID == profileID
}

// FindFormat looks up a registered format by CustomizationID and optionally
// ProfileID, among those with Import schemas. Returns nil if no matching
// format is found.
//
// The lookup logic works as follows:
//  1. Formats whose Match function claims the document
//  2. Tries to match on the full CustomizationID of any of the format's
//     specifications (for external identification)
//  3. If not found, tries to match on OutputCustomizationID (for parsing incoming documents)
//  4. Formats whose Fallback function claims the document
func FindFormat(customizationID string, profileID string) *Format {
	list := make([]*Format, 0, len(formats))
	for i := range formats {
		if len(formats[i].Import) > 0 {
			list = append(list, &formats[i])
		}
	}

	for _, f := range list {
		if f.Match != nil && f.Match(customizationID, profileID) {
			return f
		}
	}

	// First pass: try to match on full CustomizationID
	for _, f := range list {
		for _, s := range f.specifications() {
			if s.matches(customizationID, profileID) {
				return f
			}
		}
	}

	// Second pass: try to match on OutputCustomizationID (for parsing incoming documents)
	for _, f := range list {
		for _, s := range f.specifications() {
			if s.OutputCustomizationID != "" && s.OutputCustomizationID == customizationID {
				return f
			}
		}
	}

	for _, f := range list {
		if f.Fallback != nil && f.Fallback(customizationID, profileID) {
			return f
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
	format      Format
	spec        Specification
	from        cbc.URI
	to          cbc.URI
	attachments []BinaryAttachment
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

// WithAttachments embeds files, such as a PDF version of the invoice, in
// an exported invoice. Other documents cannot embed them.
func WithAttachments(atts ...BinaryAttachment) Option {
	return func(o *options) {
		o.attachments = append(o.attachments, atts...)
	}
}

// FormatEN16931 is the default format for basic UBL documents.
var FormatEN16931 = Format{
	Key:             "ubl+en16931-v2017",
	Name:            i18n.NewString("UBL EN 16931:2017"),
	Import:          []schema.ID{invoiceSchema},
	Export:          []schema.ID{invoiceSchema},
	CustomizationID: "urn:cen.eu:en16931:2017",
	Addons:          []cbc.Key{en16931.V2017},
	VESIDs: VESIDMapping{
		Invoice:    "eu.cen.en16931:ubl:1.3.16",
		CreditNote: "eu.cen.en16931:ubl-creditnote:1.3.16",
	},
	ExportFuncs: []ExportFunc{FoldToCore},
}

// FormatPeppol is the Peppol BIS 3 format. Invoices are written with Peppol
// BIS Billing 3.0, self-billed invoices with Peppol BIS Self-Billing 3.0,
// and statuses with Peppol Invoice Response 3.
var FormatPeppol = Format{
	Key:             "ubl+peppol+bis-v3",
	Name:            i18n.NewString("UBL Peppol BIS 3"),
	Import:          []schema.ID{invoiceSchema, statusSchema},
	Export:          []schema.ID{invoiceSchema, statusSchema},
	CustomizationID: "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0",
	ProfileID:       PeppolBillingProfileIDDefault,
	Addons:          []cbc.Key{en16931.V2017},
	VESIDs: VESIDMapping{
		Invoice:    "eu.peppol.bis3:invoice:2026.5",
		CreditNote: "eu.peppol.bis3:creditnote:2026.5",
	},
	Specifications: []Specification{
		SpecificationPeppolSelfBilling,
		SpecificationPeppolInvoiceResponse,
	},
	ExportFuncs: []ExportFunc{exportPeppolBilling, exportPeppolInvoiceResponse, FoldToCore},
	ImportFuncs: []ImportFunc{importPeppolInvoiceResponse},
}

// SpecificationPeppolSelfBilling is Peppol BIS Self-Billing 3.0, used for
// self-billed invoices.
var SpecificationPeppolSelfBilling = Specification{
	CustomizationID: "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:selfbilling:3.0",
	ProfileID:       "urn:fdc:peppol.eu:2017:poacc:selfbilling:01:1.0",
	Addons:          []cbc.Key{en16931.V2017},
	// The Peppol self-billing rule sets are versioned independently from the
	// regular invoice/credit-note ones.
	VESIDs: VESIDMapping{
		Invoice:    "eu.peppol.bis3:invoice-self-billing:2026.5",
		CreditNote: "eu.peppol.bis3:creditnote-self-billing:2026.5",
	},
	Applies: func(doc any) bool {
		inv, ok := doc.(*bill.Invoice)
		return ok && inv.HasTags(tax.TagSelfBilled)
	},
}

// SpecificationPeppolInvoiceResponse is Peppol Invoice Response 3, used for
// statuses.
var SpecificationPeppolInvoiceResponse = Specification{
	CustomizationID: "urn:fdc:peppol.eu:poacc:trns:invoice_response:3",
	ProfileID:       "urn:fdc:peppol.eu:poacc:bis:invoice_response:3",
	VESIDs: VESIDMapping{
		Status: "eu.peppol.bis3:invoice-message-response:2026.5",
	},
	Applies: func(doc any) bool {
		_, ok := doc.(*bill.Status)
		return ok
	},
}

// formats holds every registered format for lookups during parsing.
var formats []Format

func init() {
	registerFormats(true, []Format{
		FormatEN16931,
		FormatPeppol,
	})
}
