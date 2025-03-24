package ubl

import "github.com/invopop/gobl/bill"

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

func newDelivery(del *bill.DeliveryDetails) *Delivery {
	if del == nil {
		return nil
	}
	d := formatDate(*del.Date)
	out := &Delivery{
		ActualDeliveryDate: &d,
		DeliveryLocation: &Location{
			Address: newAddress(del.Receiver.Addresses),
		},
	}
	if len(del.Identities) > 0 {
		out.DeliveryLocation.ID = &IDType{Value: del.Identities[0].Code.String()}
	}
	return out
}
