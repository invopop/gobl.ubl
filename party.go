package ubl

import (
	"fmt"
	"strconv"

	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

// SchemeIDEmail is the EAS codelist value for email
const SchemeIDEmail = "EM"

// SupplierParty represents the supplier party in a transaction
type SupplierParty struct {
	Party *Party `xml:"cac:Party"`
}

// CustomerParty represents the customer party in a transaction
type CustomerParty struct {
	Party *Party `xml:"cac:Party"`
}

// Party represents a party involved in a transaction
type Party struct {
	EndpointID          *EndpointID       `xml:"cbc:EndpointID"`
	PartyIdentification *Identification   `xml:"cac:PartyIdentification"`
	PartyName           *PartyName        `xml:"cac:PartyName"`
	PostalAddress       *PostalAddress    `xml:"cac:PostalAddress"`
	PartyTaxScheme      []PartyTaxScheme  `xml:"cac:PartyTaxScheme"`
	PartyLegalEntity    *PartyLegalEntity `xml:"cac:PartyLegalEntity"`
	Contact             *Contact          `xml:"cac:Contact"`
}

// EndpointID represents an endpoint identifier
type EndpointID struct {
	SchemeID string `xml:"schemeID,attr"`
	Value    string `xml:",chardata"`
}

// Identification represents an identification
type Identification struct {
	ID *IDType `xml:"cbc:ID"`
}

// PartyName represents the name of a party
type PartyName struct {
	Name string `xml:"cbc:Name"`
}

// PostalAddress represents a postal address
type PostalAddress struct {
	StreetName           *string             `xml:"cbc:StreetName"`
	AdditionalStreetName *string             `xml:"cbc:AdditionalStreetName"`
	CityName             *string             `xml:"cbc:CityName"`
	PostalZone           *string             `xml:"cbc:PostalZone"`
	CountrySubentity     *string             `xml:"cbc:CountrySubentity"`
	AddressLine          []AddressLine       `xml:"cac:AddressLine"`
	Country              *Country            `xml:"cac:Country"`
	LocationCoordinate   *LocationCoordinate `xml:"cac:LocationCoordinate"`
}

// LocationCoordinate represents a location coordinate
type LocationCoordinate struct {
	LatitudeDegreesMeasure  *string `xml:"cbc:LatitudeDegreesMeasure"`
	LatitudeMinutesMeasure  *string `xml:"cbc:LatitudeMinutesMeasure"`
	LongitudeDegreesMeasure *string `xml:"cbc:LongitudeDegreesMeasure"`
	LongitudeMinutesMeasure *string `xml:"cbc:LongitudeMinutesMeasure"`
}

// AddressLine represents a line in an address
type AddressLine struct {
	Line string `xml:"cbc:Line"`
}

// Country represents a country
type Country struct {
	IdentificationCode string `xml:"cbc:IdentificationCode"`
}

// PartyTaxScheme represents a party's tax scheme
type PartyTaxScheme struct {
	CompanyID *string    `xml:"cbc:CompanyID"`
	TaxScheme *TaxScheme `xml:"cac:TaxScheme"`
}

// TaxScheme represents a tax scheme
type TaxScheme struct {
	ID string `xml:"cbc:ID"`
}

// PartyLegalEntity represents the legal entity of a party
type PartyLegalEntity struct {
	RegistrationName *string `xml:"cbc:RegistrationName"`
	CompanyID        *IDType `xml:"cbc:CompanyID"`
	CompanyLegalForm *string `xml:"cbc:CompanyLegalForm"`
}

// Contact represents contact information
type Contact struct {
	Name           *string `xml:"cbc:Name"`
	Telephone      *string `xml:"cbc:Telephone"`
	ElectronicMail *string `xml:"cbc:ElectronicMail"`
}

// CountryCode tries to determine the most appropriate tax country code
// for the party.
func (p *Party) CountryCode() string {
	if pa := p.PostalAddress; pa != nil {
		if c := pa.Country; c != nil {
			return c.IdentificationCode
		}
	}
	return ""
}

func newParty(party *org.Party) *Party {
	if party == nil {
		return nil
	}
	p := &Party{
		PostalAddress: newAddress(party.Addresses),
		PartyLegalEntity: &PartyLegalEntity{
			RegistrationName: &party.Name,
		},
	}

	contact := &Contact{}

	if tID := party.TaxID; tID != nil && party.TaxID.Code != "" {
		code := party.TaxID.String()
		p.PartyTaxScheme = []PartyTaxScheme{
			{
				CompanyID: &code,
				TaxScheme: &TaxScheme{
					ID: tID.GetScheme().String(),
				},
			},
		}
		// Override the company address's country code
		p.PostalAddress.Country = &Country{
			IdentificationCode: tID.Country.String(),
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

	if len(party.Inboxes) > 0 {
		p.EndpointID = &EndpointID{
			Value:    party.Inboxes[0].Email,
			SchemeID: "EM",
		}
	}

	if party.Alias != "" {
		p.PartyName = &PartyName{
			Name: party.Alias,
		}
	}

	if len(party.Identities) > 0 {
		for _, id := range party.Identities {
			if id.Ext != nil {
				s := id.Ext[iso.ExtKeySchemeID].String()
				p.PartyIdentification = &Identification{
					ID: &IDType{
						SchemeID: &s,
						Value:    id.Code.String(),
					},
				}
			}
		}
	}
	return p
}

func newAddress(addresses []*org.Address) *PostalAddress {
	if len(addresses) == 0 {
		return nil
	}
	// Only return the first a
	a := addresses[0]

	addr := &PostalAddress{}

	if a.Street != "" {
		l := a.LineOne()
		addr.StreetName = &l
	}

	if a.StreetExtra != "" {
		l := a.LineTwo()
		addr.AdditionalStreetName = &l
	}

	if a.Locality != "" {
		addr.CityName = &a.Locality
	}

	if a.Region != "" {
		addr.CountrySubentity = &a.Region
	}

	if a.Code != cbc.CodeEmpty {
		code := a.Code.String()
		addr.PostalZone = &code
	}

	if a.Country != "" {
		addr.Country = &Country{IdentificationCode: string(a.Country)}
	}

	if a.Coordinates != nil {
		lat := strconv.FormatFloat(*a.Coordinates.Latitude, 'f', -1, 64)
		lon := strconv.FormatFloat(*a.Coordinates.Longitude, 'f', -1, 64)
		addr.LocationCoordinate = &LocationCoordinate{
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
