package document

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
