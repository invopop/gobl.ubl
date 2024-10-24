package gtou

import "github.com/invopop/gobl/org"

func (c *Conversor) newCustomer(customer *org.Party) error {
	c.doc.AccountingCustomerParty = CustomerParty{
		Party: Party{
			PartyIdentification: []Identification{
				{ID: customer.ID},
			},
			PartyName: &PartyName{Name: customer.Name},
			PostalAddress: &PostalAddress{
				StreetName: customer.Address.Street,
				CityName:   customer.Address.City,
				PostalZone: customer.Address.PostalCode,
				Country:    &Country{IdentificationCode: customer.Address.CountryCode},
			},
			PartyTaxScheme: []PartyTaxScheme{
				{CompanyID: customer.TaxID},
			},
			PartyLegalEntity: &PartyLegalEntity{
				RegistrationName: customer.LegalName,
				CompanyID:        customer.CompanyID,
			},
			Contact: &Contact{
				Name:           customer.ContactName,
				Telephone:      customer.ContactPhone,
				ElectronicMail: customer.ContactEmail,
			},
		},
	}
	return nil
}
