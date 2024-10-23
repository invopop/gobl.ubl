package utog

import (
	"encoding/xml"
)

// Document represents the main structure of an UBL invoice
type Document struct {
	XMLName                        xml.Name            `xml:"Invoice"`
	UBLExtensions                  *Extensions         `xml:"UBLExtensions,omitempty"`
	UBLVersionID                   *string             `xml:"UBLVersionID,omitempty"`
	CustomizationID                *string             `xml:"CustomizationID,omitempty"`
	ProfileID                      *string             `xml:"ProfileID,omitempty"`
	ProfileExecutionID             *string             `xml:"ProfileExecutionID,omitempty"`
	ID                             string              `xml:"ID"`
	CopyIndicator                  *bool               `xml:"CopyIndicator,omitempty"`
	UUID                           *string             `xml:"UUID,omitempty"`
	IssueDate                      *string             `xml:"IssueDate"`
	IssueTime                      *string             `xml:"IssueTime,omitempty"`
	DueDate                        *string             `xml:"DueDate,omitempty"`
	InvoiceTypeCode                *string             `xml:"InvoiceTypeCode,omitempty"`
	Note                           []string            `xml:"Note,omitempty"`
	TaxPointDate                   *string             `xml:"TaxPointDate,omitempty"`
	DocumentCurrencyCode           *string             `xml:"DocumentCurrencyCode,omitempty"`
	TaxCurrencyCode                *string             `xml:"TaxCurrencyCode,omitempty"`
	PricingCurrencyCode            *string             `xml:"PricingCurrencyCode,omitempty"`
	PaymentCurrencyCode            *string             `xml:"PaymentCurrencyCode,omitempty"`
	PaymentAlternativeCurrencyCode *string             `xml:"PaymentAlternativeCurrencyCode,omitempty"`
	AccountingCostCode             *string             `xml:"AccountingCostCode,omitempty"`
	AccountingCost                 *string             `xml:"AccountingCost,omitempty"`
	LineCountNumeric               *int                `xml:"LineCountNumeric,omitempty"`
	BuyerReference                 *string             `xml:"BuyerReference,omitempty"`
	InvoicePeriod                  []Period            `xml:"InvoicePeriod,omitempty"`
	OrderReference                 *OrderReference     `xml:"OrderReference,omitempty"`
	BillingReference               []BillingReference  `xml:"BillingReference,omitempty"`
	DespatchDocumentReference      []DocumentReference `xml:"DespatchDocumentReference,omitempty"`
	ReceiptDocumentReference       []DocumentReference `xml:"ReceiptDocumentReference,omitempty"`
	StatementDocumentReference     []DocumentReference `xml:"StatementDocumentReference,omitempty"`
	OriginatorDocumentReference    []DocumentReference `xml:"OriginatorDocumentReference,omitempty"`
	ContractDocumentReference      []DocumentReference `xml:"ContractDocumentReference,omitempty"`
	AdditionalDocumentReference    []DocumentReference `xml:"AdditionalDocumentReference,omitempty"`
	ProjectReference               []ProjectReference  `xml:"ProjectReference,omitempty"`
	Signature                      []Signature         `xml:"Signature,omitempty"`
	AccountingSupplierParty        SupplierParty       `xml:"AccountingSupplierParty"`
	AccountingCustomerParty        CustomerParty       `xml:"AccountingCustomerParty"`
	PayeeParty                     *Party              `xml:"PayeeParty,omitempty"`
	BuyerCustomerParty             *CustomerParty      `xml:"BuyerCustomerParty,omitempty"`
	SellerSupplierParty            *SupplierParty      `xml:"SellerSupplierParty,omitempty"`
	TaxRepresentativeParty         *Party              `xml:"TaxRepresentativeParty,omitempty"`
	Delivery                       []Delivery          `xml:"Delivery,omitempty"`
	DeliveryTerms                  *DeliveryTerms      `xml:"DeliveryTerms,omitempty"`
	PaymentMeans                   []PaymentMeans      `xml:"PaymentMeans,omitempty"`
	PaymentTerms                   []PaymentTerms      `xml:"PaymentTerms,omitempty"`
	PrepaidPayment                 []PrepaidPayment    `xml:"PrepaidPayment,omitempty"`
	AllowanceCharge                []AllowanceCharge   `xml:"AllowanceCharge,omitempty"`
	TaxExchangeRate                *ExchangeRate       `xml:"TaxExchangeRate,omitempty"`
	PricingExchangeRate            *ExchangeRate       `xml:"PricingExchangeRate,omitempty"`
	PaymentExchangeRate            *ExchangeRate       `xml:"PaymentExchangeRate,omitempty"`
	PaymentAlternativeExchangeRate *ExchangeRate       `xml:"PaymentAlternativeExchangeRate,omitempty"`
	TaxTotal                       []TaxTotal          `xml:"TaxTotal,omitempty"`
	WithholdingTaxTotal            []TaxTotal          `xml:"WithholdingTaxTotal,omitempty"`
	LegalMonetaryTotal             MonetaryTotal       `xml:"LegalMonetaryTotal"`
	InvoiceLine                    []InvoiceLine       `xml:"InvoiceLine"`
}

