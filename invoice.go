package ubl

import (
	"encoding/xml"
	"fmt"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	// "github.com/nbio/xml"
)

// Main UBL Invoice Namespace
const (
	NamespaceUBLInvoice    = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
	NamespaceUBLCreditNote = "urn:oasis:names:specification:ubl:schema:xsd:CreditNote-2"
)

// Schema locationa and customization constants
const (
	SchemaLocationInvoice     = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2 http://docs.oasis-open.org/ubl/os-UBL-2.1/xsd/maindoc/UBL-Invoice-2.1.xsd"
	SchemaLocationCrediteNote = "urn:oasis:names:specification:ubl:schema:xsd:CreditNote-2 https://docs.oasis-open.org/ubl/os-UBL-2.1/xsd/maindoc/UBL-CreditNote-2.1.xsd"
)

// Invoice represents the root element of a UBL Invoice **or** Credit Note; the structures
// between the two types are so similar, that it doesn't make much sense to seperate.
type Invoice struct {
	// Attributes
	XMLName        xml.Name
	CACNamespace   string `xml:"xmlns:cac,attr"`
	CBCNamespace   string `xml:"xmlns:cbc,attr"`
	QDTNamespace   string `xml:"xmlns:qdt,attr"`
	UDTNamespace   string `xml:"xmlns:udt,attr"`
	CCTSNamespace  string `xml:"xmlns:ccts,attr"`
	UBLNamespace   string `xml:"xmlns,attr"`
	XSINamespace   string `xml:"xmlns:xsi,attr"`
	SchemaLocation string `xml:"xsi:schemaLocation,attr"`

	UBLExtensions      *Extensions `xml:"ext:UBLExtensions,omitempty"`
	UBLVersionID       string      `xml:"cbc:UBLVersionID,omitempty"`
	CustomizationID    string      `xml:"cbc:CustomizationID,omitempty"`
	ProfileID          string      `xml:"cbc:ProfileID,omitempty"`
	ProfileExecutionID string      `xml:"cbc:ProfileExecutionID,omitempty"`
	ID                 string      `xml:"cbc:ID"`
	CopyIndicator      bool        `xml:"cbc:CopyIndicator,omitempty"`
	UUID               string      `xml:"cbc:UUID,omitempty"`
	IssueDate          string      `xml:"cbc:IssueDate"`
	IssueTime          string      `xml:"cbc:IssueTime,omitempty"`
	DueDate            string      `xml:"cbc:DueDate,omitempty"`

	InvoiceTypeCode    string `xml:"cbc:InvoiceTypeCode,omitempty"`
	CreditNoteTypeCode string `xml:"cbc:CreditNoteTypeCode,omitempty"`

	Note                           []string            `xml:"cbc:Note,omitempty"`
	TaxPointDate                   string              `xml:"cbc:TaxPointDate,omitempty"`
	DocumentCurrencyCode           string              `xml:"cbc:DocumentCurrencyCode,omitempty"`
	TaxCurrencyCode                string              `xml:"cbc:TaxCurrencyCode,omitempty"`
	PricingCurrencyCode            string              `xml:"cbc:PricingCurrencyCode,omitempty"`
	PaymentCurrencyCode            string              `xml:"cbc:PaymentCurrencyCode,omitempty"`
	PaymentAlternativeCurrencyCode string              `xml:"cbc:PaymentAlternativeCurrencyCode,omitempty"`
	AccountingCost                 string              `xml:"cbc:AccountingCost,omitempty"`
	LineCountNumeric               int                 `xml:"cbc:LineCountNumeric,omitempty"`
	BuyerReference                 string              `xml:"cbc:BuyerReference,omitempty"`
	InvoicePeriod                  []Period            `xml:"cac:InvoicePeriod,omitempty"`
	OrderReference                 *OrderReference     `xml:"cac:OrderReference,omitempty"`
	BillingReference               []*BillingReference `xml:"cac:BillingReference,omitempty"`
	DespatchDocumentReference      []Reference         `xml:"cac:DespatchDocumentReference,omitempty"`
	ReceiptDocumentReference       []Reference         `xml:"cac:ReceiptDocumentReference,omitempty"`
	StatementDocumentReference     []Reference         `xml:"cac:StatementDocumentReference,omitempty"`
	OriginatorDocumentReference    []Reference         `xml:"cac:OriginatorDocumentReference,omitempty"`
	ContractDocumentReference      []Reference         `xml:"cac:ContractDocumentReference,omitempty"`
	AdditionalDocumentReference    []Reference         `xml:"cac:AdditionalDocumentReference,omitempty"`
	ProjectReference               []ProjectReference  `xml:"cac:ProjectReference,omitempty"`
	Signature                      []Signature         `xml:"cac:Signature,omitempty"`
	AccountingSupplierParty        SupplierParty       `xml:"cac:AccountingSupplierParty"`
	AccountingCustomerParty        CustomerParty       `xml:"cac:AccountingCustomerParty"`
	PayeeParty                     *Party              `xml:"cac:PayeeParty,omitempty"`
	BuyerCustomerParty             *CustomerParty      `xml:"cac:BuyerCustomerParty,omitempty"`
	SellerSupplierParty            *SupplierParty      `xml:"cac:SellerSupplierParty,omitempty"`
	TaxRepresentativeParty         *Party              `xml:"cac:TaxRepresentativeParty,omitempty"`
	Delivery                       []*Delivery         `xml:"cac:Delivery,omitempty"`
	DeliveryTerms                  *DeliveryTerms      `xml:"cac:DeliveryTerms,omitempty"`
	PaymentMeans                   []PaymentMeans      `xml:"cac:PaymentMeans,omitempty"`
	PaymentTerms                   []PaymentTerms      `xml:"cac:PaymentTerms,omitempty"`
	PrepaidPayment                 []PrepaidPayment    `xml:"cac:PrepaidPayment,omitempty"`
	AllowanceCharge                []AllowanceCharge   `xml:"cac:AllowanceCharge,omitempty"`
	TaxExchangeRate                *ExchangeRate       `xml:"cac:TaxExchangeRate,omitempty"`
	PricingExchangeRate            *ExchangeRate       `xml:"cac:PricingExchangeRate,omitempty"`
	PaymentExchangeRate            *ExchangeRate       `xml:"cac:PaymentExchangeRate,omitempty"`
	PaymentAlternativeExchangeRate *ExchangeRate       `xml:"cac:PaymentAlternativeExchangeRate,omitempty"`
	TaxTotal                       []TaxTotal          `xml:"cac:TaxTotal,omitempty"`
	WithholdingTaxTotal            []TaxTotal          `xml:"cac:WithholdingTaxTotal,omitempty"`
	LegalMonetaryTotal             MonetaryTotal       `xml:"cac:LegalMonetaryTotal"`
	InvoiceLines                   []InvoiceLine       `xml:"cac:InvoiceLine,omitempty"`
	CreditNoteLines                []InvoiceLine       `xml:"cac:CreditNoteLine,omitempty"`
}

