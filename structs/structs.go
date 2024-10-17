package structs

import (
	"encoding/xml"
)

type Invoice struct {
	XMLName                        xml.Name            `xml:"Invoice"`
	Extensions                     *Extensions         `xml:"Extensions,omitempty"`
	VersionID                      string              `xml:"VersionID"`
	CustomizationID                string              `xml:"CustomizationID"`
	ProfileID                      string              `xml:"ProfileID"`
	ProfileExecutionID             string              `xml:"ProfileExecutionID"`
	ID                             string              `xml:"ID"`
	CopyIndicator                  bool                `xml:"CopyIndicator"`
	UUID                           string              `xml:"UUID"`
	IssueDate                      string              `xml:"IssueDate"`
	IssueTime                      string              `xml:"IssueTime"`
	DueDate                        string              `xml:"DueDate"`
	InvoiceTypeCode                string              `xml:"InvoiceTypeCode"`
	Note                           []string            `xml:"Note"`
	TaxPointDate                   string              `xml:"TaxPointDate"`
	DocumentCurrencyCode           string              `xml:"DocumentCurrencyCode"`
	TaxCurrencyCode                string              `xml:"TaxCurrencyCode"`
	PricingCurrencyCode            string              `xml:"PricingCurrencyCode"`
	PaymentCurrencyCode            string              `xml:"PaymentCurrencyCode"`
	PaymentAlternativeCurrencyCode string              `xml:"PaymentAlternativeCurrencyCode"`
	AccountingCost                 string              `xml:"AccountingCost"`
	LineCountNumeric               int                 `xml:"LineCountNumeric"`
	BuyerReference                 string              `xml:"BuyerReference"`
	InvoicePeriod                  *Period             `xml:"InvoicePeriod"`
	OrderReference                 *OrderReference     `xml:"OrderReference"`
	BillingReference               *BillingReference   `xml:"BillingReference"`
	DespatchDocumentReference      *DocumentReference  `xml:"DespatchDocumentReference"`
	ReceiptDocumentReference       *DocumentReference  `xml:"ReceiptDocumentReference"`
	OriginatorDocumentReference    *DocumentReference  `xml:"OriginatorDocumentReference"`
	ContractDocumentReference      *DocumentReference  `xml:"ContractDocumentReference"`
	AdditionalDocumentReference    []DocumentReference `xml:"AdditionalDocumentReference"`
	ProjectReference               *ProjectReference   `xml:"ProjectReference"`
	AccountingSupplierParty        SupplierParty       `xml:"AccountingSupplierParty"`
	AccountingCustomerParty        CustomerParty       `xml:"AccountingCustomerParty"`
	PayeeParty                     *Party              `xml:"PayeeParty"`
	BuyerCustomerParty             *CustomerParty      `xml:"BuyerCustomerParty"`
	SellerSupplierParty            *SupplierParty      `xml:"SellerSupplierParty"`
	TaxRepresentativeParty         *Party              `xml:"TaxRepresentativeParty"`
	Delivery                       []Delivery          `xml:"Delivery"`
	DeliveryTerms                  *DeliveryTerms      `xml:"DeliveryTerms"`
	PaymentMeans                   []PaymentMeans      `xml:"PaymentMeans"`
	PaymentTerms                   []PaymentTerms      `xml:"PaymentTerms"`
	PrepaidPayment                 *PrepaidPayment     `xml:"PrepaidPayment"`
	AllowanceCharge                []AllowanceCharge   `xml:"AllowanceCharge"`
	TaxExchangeRate                *ExchangeRate       `xml:"TaxExchangeRate"`
	PricingExchangeRate            *ExchangeRate       `xml:"PricingExchangeRate"`
	PaymentExchangeRate            *ExchangeRate       `xml:"PaymentExchangeRate"`
	PaymentAlternativeExchangeRate *ExchangeRate       `xml:"PaymentAlternativeExchangeRate"`
	TaxTotal                       []TaxTotal          `xml:"TaxTotal"`
	WithholdingTaxTotal            []TaxTotal          `xml:"WithholdingTaxTotal"`
	LegalMonetaryTotal             MonetaryTotal       `xml:"LegalMonetaryTotal"`
	InvoiceLine                    []InvoiceLine       `xml:"InvoiceLine"`
}

type Extensions struct {
	Extension []Extension `xml:"Extension"`
}

type Extension struct {
	ExtensionURI     string `xml:"ExtensionURI"`
	ExtensionContent string `xml:"ExtensionContent"`
}

type Period struct {
	StartDate string `xml:"StartDate"`
	EndDate   string `xml:"EndDate"`
}

type OrderReference struct {
	ID           string `xml:"ID"`
	SalesOrderID string `xml:"SalesOrderID"`
	IssueDate    string `xml:"IssueDate"`
}

type BillingReference struct {
	InvoiceDocumentReference DocumentReference `xml:"InvoiceDocumentReference"`
}

type DocumentReference struct {
	ID           string      `xml:"ID"`
	IssueDate    string      `xml:"IssueDate"`
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
	PartyIdentification *[]Identification `xml:"PartyIdentification"`
	PartyName           *PartyName        `xml:"PartyName"`
	PostalAddress       *PostalAddress    `xml:"PostalAddress"`
	PartyTaxScheme      *[]PartyTaxScheme `xml:"PartyTaxScheme"`
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
	SchemeID string `xml:"schemeID,attr"`
	Value    string `xml:",chardata"`
}

type PartyName struct {
	Name string `xml:"Name"`
}

type PostalAddress struct {
	StreetName           *string        `xml:"StreetName"`
	AdditionalStreetName *string        `xml:"AdditionalStreetName"`
	CityName             *string        `xml:"CityName"`
	PostalZone           *string        `xml:"PostalZone"`
	CountrySubentity     *string        `xml:"CountrySubentity"`
	AddressLine          *[]AddressLine `xml:"AddressLine"`
	Country              *Country       `xml:"Country"`
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
	PaymentMeansCode      string           `xml:"PaymentMeansCode"`
	PaymentID             string           `xml:"PaymentID"`
	PayeeFinancialAccount FinancialAccount `xml:"PayeeFinancialAccount"`
}

type FinancialAccount struct {
	ID                         string `xml:"ID"`
	Name                       string `xml:"Name"`
	FinancialInstitutionBranch Branch `xml:"FinancialInstitutionBranch"`
}

type Branch struct {
	ID   string `xml:"ID"`
	Name string `xml:"Name"`
}

type PaymentTerms struct {
	Note string `xml:"Note"`
}

type PrepaidPayment struct {
	PaidAmount   Amount `xml:"PaidAmount"`
	ReceivedDate string `xml:"ReceivedDate"`
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
