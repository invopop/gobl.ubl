package utog

import (
	"encoding/xml"
)

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

type Extensions struct {
	Extension []Extension `xml:"Extension"`
}

type Extension struct {
	ID               *string `xml:"ID"`
	ExtensionURI     *string `xml:"ExtensionURI"`
	ExtensionContent string  `xml:"ExtensionContent"`
}

type Period struct {
	StartDate *string `xml:"StartDate"`
	EndDate   *string `xml:"EndDate"`
}

type OrderReference struct {
	ID                string  `xml:"ID"`
	SalesOrderID      *string `xml:"SalesOrderID"`
	IssueDate         *string `xml:"IssueDate"`
	CustomerReference *string `xml:"CustomerReference"`
}

type BillingReference struct {
	InvoiceDocumentReference DocumentReference `xml:"InvoiceDocumentReference"`
}

type DocumentReference struct {
	ID           string      `xml:"ID"`
	IssueDate    *string     `xml:"IssueDate"`
	DocumentType string      `xml:"DocumentType"`
	Attachment   *Attachment `xml:"Attachment"`
}

type Attachment struct {
	EmbeddedDocumentBinaryObject BinaryObject `xml:"EmbeddedDocumentBinaryObject"`
}

type BinaryObject struct {
	MimeCode         string `xml:"mimeCode,attr"`
	Filename         string `xml:"filename,attr"`
	EncodingCode     string `xml:"encodingCode,attr"`
	CharacterSetCode string `xml:"characterSetCode,attr"`
	URI              string `xml:"uri,attr"`
	Value            string `xml:",chardata"`
}

type ProjectReference struct {
	ID string `xml:"ID"`
}

type SupplierParty struct {
	Party Party `xml:"Party"`
}

type CustomerParty struct {
	Party Party `xml:"Party"`
}

type Party struct {
	EndpointID          *EndpointID       `xml:"EndpointID"`
	PartyIdentification []Identification  `xml:"PartyIdentification"`
	PartyName           *PartyName        `xml:"PartyName"`
	PostalAddress       *PostalAddress    `xml:"PostalAddress"`
	PartyTaxScheme      []PartyTaxScheme  `xml:"PartyTaxScheme"`
	PartyLegalEntity    *PartyLegalEntity `xml:"PartyLegalEntity"`
	Contact             *Contact          `xml:"Contact"`
}

type EndpointID struct {
	SchemeID string `xml:"schemeID,attr"`
	Value    string `xml:",chardata"`
}

type Identification struct {
	ID IDType `xml:"ID"`
}

type IDType struct {
	SchemeID   *string `xml:"schemeID,attr"`
	SchemeName *string `xml:"schemeName,attr"`
	Value      string  `xml:",chardata"`
}

type PartyName struct {
	Name string `xml:"Name"`
}

type PostalAddress struct {
	StreetName           *string       `xml:"StreetName"`
	AdditionalStreetName *string       `xml:"AdditionalStreetName"`
	CityName             *string       `xml:"CityName"`
	PostalZone           *string       `xml:"PostalZone"`
	CountrySubentity     *string       `xml:"CountrySubentity"`
	AddressLine          []AddressLine `xml:"AddressLine"`
	Country              *Country      `xml:"Country"`
}

type AddressLine struct {
	Line string `xml:"Line"`
}

type Country struct {
	IdentificationCode string `xml:"IdentificationCode"`
}

type PartyTaxScheme struct {
	CompanyID *string    `xml:"CompanyID"`
	TaxScheme *TaxScheme `xml:"TaxScheme"`
}

type TaxScheme struct {
	ID *string `xml:"ID"`
}

type PartyLegalEntity struct {
	RegistrationName *string `xml:"RegistrationName"`
	CompanyID        *string `xml:"CompanyID"`
	CompanyLegalForm *string `xml:"CompanyLegalForm"`
}

type Contact struct {
	Name           *string `xml:"Name"`
	Telephone      *string `xml:"Telephone"`
	ElectronicMail *string `xml:"ElectronicMail"`
}

type Delivery struct {
	ActualDeliveryDate string   `xml:"ActualDeliveryDate"`
	DeliveryLocation   Location `xml:"DeliveryLocation"`
}

type Location struct {
	ID      string         `xml:"ID"`
	Address *PostalAddress `xml:"Address"`
}

type DeliveryTerms struct {
	ID string `xml:"ID"`
}

