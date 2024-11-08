package document

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
