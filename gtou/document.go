package gtou

import (
	"encoding/xml"
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
	XMLName                        xml.Name            `xml:"Invoice"`
	CACNamespace                   string              `xml:"xmlns:cac,attr"`
	CBCNamespace                   string              `xml:"xmlns:cbc,attr"`
	QDTNamespace                   string              `xml:"xmlns:qdt,attr"`
	UDTNamespace                   string              `xml:"xmlns:udt,attr"`
	CCTSNamespace                  string              `xml:"xmlns:ccts,attr"`
	UBLNamespace                   string              `xml:"xmlns,attr"`
	XSINamespace                   string              `xml:"xmlns:xsi,attr"`
	SchemaLocation                 string              `xml:"xsi:schemaLocation,attr"`
	UBLExtensions                  *Extensions         `xml:"ext:UBLExtensions,omitempty"`
	UBLVersionID                   string              `xml:"cbc:UBLVersionID,omitempty"`
	CustomizationID                string              `xml:"cbc:CustomizationID,omitempty"`
	ProfileID                      string              `xml:"cbc:ProfileID,omitempty"`
	ProfileExecutionID             string              `xml:"cbc:ProfileExecutionID,omitempty"`
	ID                             string              `xml:"cbc:ID"`
	CopyIndicator                  bool                `xml:"cbc:CopyIndicator,omitempty"`
	UUID                           string              `xml:"cbc:UUID,omitempty"`
	IssueDate                      string              `xml:"cbc:IssueDate"`
	IssueTime                      string              `xml:"cbc:IssueTime,omitempty"`
	DueDate                        string              `xml:"cbc:DueDate,omitempty"`
	InvoiceTypeCode                string              `xml:"cbc:InvoiceTypeCode,omitempty"`
	Note                           []string            `xml:"cbc:Note,omitempty"`
	TaxPointDate                   string              `xml:"cbc:TaxPointDate,omitempty"`
	DocumentCurrencyCode           string              `xml:"cbc:DocumentCurrencyCode,omitempty"`
	TaxCurrencyCode                string              `xml:"cbc:TaxCurrencyCode,omitempty"`
	PricingCurrencyCode            string              `xml:"cbc:PricingCurrencyCode,omitempty"`
	PaymentCurrencyCode            string              `xml:"cbc:PaymentCurrencyCode,omitempty"`
	PaymentAlternativeCurrencyCode string              `xml:"cbc:PaymentAlternativeCurrencyCode,omitempty"`
	AccountingCostCode             string              `xml:"cbc:AccountingCostCode,omitempty"`
	AccountingCost                 string              `xml:"cbc:AccountingCost,omitempty"`
	LineCountNumeric               int                 `xml:"cbc:LineCountNumeric,omitempty"`
	BuyerReference                 string              `xml:"cbc:BuyerReference,omitempty"`
	InvoicePeriod                  []Period            `xml:"cac:InvoicePeriod,omitempty"`
	OrderReference                 *OrderReference     `xml:"cac:OrderReference,omitempty"`
	BillingReference               []BillingReference  `xml:"cac:BillingReference,omitempty"`
	DespatchDocumentReference      []DocumentReference `xml:"cac:DespatchDocumentReference,omitempty"`
	ReceiptDocumentReference       []DocumentReference `xml:"cac:ReceiptDocumentReference,omitempty"`
	StatementDocumentReference     []DocumentReference `xml:"cac:StatementDocumentReference,omitempty"`
	OriginatorDocumentReference    []DocumentReference `xml:"cac:OriginatorDocumentReference,omitempty"`
	ContractDocumentReference      []DocumentReference `xml:"cac:ContractDocumentReference,omitempty"`
	AdditionalDocumentReference    []DocumentReference `xml:"cac:AdditionalDocumentReference,omitempty"`
	ProjectReference               []ProjectReference  `xml:"cac:ProjectReference,omitempty"`
	Signature                      []Signature         `xml:"cac:Signature,omitempty"`
	AccountingSupplierParty        SupplierParty       `xml:"cac:AccountingSupplierParty"`
	AccountingCustomerParty        CustomerParty       `xml:"cac:AccountingCustomerParty"`
	PayeeParty                     *Party              `xml:"cac:PayeeParty,omitempty"`
	BuyerCustomerParty             *CustomerParty      `xml:"cac:BuyerCustomerParty,omitempty"`
	SellerSupplierParty            *SupplierParty      `xml:"cac:SellerSupplierParty,omitempty"`
	TaxRepresentativeParty         *Party              `xml:"cac:TaxRepresentativeParty,omitempty"`
	Delivery                       []Delivery          `xml:"cac:Delivery,omitempty"`
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
	InvoiceLine                    []InvoiceLine       `xml:"cac:InvoiceLine"`
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

// Period represents a time period with start and end dates
type Period struct {
	StartDate *string `xml:"cbc:StartDate"`
	EndDate   *string `xml:"cbc:EndDate"`
}

// OrderReference represents a reference to an order
type OrderReference struct {
	ID                string  `xml:"cbc:ID"`
	SalesOrderID      *string `xml:"cbc:SalesOrderID"`
	IssueDate         *string `xml:"cbc:IssueDate"`
	CustomerReference *string `xml:"cbc:CustomerReference"`
}

// BillingReference represents a reference to a billing document
type BillingReference struct {
	InvoiceDocumentReference           *DocumentReference `xml:"cac:InvoiceDocumentReference"`
	SelfBilledInvoiceDocumentReference *DocumentReference `xml:"cac:SelfBilledInvoiceDocumentReference"`
	CreditNoteDocumentReference        *DocumentReference `xml:"cac:CreditNoteDocumentReference"`
	AdditionalDocumentReference        *DocumentReference `xml:"cac:AdditionalDocumentReference"`
}

// DocumentReference represents a reference to a document
type DocumentReference struct {
	ID                  IDType      `xml:"cbc:ID"`
	IssueDate           *string     `xml:"cbc:IssueDate"`
	DocumentTypeCode    *string     `xml:"cbc:DocumentTypeCode"`
	DocumentType        *string     `xml:"cbc:DocumentType"`
	Attachment          *Attachment `xml:"cac:Attachment"`
	DocumentDescription *string     `xml:"cbc:DocumentDescription"`
	ValidityPeriod      *Period     `xml:"cac:ValidityPeriod"`
}

// Attachment represents an attached document
type Attachment struct {
	EmbeddedDocumentBinaryObject BinaryObject `xml:"cbc:EmbeddedDocumentBinaryObject"`
}

// BinaryObject represents binary data with associated metadata
type BinaryObject struct {
	MimeCode         *string `xml:"mimeCode,attr"`
	Filename         *string `xml:"filename,attr"`
	EncodingCode     *string `xml:"encodingCode,attr"`
	CharacterSetCode *string `xml:"characterSetCode,attr"`
	URI              *string `xml:"uri,attr"`
	Value            string  `xml:",chardata"`
}

// ProjectReference represents a reference to a project
type ProjectReference struct {
	ID *string `xml:"cbc:ID"`
}

// SupplierParty represents the supplier party in a transaction
type SupplierParty struct {
	Party Party `xml:"cac:Party"`
}

// CustomerParty represents the customer party in a transaction
type CustomerParty struct {
	Party Party `xml:"cac:Party"`
}

// Party represents a party involved in a transaction
type Party struct {
	EndpointID          *EndpointID       `xml:"cbc:EndpointID"`
	PartyIdentification []Identification  `xml:"cac:PartyIdentification"`
	PartyName           *PartyName        `xml:"cac:PartyName"`
	PostalAddress       *PostalAddress    `xml:"cac:PostalAddress"`
	PartyTaxScheme      []PartyTaxScheme  `xml:"cac:PartyTaxScheme"`
	PartyLegalEntity    *PartyLegalEntity `xml:"cac:PartyLegalEntity"`
	Contact             *Contact          `xml:"cac:Contact"`
}

// EndpointID represents an endpoint identifier
type EndpointID struct {
	SchemeID string `xml:"schemeID,attr"`
	Value    string `xml:",chardata"`
}

// Identification represents an identification
type Identification struct {
	ID IDType `xml:"cbc:ID"`
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

// PartyName represents the name of a party
type PartyName struct {
	Name string `xml:"cbc:Name"`
}

// PostalAddress represents a postal address
type PostalAddress struct {
	StreetName           *string             `xml:"cbc:StreetName"`
	AdditionalStreetName *string             `xml:"cbc:AdditionalStreetName"`
	CityName             *string             `xml:"cbc:CityName"`
	PostalZone           *string             `xml:"cbc:PostalZone"`
	CountrySubentity     *string             `xml:"cbc:CountrySubentity"`
	AddressLine          []AddressLine       `xml:"cac:AddressLine"`
	Country              *Country            `xml:"cac:Country"`
	LocationCoordinate   *LocationCoordinate `xml:"cac:LocationCoordinate"`
}

// LocationCoordinate represents a location coordinate
type LocationCoordinate struct {
	LatitudeDegreesMeasure  *string `xml:"cbc:LatitudeDegreesMeasure"`
	LatitudeMinutesMeasure  *string `xml:"cbc:LatitudeMinutesMeasure"`
	LongitudeDegreesMeasure *string `xml:"cbc:LongitudeDegreesMeasure"`
	LongitudeMinutesMeasure *string `xml:"cbc:LongitudeMinutesMeasure"`
}

// AddressLine represents a line in an address
type AddressLine struct {
	Line string `xml:"cbc:Line"`
}

// Country represents a country
type Country struct {
	IdentificationCode string `xml:"cbc:IdentificationCode"`
}

// PartyTaxScheme represents a party's tax scheme
type PartyTaxScheme struct {
	CompanyID *string    `xml:"cbc:CompanyID"`
	TaxScheme *TaxScheme `xml:"cac:TaxScheme"`
}

// TaxScheme represents a tax scheme
type TaxScheme struct {
	ID *string `xml:"cbc:ID"`
}

// PartyLegalEntity represents the legal entity of a party
type PartyLegalEntity struct {
	RegistrationName *string `xml:"cbc:RegistrationName"`
	CompanyID        *IDType `xml:"cbc:CompanyID"`
	CompanyLegalForm *string `xml:"cbc:CompanyLegalForm"`
}

// Contact represents contact information
type Contact struct {
	Name           *string `xml:"cbc:Name"`
	Telephone      *string `xml:"cbc:Telephone"`
	ElectronicMail *string `xml:"cbc:ElectronicMail"`
}

// Delivery represents delivery information
type Delivery struct {
	ActualDeliveryDate      *string   `xml:"cbc:ActualDeliveryDate"`
	DeliveryLocation        *Location `xml:"cac:DeliveryLocation"`
	EstimatedDeliveryPeriod *Period   `xml:"cac:EstimatedDeliveryPeriod"`
	DeliveryParty           *Party    `xml:"cac:DeliveryParty"`
}

// Location represents a location
type Location struct {
	ID      *IDType        `xml:"cbc:ID"`
	Address *PostalAddress `xml:"cac:Address"`
}

// DeliveryTerms represents the terms of delivery
type DeliveryTerms struct {
	ID string `xml:"cbc:ID"`
}

// PaymentMeans represents the means of payment
type PaymentMeans struct {
	PaymentMeansCode      IDType            `xml:"cbc:PaymentMeansCode"`
	PaymentID             *string           `xml:"cbc:PaymentID"`
	PayeeFinancialAccount *FinancialAccount `xml:"cac:PayeeFinancialAccount"`
	PayerFinancialAccount *FinancialAccount `xml:"cac:PayerFinancialAccount"`
	CardAccount           *CardAccount      `xml:"cac:CardAccount"`
	InstructionID         *string           `xml:"cbc:InstructionID"`
	InstructionNote       []string          `xml:"cbc:InstructionNote"`
	PaymentMandate        *PaymentMandate   `xml:"cac:PaymentMandate"`
}

// PaymentMandate represents a payment mandate
type PaymentMandate struct {
	ID                    IDType            `xml:"cbc:ID"`
	PayerFinancialAccount *FinancialAccount `xml:"cac:PayerFinancialAccount"`
}

// CardAccount represents a card account
type CardAccount struct {
	PrimaryAccountNumberID *string `xml:"cbc:PrimaryAccountNumberID"`
	NetworkID              *string `xml:"cbc:NetworkID"`
	HolderName             *string `xml:"cbc:HolderName"`
}

// FinancialAccount represents a financial account
type FinancialAccount struct {
	ID                         *string `xml:"cbc:ID"`
	Name                       *string `xml:"cbc:Name"`
	FinancialInstitutionBranch *Branch `xml:"cac:FinancialInstitutionBranch"`
	AccountTypeCode            *string `xml:"cbc:AccountTypeCode"`
}

// Branch represents a branch of a financial institution
type Branch struct {
	ID   *string `xml:"cbc:ID"`
	Name *string `xml:"cbc:Name"`
}

// PaymentTerms represents the terms of payment
type PaymentTerms struct {
	Note           []string `xml:"cbc:Note"`
	Amount         *Amount  `xml:"cbc:Amount"`
	PaymentPercent *string  `xml:"cbc:PaymentPercent"`
	PaymentDueDate *string  `xml:"cbc:PaymentDueDate"`
}

// PrepaidPayment represents a prepaid payment
type PrepaidPayment struct {
	ID            string  `xml:"cbc:ID"`
	PaidAmount    *Amount `xml:"cbc:PaidAmount"`
	ReceivedDate  *string `xml:"cbc:ReceivedDate"`
	InstructionID *string `xml:"cbc:InstructionID"`
}

// AllowanceCharge represents an allowance or charge
type AllowanceCharge struct {
	ChargeIndicator           bool           `xml:"cbc:ChargeIndicator"`
	AllowanceChargeReasonCode *string        `xml:"cbc:AllowanceChargeReasonCode"`
	AllowanceChargeReason     *string        `xml:"cbc:AllowanceChargeReason"`
	MultiplierFactorNumeric   *string        `xml:"cbc:MultiplierFactorNumeric"`
	Amount                    Amount         `xml:"cbc:Amount"`
	BaseAmount                *Amount        `xml:"cbc:BaseAmount"`
	TaxCategory               *[]TaxCategory `xml:"cac:TaxCategory"`
}

// ExchangeRate represents an exchange rate
type ExchangeRate struct {
	SourceCurrencyCode *string `xml:"cbc:SourceCurrencyCode"`
	TargetCurrencyCode *string `xml:"cbc:TargetCurrencyCode"`
	CalculationRate    *string `xml:"cbc:CalculationRate"`
	Date               *string `xml:"cbc:Date"`
}

// TaxTotal represents a tax total
type TaxTotal struct {
	TaxAmount   Amount        `xml:"cbc:TaxAmount"`
	TaxSubtotal []TaxSubtotal `xml:"cac:TaxSubtotal"`
}

// TaxSubtotal represents a tax subtotal
type TaxSubtotal struct {
	TaxableAmount Amount      `xml:"cbc:TaxableAmount"`
	TaxAmount     Amount      `xml:"cbc:TaxAmount"`
	TaxCategory   TaxCategory `xml:"cac:TaxCategory"`
}

// TaxCategory represents a tax category
type TaxCategory struct {
	ID                     *string    `xml:"cbc:ID"`
	Percent                *string    `xml:"cbc:Percent"`
	TaxExemptionReasonCode *string    `xml:"cbc:TaxExemptionReasonCode"`
	TaxExemptionReason     *string    `xml:"cbc:TaxExemptionReason"`
	TaxScheme              *TaxScheme `xml:"cac:TaxScheme"`
}

// MonetaryTotal represents the monetary totals of the invoice
type MonetaryTotal struct {
	LineExtensionAmount   Amount  `xml:"cbc:LineExtensionAmount"`
	TaxExclusiveAmount    Amount  `xml:"cbc:TaxExclusiveAmount"`
	TaxInclusiveAmount    Amount  `xml:"cbc:TaxInclusiveAmount"`
	AllowanceTotalAmount  *Amount `xml:"cbc:AllowanceTotalAmount"`
	ChargeTotalAmount     *Amount `xml:"cbc:ChargeTotalAmount"`
	PrepaidAmount         *Amount `xml:"cbc:PrepaidAmount"`
	PayableRoundingAmount *Amount `xml:"cbc:PayableRoundingAmount"`
	PayableAmount         *Amount `xml:"cbc:PayableAmount"`
}

// Amount represents a monetary amount
type Amount struct {
	CurrencyID *string `xml:"currencyID,attr"`
	Value      string  `xml:",chardata"`
}

// InvoiceLine represents a line item in an invoice
type InvoiceLine struct {
	ID                  string              `xml:"cbc:ID"`
	Note                []string            `xml:"cbc:Note"`
	InvoicedQuantity    *Quantity           `xml:"cbc:InvoicedQuantity"`
	LineExtensionAmount Amount              `xml:"cbc:LineExtensionAmount"`
	AccountingCost      *string             `xml:"cbc:AccountingCost"`
	InvoicePeriod       *Period             `xml:"cac:InvoicePeriod"`
	OrderLineReference  *OrderLineReference `xml:"cac:OrderLineReference"`
	AllowanceCharge     []*AllowanceCharge  `xml:"cac:AllowanceCharge"`
	Item                *Item               `xml:"cac:Item"`
	Price               *Price              `xml:"cac:Price"`
}

// Quantity represents a quantity with a unit code
type Quantity struct {
	UnitCode string `xml:"unitCode,attr"`
	Value    string `xml:",chardata"`
}

// OrderLineReference represents a reference to an order line
type OrderLineReference struct {
	LineID string `xml:"cbc:LineID"`
}

// Item represents an item in an invoice line
type Item struct {
	Description                *string                    `xml:"cbc:Description"`
	Name                       string                     `xml:"cbc:Name"`
	BuyersItemIdentification   *ItemIdentification        `xml:"cac:BuyersItemIdentification"`
	SellersItemIdentification  *ItemIdentification        `xml:"cac:SellersItemIdentification"`
	StandardItemIdentification *ItemIdentification        `xml:"cac:StandardItemIdentification"`
	OriginCountry              *Country                   `xml:"cac:OriginCountry"`
	CommodityClassification    *[]CommodityClassification `xml:"cac:CommodityClassification"`
	ClassifiedTaxCategory      *ClassifiedTaxCategory     `xml:"cac:ClassifiedTaxCategory"`
	AdditionalItemProperty     *[]AdditionalItemProperty  `xml:"cac:AdditionalItemProperty"`
}

// ItemIdentification represents an item identification
type ItemIdentification struct {
	ID *IDType `xml:"cbc:ID"`
}

// CommodityClassification represents a commodity classification
type CommodityClassification struct {
	ItemClassificationCode *IDType `xml:"cbc:ItemClassificationCode"`
}

// ClassifiedTaxCategory represents a classified tax category
type ClassifiedTaxCategory struct {
	ID        *string    `xml:"cbc:ID"`
	Percent   *string    `xml:"cbc:Percent"`
	TaxScheme *TaxScheme `xml:"cac:TaxScheme"`
}

// AdditionalItemProperty represents an additional property of an item
type AdditionalItemProperty struct {
	Name  string `xml:"cbc:Name"`
	Value string `xml:"cbc:Value"`
}

// Price represents the price of an item
type Price struct {
	PriceAmount     Amount           `xml:"cbc:PriceAmount"`
	BaseAmount      *Amount          `xml:"cbc:BaseAmount"`
	AllowanceCharge *AllowanceCharge `xml:"cac:AllowanceCharge"`
}

// Signature represents a digital signature
type Signature struct {
	ID                         string             `xml:"cbc:ID"`
	Note                       []string           `xml:"cbc:Note,omitempty"`
	ValidationDate             *string            `xml:"cbc:ValidationDate,omitempty"`
	ValidationTime             *string            `xml:"cbc:ValidationTime,omitempty"`
	ValidatorID                *string            `xml:"cbc:ValidatorID,omitempty"`
	CanonicalizationMethod     *string            `xml:"cbc:CanonicalizationMethod,omitempty"`
	SignatureMethod            *string            `xml:"cbc:SignatureMethod,omitempty"`
	SignatoryParty             *Party             `xml:"cac:SignatoryParty,omitempty"`
	DigitalSignatureAttachment *Attachment        `xml:"cac:DigitalSignatureAttachment,omitempty"`
	OriginalDocumentReference  *DocumentReference `xml:"cac:OriginalDocumentReference,omitempty"`
}
