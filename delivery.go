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
	if del == nil || (del.Receiver == nil && del.Date == nil) {
		return nil
	}

	out := new(Delivery)

	if del.Date != nil {
		date := formatDate(*del.Date)
		out.ActualDeliveryDate = &date
	}

	if del.Receiver != nil {
		out.DeliveryParty = newDeliveryParty(del.Receiver)
		out.DeliveryLocation =
			&Location{
				Address: newAddress(del.Receiver.Addresses),
			}
	}

	return out
}

func newDeliveryTerms(del *bill.DeliveryDetails) *DeliveryTerms {
	if del == nil {
		return nil
	}

	if len(del.Identities) > 0 {
		return &DeliveryTerms{ID: del.Identities[0].Code.String()}
	}

	return nil
}
