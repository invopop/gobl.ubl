package gtou

import (
	"fmt"
	"strconv"

	"github.com/invopop/gobl.ubl/document"
	"github.com/invopop/gobl/org"
)

func (c *Converter) newParty(party *org.Party) document.Party {
	if party == nil {
		return document.Party{}
	}
	p := document.Party{
		PostalAddress: newAddress(party.Addresses),
		PartyLegalEntity: &document.PartyLegalEntity{
			RegistrationName: &party.Name,
		},
	}

	contact := &document.Contact{}

	// Although taxID is mandatory, when there is a Tax Representative and the seller comes from
	// Ordering.Seller, the pointer could be nil
	if party.TaxID != nil && party.TaxID.Code != "" {
		taxID := party.TaxID.Code.String()
		p.PartyTaxScheme = []document.PartyTaxScheme{
			{
				CompanyID: &taxID,
				TaxScheme: &document.TaxScheme{
					ID: "VAT",
				},
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
		p.PartyName = &document.PartyName{
			Name: party.Alias,
		}
	}
	return p
}

func newAddress(addresses []*org.Address) *document.PostalAddress {
	if len(addresses) == 0 {
		return nil
	}
	// Only return the first a
	a := addresses[0]

	addr := &document.PostalAddress{}

	if a.Street != "" {
		addr.StreetName = &a.Street
	}

	if a.Number != "" {
		addr.AddressLine = []document.AddressLine{
			{
				Line: a.Number,
			},
		}
	}

	if a.StreetExtra != "" {
		addr.AdditionalStreetName = &a.StreetExtra
	}

	if a.Locality != "" {
		addr.CityName = &a.Locality
	}

	if a.Region != "" {
		addr.CountrySubentity = &a.Region
	}

	if a.Code != "" {
		addr.PostalZone = &a.Code
	}

	if a.Country != "" {
		addr.Country = &document.Country{IdentificationCode: string(a.Country)}
	}

	if a.Coordinates != nil {
		lat := strconv.FormatFloat(*a.Coordinates.Latitude, 'f', -1, 64)
		lon := strconv.FormatFloat(*a.Coordinates.Longitude, 'f', -1, 64)
		addr.LocationCoordinate = &document.LocationCoordinate{
			LatitudeDegreesMeasure:  &lat,
			LongitudeDegreesMeasure: &lon,
		}
	}

	return addr
}

func contactName(n *org.Name) string {
	given := n.Given
	surname := n.Surname

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
