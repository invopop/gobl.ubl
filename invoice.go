package ubl

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl.fr.ctc/addon/dgfip"
	zatca "github.com/invopop/gobl.sa.zatca/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	cur "github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/tax"
)

// Main UBL Invoice Namespace
const (
	NamespaceUBLInvoice    = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
	NamespaceUBLCreditNote = "urn:oasis:names:specification:ubl:schema:xsd:CreditNote-2"
)

// XML root element local names for the supported UBL document types.
const (
	rootNameInvoice    = "Invoice"
	rootNameCreditNote = "CreditNote"
)

// Schema location and customization constants
const (
	SchemaLocationInvoice     = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2 http://docs.oasis-open.org/ubl/os-UBL-2.1/xsd/maindoc/UBL-Invoice-2.1.xsd"
	SchemaLocationCrediteNote = "urn:oasis:names:specification:ubl:schema:xsd:CreditNote-2 https://docs.oasis-open.org/ubl/os-UBL-2.1/xsd/maindoc/UBL-CreditNote-2.1.xsd"
)

// DocumentHeader is the leading element run shared, in identical sequence, by
// the UBL Invoice and CreditNote XSDs (the root attributes through
// cbc:IssueTime). Both document structs embed it so the shared prefix can never
// drift between the two types.
type DocumentHeader struct {
	// Attributes
	CACNamespace   string `xml:"xmlns:cac,attr"`
	CBCNamespace   string `xml:"xmlns:cbc,attr"`
	QDTNamespace   string `xml:"xmlns:qdt,attr"`
	UDTNamespace   string `xml:"xmlns:udt,attr"`
	CCTSNamespace  string `xml:"xmlns:ccts,attr"`
	UBLNamespace   string `xml:"xmlns,attr"`
	XSINamespace   string `xml:"xmlns:xsi,attr"`
	EXTNamespace   string `xml:"xmlns:ext,attr"`
	SchemaLocation string `xml:"xsi:schemaLocation,attr,omitempty"`

	Extensions         *Extensions `xml:"ext:UBLExtensions,omitempty"`
	UBLVersionID       string      `xml:"cbc:UBLVersionID,omitempty"`
	CustomizationID    string      `xml:"cbc:CustomizationID,omitempty"`
	ProfileID          string      `xml:"cbc:ProfileID,omitempty"`
	ProfileExecutionID string      `xml:"cbc:ProfileExecutionID,omitempty"`
	ID                 string      `xml:"cbc:ID"`
	CopyIndicator      bool        `xml:"cbc:CopyIndicator,omitempty"`
	UUID               string      `xml:"cbc:UUID,omitempty"`
	IssueDate          string      `xml:"cbc:IssueDate"`
	IssueTime          string      `xml:"cbc:IssueTime,omitempty"`
}

// DocumentCurrency is the currency/accounting element run shared, in identical
// sequence, by both XSDs (cbc:DocumentCurrencyCode through cac:InvoicePeriod).
type DocumentCurrency struct {
	DocumentCurrencyCode           string   `xml:"cbc:DocumentCurrencyCode,omitempty"`
	TaxCurrencyCode                string   `xml:"cbc:TaxCurrencyCode,omitempty"`
	PricingCurrencyCode            string   `xml:"cbc:PricingCurrencyCode,omitempty"`
	PaymentCurrencyCode            string   `xml:"cbc:PaymentCurrencyCode,omitempty"`
	PaymentAlternativeCurrencyCode string   `xml:"cbc:PaymentAlternativeCurrencyCode,omitempty"`
	AccountingCostCode             string   `xml:"cbc:AccountingCostCode,omitempty"`
	AccountingCost                 string   `xml:"cbc:AccountingCost,omitempty"`
	LineCountNumeric               int      `xml:"cbc:LineCountNumeric,omitempty"`
	BuyerReference                 string   `xml:"cbc:BuyerReference,omitempty"`
	InvoicePeriod                  []Period `xml:"cac:InvoicePeriod,omitempty"`
}

// DocumentParties is the party/delivery/payment element run shared, in identical
// sequence, by both XSDs (cac:Signature through cac:PaymentTerms).
type DocumentParties struct {
	Signature               []Signature    `xml:"cac:Signature,omitempty"`
	AccountingSupplierParty SupplierParty  `xml:"cac:AccountingSupplierParty"`
	AccountingCustomerParty CustomerParty  `xml:"cac:AccountingCustomerParty"`
	PayeeParty              *Party         `xml:"cac:PayeeParty,omitempty"`
	BuyerCustomerParty      *CustomerParty `xml:"cac:BuyerCustomerParty,omitempty"`
	SellerSupplierParty     *SupplierParty `xml:"cac:SellerSupplierParty,omitempty"`
	TaxRepresentativeParty  *Party         `xml:"cac:TaxRepresentativeParty,omitempty"`
	Delivery                []*Delivery    `xml:"cac:Delivery,omitempty"`
	DeliveryTerms           *DeliveryTerms `xml:"cac:DeliveryTerms,omitempty"`
	PaymentMeans            []PaymentMeans `xml:"cac:PaymentMeans,omitempty"`
	PaymentTerms            *PaymentTerms  `xml:"cac:PaymentTerms,omitempty"`
}

