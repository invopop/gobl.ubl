package ubl

import (
	"errors"
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/validation"
)

// UBL schema constants
const (
	NamespaceCBC  = "urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"
	NamespaceCAC  = "urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
	NamespaceQDT  = "urn:oasis:names:specification:ubl:schema:xsd:QualifiedDataTypes-2"
	NamespaceUDT  = "urn:oasis:names:specification:ubl:schema:xsd:UnqualifiedDataTypes-2"
	NamespaceCCTS = "urn:un:unece:uncefact:documentation:2"
	NamespaceXSI  = "http://www.w3.org/2001/XMLSchema-instance"
)

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
	SchemeAgencyID *string `xml:"schemeAgencyID,attr"`
	ListAgencyID   *string `xml:"listAgencyID,attr"`
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
	ID                     *IDType    `xml:"cbc:ID,omitempty"`
	Percent                *string    `xml:"cbc:Percent,omitempty"`
	TaxExemptionReasonCode *string    `xml:"cbc:TaxExemptionReasonCode,omitempty"`
	TaxScheme              *TaxScheme `xml:"cac:TaxScheme,omitempty"`
}

// AdditionalItemProperty represents an additional property of an item
type AdditionalItemProperty struct {
	Name  string `xml:"cbc:Name"`
	Value string `xml:"cbc:Value"`
}

// Price represents the price of an item
type Price struct {
	PriceAmount     Amount           `xml:"cbc:PriceAmount"`
	BaseQuantity    *Quantity        `xml:"cbc:BaseQuantity,omitempty"`
	AllowanceCharge *AllowanceCharge `xml:"cac:AllowanceCharge,omitempty"`
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

// buildTaxCategoryKey constructs a unique key for a tax category from its scheme ID and category ID
func buildTaxCategoryKey(taxSchemeID, categoryID string) string {
	return taxSchemeID + ":" + categoryID
}

// normalizeNumericString cleans up numeric strings to ensure they can be parsed correctly.
// It handles:
// - Leading/trailing whitespace (e.g., " 123.45 " -> "123.45")
// - Numbers starting with decimal point (e.g., ".07" -> "0.07")
func normalizeNumericString(s string) string {
	// Trim whitespace
	s = strings.TrimSpace(s)

	// Add leading zero if string starts with decimal point
	if strings.HasPrefix(s, ".") {
		s = "0" + s
	}

	return s
}
