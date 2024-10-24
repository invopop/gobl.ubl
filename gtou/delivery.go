package gtou

import "github.com/invopop/gobl/bill"

func (c *Conversor) createDelivery(delivery *bill.Delivery) error {
	c.doc.Delivery = []Delivery{
		{
			ActualDeliveryDate: delivery.Date.Format("2006-01-02"),
			DeliveryLocation: &Location{
				ID: &IDType{Value: delivery.LocationID},
				Address: &PostalAddress{
					StreetName: delivery.Address.Street,
					CityName:   delivery.Address.City,
					PostalZone: delivery.Address.PostalCode,
					Country:    &Country{IdentificationCode: delivery.Address.CountryCode},
				},
			},
		},
	}
}
