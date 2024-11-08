// Package document contains the UBL document model for conversion
package document

import (
	"github.com/nbio/xml"
)

// UBL schema constants
const (
	CBC             = "urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"
	CAC             = "urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
	QDT             = "urn:oasis:names:specification:ubl:schema:xsd:QualifiedDataTypes-2"
	UDT             = "urn:oasis:names:specification:ubl:schema:xsd:UnqualifiedDataTypes-2"
	CCTS            = "urn:un:unece:uncefact:documentation:2"
	UBL             = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
	XSI             = "http://www.w3.org/2001/XMLSchema-instance"
	SchemaLocation  = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2 http://docs.oasis-open.org/ubl/os-UBL-2.1/xsd/maindoc/UBL-Invoice-2.1.xsd"
	CustomizationID = "urn:cen.eu:en16931:2017"
)

// Document represents the root element of a UBL invoice
type Document struct {
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
	Delivery                       []Delivery         `xml:"cac:Delivery,omitempty"`
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

// AllowanceCharge represents an allowance or charge
type AllowanceCharge struct {
	ChargeIndicator           bool           `xml:"cbc:ChargeIndicator"`
	AllowanceChargeReasonCode *string        `xml:"cbc:AllowanceChargeReasonCode"`
	AllowanceChargeReason     *string        `xml:"cbc:AllowanceChargeReason"`
	MultiplierFactorNumeric   *string        `xml:"cbc:MultiplierFactorNumeric"`
	Amount                    Amount         `xml:"cbc:Amount"`
	BaseAmount                *Amount        `xml:"cbc:BaseAmount"`
	TaxCategory               []*TaxCategory `xml:"cac:TaxCategory"`
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
