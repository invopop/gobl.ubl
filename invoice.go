package ubl

import (
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/validation"
	// "github.com/nbio/xml"
)

// UBL schema constants
const (
	NamespaceCBC    = "urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"
	NamespaceCAC    = "urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
	NamespaceQDT    = "urn:oasis:names:specification:ubl:schema:xsd:QualifiedDataTypes-2"
	NamespaceUDT    = "urn:oasis:names:specification:ubl:schema:xsd:UnqualifiedDataTypes-2"
	NamespaceCCTS   = "urn:un:unece:uncefact:documentation:2"
	NamespaceUBL    = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
	NamespaceXSI    = "http://www.w3.org/2001/XMLSchema-instance"
	SchemaLocation  = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2 http://docs.oasis-open.org/ubl/os-UBL-2.1/xsd/maindoc/UBL-Invoice-2.1.xsd"
	CustomizationID = "urn:cen.eu:en16931:2017"
)

// Invoice represents the root element of a UBL invoice
type Invoice struct {
	XMLName                        xml.Name           `xml:"Invoice"`
	CACNamespace                   string             `xml:"xmlns:cac,attr"`
	CBCNamespace                   string             `xml:"xmlns:cbc,attr"`
	QDTNamespace                   string             `xml:"xmlns:qdt,attr"`
	UDTNamespace                   string             `xml:"xmlns:udt,attr"`
	CCTSNamespace                  string             `xml:"xmlns:ccts,attr"`
	UBLNamespace                   string             `xml:"xmlns,attr"`
	XSINamespace                   string             `xml:"xmlns:xsi,attr"`
	SchemaLocation                 string             `xml:"xsi:schemaLocation,attr"`
	UBLExtensions                  *Extensions        `xml:"ext:UBLExtensions,omitempty"`
	UBLVersionID                   string             `xml:"cbc:UBLVersionID,omitempty"`
	CustomizationID                string             `xml:"cbc:CustomizationID,omitempty"`
	ProfileID                      string             `xml:"cbc:ProfileID,omitempty"`
	ProfileExecutionID             string             `xml:"cbc:ProfileExecutionID,omitempty"`
	ID                             string             `xml:"cbc:ID"`
	CopyIndicator                  bool               `xml:"cbc:CopyIndicator,omitempty"`
	UUID                           string             `xml:"cbc:UUID,omitempty"`
	IssueDate                      string             `xml:"cbc:IssueDate"`
	IssueTime                      string             `xml:"cbc:IssueTime,omitempty"`
	DueDate                        string             `xml:"cbc:DueDate,omitempty"`
	InvoiceTypeCode                string             `xml:"cbc:InvoiceTypeCode,omitempty"`
	Note                           []string           `xml:"cbc:Note,omitempty"`
	TaxPointDate                   string             `xml:"cbc:TaxPointDate,omitempty"`
	DocumentCurrencyCode           string             `xml:"cbc:DocumentCurrencyCode,omitempty"`
	TaxCurrencyCode                string             `xml:"cbc:TaxCurrencyCode,omitempty"`
	PricingCurrencyCode            string             `xml:"cbc:PricingCurrencyCode,omitempty"`
	PaymentCurrencyCode            string             `xml:"cbc:PaymentCurrencyCode,omitempty"`
	PaymentAlternativeCurrencyCode string             `xml:"cbc:PaymentAlternativeCurrencyCode,omitempty"`
	AccountingCostCode             string             `xml:"cbc:AccountingCostCode,omitempty"`
	AccountingCost                 string             `xml:"cbc:AccountingCost,omitempty"`
	LineCountNumeric               int                `xml:"cbc:LineCountNumeric,omitempty"`
	BuyerReference                 string             `xml:"cbc:BuyerReference,omitempty"`
	InvoicePeriod                  []Period           `xml:"cac:InvoicePeriod,omitempty"`
	OrderReference                 *OrderReference    `xml:"cac:OrderReference,omitempty"`
	BillingReference               []BillingReference `xml:"cac:BillingReference,omitempty"`
	DespatchDocumentReference      []Reference        `xml:"cac:DespatchDocumentReference,omitempty"`
	ReceiptDocumentReference       []Reference        `xml:"cac:ReceiptDocumentReference,omitempty"`
	StatementDocumentReference     []Reference        `xml:"cac:StatementDocumentReference,omitempty"`
	OriginatorDocumentReference    []Reference        `xml:"cac:OriginatorDocumentReference,omitempty"`
	ContractDocumentReference      []Reference        `xml:"cac:ContractDocumentReference,omitempty"`
	AdditionalDocumentReference    []Reference        `xml:"cac:AdditionalDocumentReference,omitempty"`
	ProjectReference               []ProjectReference `xml:"cac:ProjectReference,omitempty"`
	Signature                      []Signature        `xml:"cac:Signature,omitempty"`
	AccountingSupplierParty        SupplierParty      `xml:"cac:AccountingSupplierParty"`
	AccountingCustomerParty        CustomerParty      `xml:"cac:AccountingCustomerParty"`
	PayeeParty                     *Party             `xml:"cac:PayeeParty,omitempty"`
	BuyerCustomerParty             *CustomerParty     `xml:"cac:BuyerCustomerParty,omitempty"`
	SellerSupplierParty            *SupplierParty     `xml:"cac:SellerSupplierParty,omitempty"`
	TaxRepresentativeParty         *Party             `xml:"cac:TaxRepresentativeParty,omitempty"`
	Delivery                       []*Delivery        `xml:"cac:Delivery,omitempty"`
	DeliveryTerms                  *DeliveryTerms     `xml:"cac:DeliveryTerms,omitempty"`
	PaymentMeans                   []PaymentMeans     `xml:"cac:PaymentMeans,omitempty"`
	PaymentTerms                   []PaymentTerms     `xml:"cac:PaymentTerms,omitempty"`
	PrepaidPayment                 []PrepaidPayment   `xml:"cac:PrepaidPayment,omitempty"`
	AllowanceCharge                []AllowanceCharge  `xml:"cac:AllowanceCharge,omitempty"`
	TaxExchangeRate                *ExchangeRate      `xml:"cac:TaxExchangeRate,omitempty"`
	PricingExchangeRate            *ExchangeRate      `xml:"cac:PricingExchangeRate,omitempty"`
	PaymentExchangeRate            *ExchangeRate      `xml:"cac:PaymentExchangeRate,omitempty"`
	PaymentAlternativeExchangeRate *ExchangeRate      `xml:"cac:PaymentAlternativeExchangeRate,omitempty"`
	TaxTotal                       []TaxTotal         `xml:"cac:TaxTotal,omitempty"`
	WithholdingTaxTotal            []TaxTotal         `xml:"cac:WithholdingTaxTotal,omitempty"`
	LegalMonetaryTotal             MonetaryTotal      `xml:"cac:LegalMonetaryTotal"`
	InvoiceLine                    []InvoiceLine      `xml:"cac:InvoiceLine"`
}

