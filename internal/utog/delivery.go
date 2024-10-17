package ubl

import (
	"github.com/invopop/gobl.ubl/structs"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

func ParseUtoGDelivery(inv *bill.Invoice, doc *structs.Invoice) *bill.Delivery {
	delivery := &bill.Delivery{}

	if len(doc.Delivery) > 0 {
		if doc.Delivery[0].DeliveryLocation.Address != nil {
			delivery.Receiver = &org.Party{
				Addresses: []*org.Address{
					parseAddress(doc.Delivery[0].DeliveryLocation.Address),
				},
			}
		}

		if doc.Delivery[0].ActualDeliveryDate != "" {
			deliveryDate := ParseDate(doc.Delivery[0].ActualDeliveryDate)
			delivery.Date = &deliveryDate
		}
	}

	if delivery.Receiver != nil || delivery.Date != nil {
		return delivery
	}

	if doc.DeliveryTerms != nil {
		delivery.Identities = []*org.Identity{
			{
				Code: cbc.Code(doc.DeliveryTerms.ID),
			},
		}
	}

	if delivery.Receiver != nil || delivery.Date != nil || delivery.Identities != nil {
		return delivery
	}
	return nil
}