// Invoice represents a UBL Invoice document, with fields declared in the exact
// sequence of the UBL-Invoice-2.1 XSD.
//
// It doubles as the parse target for **both** Invoice and CreditNote XML:
// unmarshalling is order-independent, so the extra CreditNote-only fields it
// carries (CreditNoteTypeCode, CreditNoteLines) are populated when a credit note
// is parsed and simply stay empty — and therefore omitted — when marshalling
// a real invoice. Marshalling a credit note goes through CreditNote (see
// creditnote.go), which lays the divergent elements out in CreditNote-XSD order.
//
// The shared element runs live in the embedded DocumentHeader, DocumentCurrency
// and DocumentParties types. Because their fields are promoted, a keyed struct
// literal cannot set them directly (e.g. Invoice{Signature: ...} does not
// compile); construct via the embedded type (Invoice{DocumentParties:
// DocumentParties{Signature: ...}}) or assign the promoted field after
// construction (inv.Signature = ...). Most callers should use Convert instead.
type Invoice struct {
	XMLName xml.Name
	DocumentHeader

	DueDate string `xml:"cbc:DueDate,omitempty"`

	InvoiceTypeCode    *IDType `xml:"cbc:InvoiceTypeCode,omitempty"`
	CreditNoteTypeCode *IDType `xml:"cbc:CreditNoteTypeCode,omitempty"`

	Note         []string `xml:"cbc:Note,omitempty"`
	TaxPointDate string   `xml:"cbc:TaxPointDate,omitempty"`

	DocumentCurrency

	OrderReference              *OrderReference     `xml:"cac:OrderReference,omitempty"`
	BillingReference            []*BillingReference `xml:"cac:BillingReference,omitempty"`
	DespatchDocumentReference   []Reference         `xml:"cac:DespatchDocumentReference,omitempty"`
	ReceiptDocumentReference    []Reference         `xml:"cac:ReceiptDocumentReference,omitempty"`
	StatementDocumentReference  []Reference         `xml:"cac:StatementDocumentReference,omitempty"`
	OriginatorDocumentReference []Reference         `xml:"cac:OriginatorDocumentReference,omitempty"`
	ContractDocumentReference   []Reference         `xml:"cac:ContractDocumentReference,omitempty"`
	AdditionalDocumentReference []Reference         `xml:"cac:AdditionalDocumentReference,omitempty"`
	ProjectReference            []ProjectReference  `xml:"cac:ProjectReference,omitempty"`

	DocumentParties

	PrepaidPayment                 []PrepaidPayment  `xml:"cac:PrepaidPayment,omitempty"`
	AllowanceCharge                []AllowanceCharge `xml:"cac:AllowanceCharge,omitempty"`
	TaxExchangeRate                *ExchangeRate     `xml:"cac:TaxExchangeRate,omitempty"`
	PricingExchangeRate            *ExchangeRate     `xml:"cac:PricingExchangeRate,omitempty"`
	PaymentExchangeRate            *ExchangeRate     `xml:"cac:PaymentExchangeRate,omitempty"`
	PaymentAlternativeExchangeRate *ExchangeRate     `xml:"cac:PaymentAlternativeExchangeRate,omitempty"`
	TaxTotal                       []TaxTotal        `xml:"cac:TaxTotal,omitempty"`
	WithholdingTaxTotal            []TaxTotal        `xml:"cac:WithholdingTaxTotal,omitempty"`
	LegalMonetaryTotal             MonetaryTotal     `xml:"cac:LegalMonetaryTotal"`
	InvoiceLines                   []InvoiceLine     `xml:"cac:InvoiceLine,omitempty"`
	CreditNoteLines                []InvoiceLine     `xml:"cac:CreditNoteLine,omitempty"`
}

