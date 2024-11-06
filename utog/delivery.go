package utog

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

func (c *Converter) getDelivery(doc *Document) error {
	delivery := &bill.Delivery{}

	// Only one delivery Location and Receiver are supported, so if more than one is passed the former will be overwritten
	if len(doc.Delivery) > 0 {
		for _, del := range doc.Delivery {
			if del.ActualDeliveryDate != nil {
				deliveryDate, err := ParseDate(*del.ActualDeliveryDate)
				if err != nil {
					return err
				}
				delivery.Date = &deliveryDate
			}
			if del.EstimatedDeliveryPeriod != nil {
				delivery.Period = c.setPeriodDates(*del.EstimatedDeliveryPeriod)
			}
			if del.DeliveryLocation != nil && del.DeliveryLocation.ID != nil {
				id := &org.Identity{
					Code: cbc.Code(del.DeliveryLocation.ID.Value),
				}
				if del.DeliveryLocation.ID.SchemeID != nil {
					id.Label = *del.DeliveryLocation.ID.SchemeID
				}
				delivery.Identities = []*org.Identity{id}
			}
			if del.DeliveryParty != nil {
				delivery.Receiver = c.getParty(del.DeliveryParty)
			}
			if del.DeliveryLocation != nil && del.DeliveryLocation.Address != nil {
				delivery.Receiver = &org.Party{
					Addresses: []*org.Address{
						parseAddress(del.DeliveryLocation.Address),
					},
				}
			}
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