type PaymentMeans struct {
	PaymentMeansCode      string            `xml:"PaymentMeansCode"`
	PaymentID             *string           `xml:"PaymentID"`
	PayeeFinancialAccount *FinancialAccount `xml:"PayeeFinancialAccount"`
	PayerFinancialAccount *FinancialAccount `xml:"PayerFinancialAccount"`
	InstructionID         *string           `xml:"InstructionID"`
}

type FinancialAccount struct {
	ID                         *string `xml:"ID"`
	Name                       *string `xml:"Name"`
	FinancialInstitutionBranch *Branch `xml:"FinancialInstitutionBranch"`
	AccountTypeCode            *string `xml:"AccountTypeCode"`
}

type Branch struct {
	ID   *string `xml:"ID"`
	Name *string `xml:"Name"`
}

type PaymentTerms struct {
	Note []string `xml:"Note"`
}

type PrepaidPayment struct {
	ID            *string `xml:"ID"`
	PaidAmount    *Amount `xml:"PaidAmount"`
	ReceivedDate  *string `xml:"ReceivedDate"`
	InstructionID *string `xml:"InstructionID"`
}

type AllowanceCharge struct {
	ChargeIndicator           bool         `xml:"ChargeIndicator"`
	AllowanceChargeReasonCode *string      `xml:"AllowanceChargeReasonCode"`
	AllowanceChargeReason     *string      `xml:"AllowanceChargeReason"`
	MultiplierFactorNumeric   *string      `xml:"MultiplierFactorNumeric"`
	Amount                    Amount       `xml:"Amount"`
	BaseAmount                *Amount      `xml:"BaseAmount"`
	TaxCategory               *TaxCategory `xml:"TaxCategory"`
}

type ExchangeRate struct {
	SourceCurrencyCode string `xml:"SourceCurrencyCode"`
	TargetCurrencyCode string `xml:"TargetCurrencyCode"`
	CalculationRate    string `xml:"CalculationRate"`
	Date               string `xml:"Date"`
}

type TaxTotal struct {
	TaxAmount   Amount        `xml:"TaxAmount"`
	TaxSubtotal []TaxSubtotal `xml:"TaxSubtotal"`
}

type TaxSubtotal struct {
	TaxableAmount Amount      `xml:"TaxableAmount"`
	TaxAmount     Amount      `xml:"TaxAmount"`
	TaxCategory   TaxCategory `xml:"TaxCategory"`
}

type TaxCategory struct {
	ID                     string     `xml:"ID"`
	Percent                *string    `xml:"Percent"`
	TaxExemptionReasonCode string     `xml:"TaxExemptionReasonCode"`
	TaxExemptionReason     string     `xml:"TaxExemptionReason"`
	TaxScheme              *TaxScheme `xml:"TaxScheme"`
}

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

type Amount struct {
	CurrencyID string `xml:"currencyID,attr"`
	Value      string `xml:",chardata"`
}

type InvoiceLine struct {
	ID                  *string             `xml:"ID"`
	InvoicedQuantity    *Quantity           `xml:"InvoicedQuantity"`
	LineExtensionAmount Amount              `xml:"LineExtensionAmount"`
	AccountingCost      *string             `xml:"AccountingCost"`
	InvoicePeriod       *Period             `xml:"InvoicePeriod"`
	OrderLineReference  *OrderLineReference `xml:"OrderLineReference"`
	AllowanceCharge     *[]AllowanceCharge  `xml:"AllowanceCharge"`
	Item                *Item               `xml:"Item"`
	Price               *Price              `xml:"Price"`
}

type Quantity struct {
	UnitCode string `xml:"unitCode,attr"`
	Value    string `xml:",chardata"`
}

type OrderLineReference struct {
	LineID string `xml:"LineID"`
}

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

type ItemIdentification struct {
	ID *string `xml:"ID"`
}

type CommodityClassification struct {
	ItemClassificationCode CodeType `xml:"ItemClassificationCode"`
}

type CodeType struct {
	ListID        string `xml:"listID,attr"`
	ListVersionID string `xml:"listVersionID,attr"`
	Name          string `xml:"name,attr"`
	Value         string `xml:",chardata"`
}

type ClassifiedTaxCategory struct {
	ID        string    `xml:"ID"`
	Percent   string    `xml:"Percent"`
	TaxScheme TaxScheme `xml:"TaxScheme"`
}

type AdditionalItemProperty struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
}

type Price struct {
	PriceAmount     Amount          `xml:"PriceAmount"`
	BaseQuantity    Quantity        `xml:"BaseQuantity"`
	AllowanceCharge AllowanceCharge `xml:"AllowanceCharge"`
}

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
