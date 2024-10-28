package gtou

import "github.com/invopop/gobl/bill"

func (c *Conversor) newDelivery(delivery *bill.Delivery) error {
	if delivery == nil {
		return nil
	}
	d := formatDate(*delivery.Date)
	c.doc.Delivery = []Delivery{
		{
			ActualDeliveryDate: &d,
			DeliveryLocation: &Location{
				Address: newAddress(delivery.Receiver.Addresses),
			},
		},
	}
	if len(delivery.Identities) > 0 {
		c.doc.Delivery[0].DeliveryLocation.ID = &IDType{Value: delivery.Identities[0].Code.String()}
	}
	return nil
}
