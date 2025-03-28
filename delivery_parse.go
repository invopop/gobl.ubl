package ubl

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

func goblAddDelivery(in *Invoice, out *bill.Invoice) error {
	d := &bill.DeliveryDetails{}

	// Only one delivery Location and Receiver are supported, so if more than one is passed the former will be overwritten
	if len(in.Delivery) > 0 {
		for _, del := range in.Delivery {
			if del.ActualDeliveryDate != nil {
				deliveryDate, err := parseDate(*del.ActualDeliveryDate)
				if err != nil {
					return err
				}
				d.Date = &deliveryDate
			}
			if del.EstimatedDeliveryPeriod != nil {
				d.Period = goblPeriodDates(del.EstimatedDeliveryPeriod)
			}
			if del.DeliveryLocation != nil && del.DeliveryLocation.ID != nil {
				id := &org.Identity{
					Code: cbc.Code(del.DeliveryLocation.ID.Value),
				}
				if del.DeliveryLocation.ID.SchemeID != nil {
					id.Label = *del.DeliveryLocation.ID.SchemeID
				}
				d.Identities = []*org.Identity{id}
			}
			if del.DeliveryParty != nil {
				d.Receiver = goblParty(del.DeliveryParty)
			}
			if del.DeliveryLocation != nil && del.DeliveryLocation.Address != nil {
				d.Receiver = &org.Party{
					Addresses: []*org.Address{
						parseAddress(del.DeliveryLocation.Address),
					},
				}
			}
		}
	}

	if in.DeliveryTerms != nil {
		d.Identities = []*org.Identity{
			{
				Code: cbc.Code(in.DeliveryTerms.ID),
			},
		}
	}

	if d.Receiver != nil || d.Date != nil || d.Identities != nil {
		out.Delivery = d
	}
	return nil
}