// Extensions represents UBL extensions
type Extensions struct {
	Extension []Extension `xml:"Extension"`
}

// Extension represents a single UBL extension
type Extension struct {
	ID               *string `xml:"ID"`
	ExtensionURI     *string `xml:"ExtensionURI"`
	ExtensionContent string  `xml:"ExtensionContent"`
}

// Period represents a time period with start and end dates
type Period struct {
	StartDate *string `xml:"StartDate"`
	EndDate   *string `xml:"EndDate"`
}

// OrderReference represents a reference to an order
type OrderReference struct {
	ID                string  `xml:"ID"`
	SalesOrderID      *string `xml:"SalesOrderID"`
	IssueDate         *string `xml:"IssueDate"`
	CustomerReference *string `xml:"CustomerReference"`
}

// BillingReference represents a reference to a billing document
type BillingReference struct {
	InvoiceDocumentReference           *DocumentReference `xml:"InvoiceDocumentReference"`
	SelfBilledInvoiceDocumentReference *DocumentReference `xml:"SelfBilledInvoiceDocumentReference"`
	CreditNoteDocumentReference        *DocumentReference `xml:"CreditNoteDocumentReference"`
	AdditionalDocumentReference        *DocumentReference `xml:"AdditionalDocumentReference"`
}

// DocumentReference represents a reference to a document
type DocumentReference struct {
	ID                  IDType      `xml:"ID"`
	IssueDate           *string     `xml:"IssueDate"`
	DocumentTypeCode    *string     `xml:"DocumentTypeCode"`
	DocumentType        *string     `xml:"DocumentType"`
	Attachment          *Attachment `xml:"Attachment"`
	DocumentDescription *string     `xml:"DocumentDescription"`
	ValidityPeriod      *Period     `xml:"ValidityPeriod"`
}

// Attachment represents an attached document
type Attachment struct {
	EmbeddedDocumentBinaryObject BinaryObject `xml:"EmbeddedDocumentBinaryObject"`
}

// BinaryObject represents binary data with associated metadata
type BinaryObject struct {
	MimeCode         string `xml:"mimeCode,attr"`
	Filename         string `xml:"filename,attr"`
	EncodingCode     string `xml:"encodingCode,attr"`
	CharacterSetCode string `xml:"characterSetCode,attr"`
	URI              string `xml:"uri,attr"`
	Value            string `xml:",chardata"`
}

// ProjectReference represents a reference to a project
type ProjectReference struct {
	ID string `xml:"ID"`
}

// SupplierParty represents the supplier party in a transaction
type SupplierParty struct {
	Party Party `xml:"Party"`
}

