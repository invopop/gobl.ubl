package gtou

import (
	"fmt"
	"strconv"

	"github.com/invopop/gobl/org"
)

func (c *Conversor) newParty(party *org.Party) Party {
	if party == nil {
		return Party{}
	}
	p := Party{
		PostalAddress: newAddress(party.Addresses),
		PartyLegalEntity: &PartyLegalEntity{
			RegistrationName: &party.Name,
		},
	}

	contact := &Contact{}

	// Although taxID is mandatory, when there is a Tax Representative and the seller comes from
	// Ordering.Seller, the pointer could be nil
	if party.TaxID != nil && party.TaxID.Code != "" {
		taxID := party.TaxID.Code.String()
		p.PartyTaxScheme = []PartyTaxScheme{
			{
				CompanyID: &taxID,
			},
		}
	}

	if len(party.Emails) > 0 {
		contact.ElectronicMail = &party.Emails[0].Address
	}

	if len(party.Telephones) > 0 {
		contact.Telephone = &party.Telephones[0].Number
	}

	if len(party.People) > 0 {
		n := contactName(party.People[0].Name)
		contact.Name = &n
	}

	if contact.Name != nil || contact.Telephone != nil || contact.ElectronicMail != nil {
		p.Contact = contact
	}

	if party.Alias != "" {
		p.PartyName = &PartyName{
			Name: party.Alias,
		}
	}
	return p
}

func newAddress(addresses []*org.Address) *PostalAddress {
	if len(addresses) == 0 {
		return nil
	}
	// Only return the first address
	address := addresses[0]

	postalTradeAddress := &PostalAddress{}

	if address.Street != "" {
		postalTradeAddress.StreetName = &address.Street
	}

	if address.StreetExtra != "" {
		postalTradeAddress.AdditionalStreetName = &address.StreetExtra
	}

	if address.Locality != "" {
		postalTradeAddress.CityName = &address.Locality
	}

	if address.Region != "" {
		postalTradeAddress.CountrySubentity = &address.Region
	}

	if address.Code != "" {
		postalTradeAddress.PostalZone = &address.Code
	}

	if address.Country != "" {
		postalTradeAddress.Country = &Country{IdentificationCode: string(address.Country)}
	}

	if address.Coordinates != nil {
		latitude := strconv.FormatFloat(*address.Coordinates.Latitude, 'f', -1, 64)
		longitude := strconv.FormatFloat(*address.Coordinates.Longitude, 'f', -1, 64)
		postalTradeAddress.LocationCoordinate = &LocationCoordinate{
			LatitudeDegreesMeasure:  &latitude,
			LongitudeDegreesMeasure: &longitude,
		}
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