func newInvoice(inv *bill.Invoice, o *options) (*Invoice, error) {
	tc, err := getTypeCode(inv)
	if err != nil {
		return nil, err
	}

	// Create the UBL document
	out := &Invoice{
		XMLName:                 xml.Name{Local: "Invoice"},
		CACNamespace:            NamespaceCAC,
		CBCNamespace:            NamespaceCBC,
		QDTNamespace:            NamespaceQDT,
		UDTNamespace:            NamespaceUDT,
		UBLNamespace:            NamespaceUBLInvoice,
		CCTSNamespace:           NamespaceCCTS,
		XSINamespace:            NamespaceXSI,
		SchemaLocation:          SchemaLocationInvoice,
		CustomizationID:         o.context.CustomizationID,
		ProfileID:               o.context.ProfileID,
		ID:                      invoiceNumber(inv.Series, inv.Code),
		IssueDate:               formatDate(inv.IssueDate),
		AccountingCost:          "", // TODO: ordering cost
		InvoiceTypeCode:         tc,
		DocumentCurrencyCode:    string(inv.Currency),
		AccountingSupplierParty: SupplierParty{Party: newParty(inv.Supplier)},
		AccountingCustomerParty: CustomerParty{Party: newParty(inv.Customer)},
	}

	if inv.Type.In(bill.InvoiceTypeCreditNote) {
		out.XMLName = xml.Name{Local: "CreditNote"}
		out.UBLNamespace = NamespaceUBLCreditNote
		out.SchemaLocation = SchemaLocationCrediteNote
		out.InvoiceTypeCode = ""
		out.CreditNoteTypeCode = tc
	}

	if len(inv.Notes) > 0 {
		out.Note = make([]string, len(inv.Notes))
		for i, note := range inv.Notes {
			out.Note[i] = note.Text
		}
	}

	out.addPreceding(inv.Preceding)
	out.addOrdering(inv.Ordering)
	out.addCharges(inv)
	out.addTotals(inv.Totals, string(inv.Currency))
	out.addLines(inv)

	if err = out.addPayment(inv.Payment); err != nil {
		return nil, err
	}
	if d := newDelivery(inv.Delivery); d != nil {
		out.Delivery = []*Delivery{d}
	}

	return out, nil
}

// Bytes returns the raw XML of the UBL Invoice or Credit Note including
// the XML Header.
func (out *Invoice) Bytes() ([]byte, error) {
	bytes, err := xml.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), bytes...), nil
}

func invoiceNumber(series cbc.Code, code cbc.Code) string {
	if series == "" {
		return code.String()
	}
	return fmt.Sprintf("%s-%s", series, code)
}