// CustomerParty represents the customer party in a transaction
type CustomerParty struct {
	Party Party `xml:"Party"`
}

// Party represents a party involved in a transaction
type Party struct {
	EndpointID          *EndpointID       `xml:"EndpointID"`
	PartyIdentification []Identification  `xml:"PartyIdentification"`
	PartyName           *PartyName        `xml:"PartyName"`
	PostalAddress       *PostalAddress    `xml:"PostalAddress"`
	PartyTaxScheme      []PartyTaxScheme  `xml:"PartyTaxScheme"`
	PartyLegalEntity    *PartyLegalEntity `xml:"PartyLegalEntity"`
	Contact             *Contact          `xml:"Contact"`
}

// EndpointID represents an endpoint identifier
type EndpointID struct {
	SchemeID string `xml:"schemeID,attr"`
	Value    string `xml:",chardata"`
}

// Identification represents an identification
type Identification struct {
	ID IDType `xml:"ID"`
}

// IDType represents an ID with optional scheme attributes
type IDType struct {
	SchemeID   *string `xml:"schemeID,attr"`
	SchemeName *string `xml:"schemeName,attr"`
	Value      string  `xml:",chardata"`
}

// PartyName represents the name of a party
type PartyName struct {
	Name string `xml:"Name"`
}

// PostalAddress represents a postal address
type PostalAddress struct {
	StreetName           *string       `xml:"StreetName"`
	AdditionalStreetName *string       `xml:"AdditionalStreetName"`
	CityName             *string       `xml:"CityName"`
	PostalZone           *string       `xml:"PostalZone"`
	CountrySubentity     *string       `xml:"CountrySubentity"`
	AddressLine          []AddressLine `xml:"AddressLine"`
	Country              *Country      `xml:"Country"`
}

// AddressLine represents a line in an address
type AddressLine struct {
	Line string `xml:"Line"`
}

// Country represents a country
type Country struct {
	IdentificationCode string `xml:"IdentificationCode"`
}

// PartyTaxScheme represents a party's tax scheme
type PartyTaxScheme struct {
	CompanyID *string    `xml:"CompanyID"`
	TaxScheme *TaxScheme `xml:"TaxScheme"`
}

// TaxScheme represents a tax scheme
type TaxScheme struct {
	ID *string `xml:"ID"`
}

// PartyLegalEntity represents the legal entity of a party
type PartyLegalEntity struct {
	RegistrationName *string `xml:"RegistrationName"`
	CompanyID        *IDType `xml:"CompanyID"`
	CompanyLegalForm *string `xml:"CompanyLegalForm"`
}

// Contact represents contact information
type Contact struct {
	Name           *string `xml:"Name"`
	Telephone      *string `xml:"Telephone"`
	ElectronicMail *string `xml:"ElectronicMail"`
}

// Delivery represents delivery information
type Delivery struct {
	ActualDeliveryDate      *string   `xml:"ActualDeliveryDate"`
	DeliveryLocation        *Location `xml:"DeliveryLocation"`
	EstimatedDeliveryPeriod *Period   `xml:"EstimatedDeliveryPeriod"`
	DeliveryParty           *Party    `xml:"DeliveryParty"`
}

// Location represents a location
type Location struct {
	ID      *IDType        `xml:"ID"`
	Address *PostalAddress `xml:"Address"`
}

// DeliveryTerms represents the terms of delivery
type DeliveryTerms struct {
	ID string `xml:"ID"`
}

// PaymentMeans represents the means of payment
type PaymentMeans struct {
	PaymentMeansCode      string            `xml:"PaymentMeansCode"`
	PaymentID             *string           `xml:"PaymentID"`
	PayeeFinancialAccount *FinancialAccount `xml:"PayeeFinancialAccount"`
	PayerFinancialAccount *FinancialAccount `xml:"PayerFinancialAccount"`
	InstructionID         *string           `xml:"InstructionID"`
}

