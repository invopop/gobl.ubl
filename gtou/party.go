package gtou

import (
	"fmt"

	"github.com/invopop/gobl/org"
)

func (c *Conversor) newParty(party *org.Party) Party {
	if party == nil {
		return Party{}
	}
	taxID := party.TaxID.Code.String()
	p := Party{
		PostalAddress: newAddress(party.Addresses),
		PartyTaxScheme: []PartyTaxScheme{
			{
				CompanyID: &taxID,
			},
		},
		PartyLegalEntity: &PartyLegalEntity{
			RegistrationName: party.Name,
		},
	}

	contact := &Contact{}

	if len(party.Emails) > 0 {
		contact.ElectronicMail = party.Emails[0].Address
	}

	if len(party.Telephones) > 0 {
		contact.Telephone = party.Telephones[0].Number
	}

	if len(party.People) > 0 {
		contact.Name = contactName(party.People[0].Name)
	}

	if contact.Name != "" || contact.Telephone != "" || contact.ElectronicMail != "" {
		p.Contact = contact
	}

	if party.Alias != "" {
		p.PartyName = &PartyName{
			Name: party.Alias,
		}
	}
	return p
}

func (c *Conversor) createPartyName(party *org.Party) {
	c.doc.AccountingSupplierParty.Party.PartyName = &PartyName{
		Name: party.Name,
	}
}

func newAddress(addresses []*org.Address) *PostalAddress {
	if len(addresses) == 0 {
		return nil
	}
	// Only return the first address
	address := addresses[0]

	postalTradeAddress := &PostalAddress{
		StreetName:           address.Street,
		AdditionalStreetName: address.StreetExtra,
		CityName:             address.Locality,
		PostalZone:           address.Code,
		CountrySubentity:     address.Region,
		Country:              &Country{IdentificationCode: string(address.Country)},
	}

	return postalTradeAddress
}

func contactName(personName *org.Name) string {
	given := personName.Given
	surname := personName.Surname

	if given == "" && surname == "" {
		return ""
	}
	if given == "" {
		return surname
	}
	if surname == "" {
		return given
	}

	return fmt.Sprintf("%s %s", given, surname)
}
