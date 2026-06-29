package ubl

import "github.com/invopop/gobl/bill"

// Delivery represents delivery information
type Delivery struct {
	ActualDeliveryDate      *string   `xml:"cbc:ActualDeliveryDate"`
	LatestDeliveryDate      *string   `xml:"cbc:LatestDeliveryDate"`
	DeliveryLocation        *Location `xml:"cac:DeliveryLocation"`
	RequestedDeliveryPeriod *Period   `xml:"cac:RequestedDeliveryPeriod"`
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

func newDelivery(del *bill.DeliveryDetails, ctx Context) *Delivery {
	if del == nil {
		return nil
	}

	out := new(Delivery)

	if del.Date != nil {
		date := formatDate(*del.Date)
		out.ActualDeliveryDate = &date
	}

	if del.Period != nil {
		if ctx.Is(ContextOIOUBL21) {
			// A delivery window maps to RequestedDeliveryPeriod — the only delivery
			// period OIOUBL permits, since it forbids LatestDeliveryDate (F-INV087)
			// and the Promised/Estimated periods (F-INV089/F-INV090).
			out.RequestedDeliveryPeriod = &Period{
				StartDate: formatDate(del.Period.Start),
				EndDate:   formatDate(del.Period.End),
			}
		} else {
			start := formatDate(del.Period.Start)
			out.ActualDeliveryDate = &start
			end := formatDate(del.Period.End)
			out.LatestDeliveryDate = &end
		}
	}

	if del.Receiver != nil {
		out.DeliveryParty = newDeliveryParty(del.Receiver)
		// OIOUBL requires a non-empty CompanyID whenever PartyLegalEntity is
		// present (F-LIB187), but a delivery party only identifies a location and
		// carries no company id. PartyLegalEntity isn't mandatory here, so drop it
		// and keep just the PartyName.
		if ctx.Is(ContextOIOUBL21) && out.DeliveryParty != nil {
			out.DeliveryParty.PartyLegalEntity = nil
		}
		out.DeliveryLocation =
			&Location{
				Address: newAddress(del.Receiver.Addresses, ctx),
			}
		if len(del.Identities) > 0 {
			out.DeliveryLocation.ID = &IDType{Value: del.Identities[0].Code.String()}
		}
	}

	return out
}