// Extensions represents UBL extensions
type Extensions struct {
	Extension []Extension `xml:"ext:Extension"`
}

// Extension represents a single UBL extension
type Extension struct {
	ID               string  `xml:"cbc:ID"`
	ExtensionURI     *string `xml:"cbc:ExtensionURI"`
	ExtensionContent *string `xml:"ext:ExtensionContent"`
}

// IDType represents an ID with optional scheme attributes
type IDType struct {
	ListID        *string `xml:"listID,attr"`
	ListVersionID *string `xml:"listVersionID,attr"`
	SchemeID      *string `xml:"schemeID,attr"`
	SchemeName    *string `xml:"schemeName,attr"`
	Name          *string `xml:"name,attr"`
	Value         string  `xml:",chardata"`
}

// ExchangeRate represents an exchange rate
type ExchangeRate struct {
	SourceCurrencyCode *string `xml:"cbc:SourceCurrencyCode"`
	TargetCurrencyCode *string `xml:"cbc:TargetCurrencyCode"`
	CalculationRate    *string `xml:"cbc:CalculationRate"`
	Date               *string `xml:"cbc:Date"`
}

// Amount represents a monetary amount
type Amount struct {
	CurrencyID *string `xml:"currencyID,attr"`
	Value      string  `xml:",chardata"`
}

// Signature represents a digital signature
type Signature struct {
	ID                         string      `xml:"cbc:ID"`
	Note                       []string    `xml:"cbc:Note,omitempty"`
	ValidationDate             *string     `xml:"cbc:ValidationDate,omitempty"`
	ValidationTime             *string     `xml:"cbc:ValidationTime,omitempty"`
	ValidatorID                *string     `xml:"cbc:ValidatorID,omitempty"`
	CanonicalizationMethod     *string     `xml:"cbc:CanonicalizationMethod,omitempty"`
	SignatureMethod            *string     `xml:"cbc:SignatureMethod,omitempty"`
	SignatoryParty             *Party      `xml:"cac:SignatoryParty,omitempty"`
	DigitalSignatureAttachment *Attachment `xml:"cac:DigitalSignatureAttachment,omitempty"`
	OriginalDocumentReference  *Reference  `xml:"cac:OriginalDocumentReference,omitempty"`
}

// Bytes returns the XML representation of the document in bytes
func (d *Invoice) Bytes() ([]byte, error) {
	bytes, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), bytes...), nil
}

func newInvoice(inv *bill.Invoice) (*Invoice, error) {
	tc, err := getTypeCode(inv)
	if err != nil {
		return nil, err
	}

	// Create the UBL document
	out := &Invoice{
		CACNamespace:            NamespaceCAC,
		CBCNamespace:            NamespaceCBC,
		QDTNamespace:            NamespaceQDT,
		UDTNamespace:            NamespaceUDT,
		UBLNamespace:            NamespaceUBL,
		CCTSNamespace:           NamespaceCCTS,
		XSINamespace:            NamespaceXSI,
		SchemaLocation:          SchemaLocation,
		CustomizationID:         CustomizationID,
		ID:                      invoiceNumber(inv.Series, inv.Code),
		IssueDate:               formatDate(inv.IssueDate),
		InvoiceTypeCode:         tc,
		DocumentCurrencyCode:    string(inv.Currency),
		AccountingSupplierParty: SupplierParty{Party: newParty(inv.Supplier)},
		AccountingCustomerParty: CustomerParty{Party: newParty(inv.Customer)},
	}

	if len(inv.Notes) > 0 {
		out.Note = make([]string, len(inv.Notes))
		for i, note := range inv.Notes {
			out.Note[i] = note.Text
		}
	}

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

func getTypeCode(inv *bill.Invoice) (string, error) {
	if inv.Tax == nil || inv.Tax.Ext == nil || inv.Tax.Ext[untdid.ExtKeyDocumentType].String() == "" {
		return "", validation.Errors{
			"tax": validation.Errors{
				"ext": validation.Errors{
					untdid.ExtKeyDocumentType.String(): errors.New("required"),
				},
			},
		}
	}
	return inv.Tax.Ext.Get(untdid.ExtKeyDocumentType).String(), nil
}

func invoiceNumber(series cbc.Code, code cbc.Code) string {
	if series == "" {
		return code.String()
	}
	return fmt.Sprintf("%s-%s", series, code)
}
