package gtou

import (
	"encoding/xml"
)

// UBL schema constants
const (
	CBC = "urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"
	CAC = "urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
	UBL = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
)

// Document represents the root element of a UBL invoice
type Document struct {
	XMLName                 xml.Name       `xml:"Invoice"`
	CBCNamespace            string         `xml:"xmlns:cbc,attr"`
	CACNamespace            string         `xml:"xmlns:cac,attr"`
	UBLNamespace            string         `xml:"xmlns,attr"`
	CustomizationID         string         `xml:"cbc:CustomizationID"`
	ProfileID               string         `xml:"cbc:ProfileID"`
	ID                      string         `xml:"cbc:ID"`
	IssueDate               string         `xml:"cbc:IssueDate"`
	DueDate                 string         `xml:"cbc:DueDate,omitempty"`
	InvoiceTypeCode         string         `xml:"cbc:InvoiceTypeCode"`
	Note                    string         `xml:"cbc:Note,omitempty"`
	DocumentCurrencyCode    string         `xml:"cbc:DocumentCurrencyCode"`
	TaxCurrencyCode         string         `xml:"cbc:TaxCurrencyCode,omitempty"`
	AccountingSupplierParty *Party         `xml:"cac:AccountingSupplierParty"`
	AccountingCustomerParty *Party         `xml:"cac:AccountingCustomerParty"`
	Delivery                *Delivery      `xml:"cac:Delivery,omitempty"`
	PaymentMeans            *PaymentMeans  `xml:"cac:PaymentMeans,omitempty"`
	TaxTotal                *TaxTotal      `xml:"cac:TaxTotal"`
	LegalMonetaryTotal      *MonetaryTotal `xml:"cac:LegalMonetaryTotal"`
	InvoiceLines            []*InvoiceLine `xml:"cac:InvoiceLine"`
}

// Party represents a party (supplier or customer) in the invoice
type Party struct {
	Party *PartyDetails `xml:"cac:Party"`
}

// PartyDetails contains the details of a party
type PartyDetails struct {
	PartyIdentification *PartyIdentification `xml:"cac:PartyIdentification,omitempty"`
	PartyName           *PartyName           `xml:"cac:PartyName"`
	PostalAddress       *Address             `xml:"cac:PostalAddress"`
	PartyTaxScheme      *PartyTaxScheme      `xml:"cac:PartyTaxScheme,omitempty"`
	LegalEntity         *LegalEntity         `xml:"cac:PartyLegalEntity,omitempty"`
	Contact             *Contact             `xml:"cac:Contact,omitempty"`
}

// PartyIdentification represents the identification of a party
type PartyIdentification struct {
	ID string `xml:"cbc:ID"`
}

// PartyName represents the name of a party
type PartyName struct {
	Name string `xml:"cbc:Name"`
}

// Address represents a postal address
type Address struct {
	StreetName           string   `xml:"cbc:StreetName,omitempty"`
	AdditionalStreetName string   `xml:"cbc:AdditionalStreetName,omitempty"`
	CityName             string   `xml:"cbc:CityName"`
	PostalZone           string   `xml:"cbc:PostalZone"`
	CountrySubentity     string   `xml:"cbc:CountrySubentity,omitempty"`
	Country              *Country `xml:"cac:Country"`
}

// Country represents a country
type Country struct {
	IdentificationCode string `xml:"cbc:IdentificationCode"`
}

// PartyTaxScheme represents the tax scheme of a party
type PartyTaxScheme struct {
	CompanyID string     `xml:"cbc:CompanyID"`
	TaxScheme *TaxScheme `xml:"cac:TaxScheme"`
}

// TaxScheme represents a tax scheme
type TaxScheme struct {
	ID string `xml:"cbc:ID"`
}

// LegalEntity represents the legal entity information of a party
type LegalEntity struct {
	RegistrationName string `xml:"cbc:RegistrationName"`
	CompanyID        string `xml:"cbc:CompanyID,omitempty"`
}

// Contact represents contact information
type Contact struct {
	Name           string `xml:"cbc:Name,omitempty"`
	Telephone      string `xml:"cbc:Telephone,omitempty"`
	ElectronicMail string `xml:"cbc:ElectronicMail,omitempty"`
}

// Delivery represents delivery information
type Delivery struct {
	ActualDeliveryDate string    `xml:"cbc:ActualDeliveryDate,omitempty"`
	DeliveryLocation   *Location `xml:"cac:DeliveryLocation,omitempty"`
}

// Location represents a delivery location
type Location struct {
	Address *Address `xml:"cac:Address"`
}

// PaymentMeans represents payment means information
type PaymentMeans struct {
	PaymentMeansCode      string            `xml:"cbc:PaymentMeansCode"`
	PaymentID             string            `xml:"cbc:PaymentID,omitempty"`
	PayeeFinancialAccount *FinancialAccount `xml:"cac:PayeeFinancialAccount,omitempty"`
}

// FinancialAccount represents a financial account
type FinancialAccount struct {
	ID string `xml:"cbc:ID"`
}

// TaxTotal represents the total tax amount
type TaxTotal struct {
	TaxAmount   string       `xml:"cbc:TaxAmount"`
	TaxSubtotal *TaxSubtotal `xml:"cac:TaxSubtotal"`
}

// TaxSubtotal represents a tax subtotal
type TaxSubtotal struct {
	TaxableAmount string       `xml:"cbc:TaxableAmount"`
	TaxAmount     string       `xml:"cbc:TaxAmount"`
	TaxCategory   *TaxCategory `xml:"cac:TaxCategory"`
}

// TaxCategory represents a tax category
type TaxCategory struct {
	ID        string     `xml:"cbc:ID"`
	Percent   string     `xml:"cbc:Percent"`
	TaxScheme *TaxScheme `xml:"cac:TaxScheme"`
}

// MonetaryTotal represents the monetary totals of the invoice
type MonetaryTotal struct {
	LineExtensionAmount string `xml:"cbc:LineExtensionAmount"`
	TaxExclusiveAmount  string `xml:"cbc:TaxExclusiveAmount"`
	TaxInclusiveAmount  string `xml:"cbc:TaxInclusiveAmount"`
	PayableAmount       string `xml:"cbc:PayableAmount"`
}

// InvoiceLine represents an invoice line
type InvoiceLine struct {
	ID                  string `xml:"cbc:ID"`
	InvoicedQuantity    string `xml:"cbc:InvoicedQuantity"`
	LineExtensionAmount string `xml:"cbc:LineExtensionAmount"`
	Item                *Item  `xml:"cac:Item"`
	Price               *Price `xml:"cac:Price"`
}

// Item represents an item in an invoice line
type Item struct {
	Description           string       `xml:"cbc:Description"`
	Name                  string       `xml:"cbc:Name"`
	ClassifiedTaxCategory *TaxCategory `xml:"cac:ClassifiedTaxCategory,omitempty"`
}

// Price represents the price of an item
type Price struct {
	PriceAmount string `xml:"cbc:PriceAmount"`
}