// FinancialAccount represents a financial account
type FinancialAccount struct {
	ID                         *string `xml:"ID"`
	Name                       *string `xml:"Name"`
	FinancialInstitutionBranch *Branch `xml:"FinancialInstitutionBranch"`
	AccountTypeCode            *string `xml:"AccountTypeCode"`
}

// Branch represents a branch of a financial institution
type Branch struct {
	ID   *string `xml:"ID"`
	Name *string `xml:"Name"`
}

// PaymentTerms represents the terms of payment
type PaymentTerms struct {
	Note           []string `xml:"Note"`
	Amount         *Amount  `xml:"Amount"`
	PaymentPercent *string  `xml:"PaymentPercent"`
	PaymentDueDate *string  `xml:"PaymentDueDate"`
}

// PrepaidPayment represents a prepaid payment
type PrepaidPayment struct {
	ID            *string `xml:"ID"`
	PaidAmount    *Amount `xml:"PaidAmount"`
	ReceivedDate  *string `xml:"ReceivedDate"`
	InstructionID *string `xml:"InstructionID"`
}

// AllowanceCharge represents an allowance or charge
type AllowanceCharge struct {
	ChargeIndicator           bool         `xml:"ChargeIndicator"`
	AllowanceChargeReasonCode *string      `xml:"AllowanceChargeReasonCode"`
	AllowanceChargeReason     *string      `xml:"AllowanceChargeReason"`
	MultiplierFactorNumeric   *string      `xml:"MultiplierFactorNumeric"`
	Amount                    Amount       `xml:"Amount"`
	BaseAmount                *Amount      `xml:"BaseAmount"`
	TaxCategory               *TaxCategory `xml:"TaxCategory"`
}

// ExchangeRate represents an exchange rate
type ExchangeRate struct {
	SourceCurrencyCode string `xml:"SourceCurrencyCode"`
	TargetCurrencyCode string `xml:"TargetCurrencyCode"`
	CalculationRate    string `xml:"CalculationRate"`
	Date               string `xml:"Date"`
}

// TaxTotal represents a tax total
type TaxTotal struct {
	TaxAmount   Amount        `xml:"TaxAmount"`
	TaxSubtotal []TaxSubtotal `xml:"TaxSubtotal"`
}

// TaxSubtotal represents a tax subtotal
type TaxSubtotal struct {
	TaxableAmount Amount      `xml:"TaxableAmount"`
	TaxAmount     Amount      `xml:"TaxAmount"`
	TaxCategory   TaxCategory `xml:"TaxCategory"`
}

// TaxCategory represents a tax category
type TaxCategory struct {
	ID                     string     `xml:"ID"`
	Percent                *string    `xml:"Percent"`
	TaxExemptionReasonCode string     `xml:"TaxExemptionReasonCode"`
	TaxExemptionReason     string     `xml:"TaxExemptionReason"`
	TaxScheme              *TaxScheme `xml:"TaxScheme"`
}

// MonetaryTotal represents a monetary total
type MonetaryTotal struct {
	LineExtensionAmount   Amount `xml:"LineExtensionAmount"`
	TaxExclusiveAmount    Amount `xml:"TaxExclusiveAmount"`
	TaxInclusiveAmount    Amount `xml:"TaxInclusiveAmount"`
	AllowanceTotalAmount  Amount `xml:"AllowanceTotalAmount"`
	ChargeTotalAmount     Amount `xml:"ChargeTotalAmount"`
	PrepaidAmount         Amount `xml:"PrepaidAmount"`
	PayableRoundingAmount Amount `xml:"PayableRoundingAmount"`
	PayableAmount         Amount `xml:"PayableAmount"`
}

// Amount represents a monetary amount
type Amount struct {
	CurrencyID string `xml:"currencyID,attr"`
	Value      string `xml:",chardata"`
}