func ublInvoice(inv *bill.Invoice, o *options) (*Invoice, error) {
	tc, err := getTypeCode(inv)
	if err != nil {
		return nil, err
	}

	// Determine CustomizationID to use in output
	customizationID := o.context.CustomizationID
	if o.context.OutputCustomizationID != "" {
		customizationID = o.context.OutputCustomizationID
	}

	// Determine ProfileID to use in output
	// First check meta field, then fall back to context
	profileID := o.context.ProfileID
	if o.context.Is(ContextPeppolFranceCIUS) || o.context.Is(ContextPeppolFranceExtended) {
		if profile := inv.Tax.GetExt(dgfip.ExtKeyBillingMode); profile != cbc.CodeEmpty {
			profileID = profile.String()
		}
	}

	// Create the UBL document
	out := &Invoice{
		XMLName: xml.Name{Local: rootNameInvoice},
		DocumentHeader: DocumentHeader{
			CACNamespace:    NamespaceCAC,
			CBCNamespace:    NamespaceCBC,
			QDTNamespace:    NamespaceQDT,
			UDTNamespace:    NamespaceUDT,
			UBLNamespace:    NamespaceUBLInvoice,
			CCTSNamespace:   NamespaceCCTS,
			XSINamespace:    NamespaceXSI,
			EXTNamespace:    NamespaceEXT,
			SchemaLocation:  SchemaLocationInvoice,
			CustomizationID: customizationID,
			ProfileID:       profileID,
			ID:              invoiceNumber(inv.Series, inv.Code),
			IssueDate:       formatDate(inv.IssueDate),
		},
		InvoiceTypeCode: &IDType{Value: tc},
		DocumentCurrency: DocumentCurrency{
			AccountingCost:       "", // TODO: ordering cost
			DocumentCurrencyCode: string(inv.Currency),
		},
		DocumentParties: DocumentParties{
			AccountingSupplierParty: SupplierParty{Party: newParty(inv.Supplier, o.context)},
			AccountingCustomerParty: CustomerParty{Party: newParty(inv.Customer, o.context)},
		},
	}

	// PEPPOL-EN16931-R005 / BR-53: only map BT-6 when a matching exchange rate
	// is available. BT-111 is only added in that case and BR-53 requires
	// BT-111 whenever BT-6 is present, so the two must be gated identically.
	if taxCurrency := inv.RegimeDef().Currency; taxCurrency != inv.Currency &&
		cur.MatchExchangeRate(inv.ExchangeRates, inv.Currency, taxCurrency) != nil {
		out.TaxCurrencyCode = string(taxCurrency)
	}

	docType := inv.Type
	if o.context.Is(ContextZATCA) {
		out.SchemaLocation = ""
		// BR-KSA-03
		out.UUID = string(inv.UUID)
		// BR-KSA-70
		out.IssueTime = inv.IssueTime.String()
		// BR-KSA-70
		out.TaxCurrencyCode = string(inv.RegimeDef().Currency)
		// BR-KSA-06
		invType := inv.Tax.GetExt(zatca.ExtKeyInvoiceType).String()
		out.InvoiceTypeCode.Name = &invType
		// ZATCA treats all documents as invoices
		docType = bill.InvoiceTypeStandard
	}

	if docType.In(bill.InvoiceTypeCreditNote) {
		out.XMLName = xml.Name{Local: rootNameCreditNote}
		out.UBLNamespace = NamespaceUBLCreditNote
		out.SchemaLocation = SchemaLocationCrediteNote
		out.InvoiceTypeCode = nil
		out.CreditNoteTypeCode = &IDType{Value: tc}
	}

	// BT-7: VAT point date
	if inv.ValueDate != nil {
		out.TaxPointDate = formatDate(*inv.ValueDate)
	}

	if len(inv.Notes) > 0 {
		var noteTexts []string
		for _, note := range inv.Notes {
			if text := formatNote(note); text != "" {
				noteTexts = append(noteTexts, text)
			}
		}

		if len(noteTexts) > 0 {
			if o.context.Is(ContextPeppol) {
				// Peppol only allows one note, so concatenate all notes
				out.Note = []string{strings.Join(noteTexts, "\n\n")}
			} else {
				out.Note = noteTexts
			}
		}
	}

	out.addPreceding(inv.Preceding)
	out.addOrdering(inv.Ordering, o.context)
	out.addTaxPoint(inv.Tax)
	out.addCharges(inv)
	out.addTotals(inv, o.context)
	out.addLines(inv, o.context)
	out.AddAttachments(inv.Attachments)

	if err = out.addPayment(inv, o.context); err != nil {
		return nil, err
	}
	if d := newDelivery(inv.Delivery, o.context); d != nil {
		out.Delivery = []*Delivery{d}
	}

	return out, nil
}

// taxPointCodeMap maps GOBL tax point keys to UNTDID 2005 codes for UBL.
var taxPointCodeMap = map[cbc.Key]string{
	tax.PointIssue:    "3",
	tax.PointDelivery: "35",
	tax.PointPayment:  "432",
}

// taxPointKeyMap is the reverse mapping from UNTDID 2005 codes to GOBL tax point keys.
var taxPointKeyMap = map[string]cbc.Key{
	"3":   tax.PointIssue,
	"35":  tax.PointDelivery,
	"432": tax.PointPayment,
}

// addTaxPoint maps the GOBL tax point key (BT-8) to the UBL InvoicePeriod DescriptionCode.
func (ui *Invoice) addTaxPoint(t *bill.Tax) {
	if t == nil || t.Point == cbc.KeyEmpty {
		return
	}
	code, ok := taxPointCodeMap[t.Point]
	if !ok {
		return
	}
	if len(ui.InvoicePeriod) == 0 {
		ui.InvoicePeriod = []Period{{}}
	}
	ui.InvoicePeriod[0].DescriptionCode = code
}

func invoiceNumber(series cbc.Code, code cbc.Code) string {
	if series == "" {
		return code.String()
	}
	return fmt.Sprintf("%s-%s", series, code)
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

// Some countries don't differentiate between Invoice and notes
// treating all the same. This helper returns the invoice type
// based on XML name instead of gobl's invoice type key
func (ui *Invoice) getInvoiceTypeBasedOnXMLName() cbc.Key {
	switch ui.XMLName.Local {
	case rootNameCreditNote:
		return bill.InvoiceTypeCreditNote
	default:
		return bill.InvoiceTypeStandard
	}
}
