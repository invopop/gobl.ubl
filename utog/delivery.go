package utog

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

func (c *Conversor) getDelivery(doc *Document) error {
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
			deliveryDate, err := ParseDate(doc.Delivery[0].ActualDeliveryDate)
			if err != nil {
				return err
			}
			delivery.Date = &deliveryDate
		}
	}

	if doc.DeliveryTerms != nil {
		delivery.Identities = []*org.Identity{
			{
				Code: cbc.Code(doc.DeliveryTerms.ID),
			},
		}
	}

	if delivery.Receiver != nil || delivery.Date != nil || delivery.Identities != nil {
		c.inv.Delivery = delivery
	}
	return nil
}