// InvoiceLine represents a line item in an invoice
type InvoiceLine struct {
	ID                  *string             `xml:"ID"`
	Note                []string            `xml:"Note"`
	InvoicedQuantity    *Quantity           `xml:"InvoicedQuantity"`
	LineExtensionAmount Amount              `xml:"LineExtensionAmount"`
	AccountingCost      *string             `xml:"AccountingCost"`
	InvoicePeriod       *Period             `xml:"InvoicePeriod"`
	OrderLineReference  *OrderLineReference `xml:"OrderLineReference"`
	AllowanceCharge     *[]AllowanceCharge  `xml:"AllowanceCharge"`
	Item                *Item               `xml:"Item"`
	Price               *Price              `xml:"Price"`
}

// Quantity represents a quantity with a unit code
type Quantity struct {
	UnitCode string `xml:"unitCode,attr"`
	Value    string `xml:",chardata"`
}

// OrderLineReference represents a reference to an order line
type OrderLineReference struct {
	LineID string `xml:"LineID"`
}

// Item represents an item in an invoice line
type Item struct {
	Description                *string                    `xml:"Description"`
	Name                       *string                    `xml:"Name"`
	BuyersItemIdentification   *ItemIdentification        `xml:"BuyersItemIdentification"`
	SellersItemIdentification  *ItemIdentification        `xml:"SellersItemIdentification"`
	StandardItemIdentification *ItemIdentification        `xml:"StandardItemIdentification"`
	OriginCountry              *Country                   `xml:"OriginCountry"`
	CommodityClassification    *[]CommodityClassification `xml:"CommodityClassification"`
	ClassifiedTaxCategory      *ClassifiedTaxCategory     `xml:"ClassifiedTaxCategory"`
	AdditionalItemProperty     *[]AdditionalItemProperty  `xml:"AdditionalItemProperty"`
}

// ItemIdentification represents an item identification
type ItemIdentification struct {
	ID *IDType `xml:"ID"`
}

// CommodityClassification represents a commodity classification
type CommodityClassification struct {
	ItemClassificationCode CodeType `xml:"ItemClassificationCode"`
}

// CodeType represents a code with associated metadata
type CodeType struct {
	ListID        *string `xml:"listID,attr"`
	ListVersionID *string `xml:"listVersionID,attr"`
	Name          *string `xml:"name,attr"`
	Value         string  `xml:",chardata"`
}

// ClassifiedTaxCategory represents a classified tax category
type ClassifiedTaxCategory struct {
	ID        string    `xml:"ID"`
	Percent   string    `xml:"Percent"`
	TaxScheme TaxScheme `xml:"TaxScheme"`
}

// AdditionalItemProperty represents an additional property of an item
type AdditionalItemProperty struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
}

// Price represents the price of an item
type Price struct {
	PriceAmount     Amount          `xml:"PriceAmount"`
	BaseQuantity    Quantity        `xml:"BaseQuantity"`
	AllowanceCharge AllowanceCharge `xml:"AllowanceCharge"`
}

// Signature represents a digital signature
type Signature struct {
	ID                         string             `xml:"cbc:ID"`
	Note                       []string           `xml:"cbc:Note,omitempty"`
	ValidationDate             string             `xml:"cbc:ValidationDate,omitempty"`
	ValidationTime             string             `xml:"cbc:ValidationTime,omitempty"`
	ValidatorID                string             `xml:"cbc:ValidatorID,omitempty"`
	CanonicalizationMethod     string             `xml:"cbc:CanonicalizationMethod,omitempty"`
	SignatureMethod            string             `xml:"cbc:SignatureMethod,omitempty"`
	SignatoryParty             *Party             `xml:"cac:SignatoryParty,omitempty"`
	DigitalSignatureAttachment *Attachment        `xml:"cac:DigitalSignatureAttachment,omitempty"`
	OriginalDocumentReference  *DocumentReference `xml:"cac:OriginalDocumentReference,omitempty"`
}
