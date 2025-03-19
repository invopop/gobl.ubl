package gtou

import (
	"github.com/invopop/gobl.ubl/document"
	"github.com/invopop/gobl/bill"
)

func (c *Converter) newDelivery(del *bill.DeliveryDetails) error {
	if del == nil {
		return nil
	}
	d := formatDate(*del.Date)
	c.doc.Delivery = []document.Delivery{
		{
			ActualDeliveryDate: &d,
			DeliveryLocation: &document.Location{
				Address: newAddress(del.Receiver.Addresses),
			},
		},
	}
	if len(del.Identities) > 0 {
		c.doc.Delivery[0].DeliveryLocation.ID = &document.IDType{Value: del.Identities[0].Code.String()}
	}
	return nil
}
