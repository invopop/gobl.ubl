package ubl

import (
	"fmt"
	"strings"
	"strconv"

	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

// SchemeIDEmail is the EAS codelist value for email
const SchemeIDEmail = "EM"

// TaxSchemeVAT is the tax scheme code for VAT
const TaxSchemeVAT = "VAT"

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
	PartyIdentification []Identification  `xml:"cac:PartyIdentification"`
	PartyName           *PartyName        `xml:"cac:PartyName"`
	PostalAddress       *PostalAddress    `xml:"cac:PostalAddress"`
	PartyTaxScheme      []PartyTaxScheme  `xml:"cac:PartyTaxScheme"`
	PartyLegalEntity    *PartyLegalEntity `xml:"cac:PartyLegalEntity"`
	Contact             *Contact          `xml:"cac:Contact"`
}

// EndpointID represents an endpoint identifier
type EndpointID struct {
	SchemeAgencyID *string `xml:"schemeAgencyID,attr"`
	SchemeID       string  `xml:"schemeID,attr"`
	Value          string  `xml:",chardata"`
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
	AddressFormatCode     *IDType             `xml:"cbc:AddressFormatCode"`
	StreetName           *string             `xml:"cbc:StreetName"`
	BuildingNumber       *string             `xml:"cbc:BuildingNumber"`
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
	CompanyID *IDType    `xml:"cbc:CompanyID"`
	TaxScheme *TaxScheme `xml:"cac:TaxScheme"`
}

// TaxScheme represents a tax scheme
type TaxScheme struct {
	ID          IDType  `xml:"cbc:ID"`
	Name        *string `xml:"cbc:Name"`
	TaxTypeCode string  `xml:"cbc:TaxTypeCode,omitempty"`
}

// PartyLegalEntity represents the legal entity of a party
type PartyLegalEntity struct {
	RegistrationName *string `xml:"cbc:RegistrationName"`
	CompanyID        *IDType `xml:"cbc:CompanyID"`
	CompanyLegalForm *string `xml:"cbc:CompanyLegalForm"`
}

// Contact represents contact information
type Contact struct {
	ID             *string `xml:"cbc:ID"`
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

func newParty(party *org.Party) *Party { //nolint:gocyclo
	if party == nil {
		return nil
	}
	p := &Party{
		PostalAddress: newAddress(party.Addresses),
	}

	// Only add PartyName if name is not empty
	if party.Name != "" {
		p.PartyName = &PartyName{
			Name: party.Name,
		}
		// Only add PartyLegalEntity if name is not empty
		p.PartyLegalEntity = &PartyLegalEntity{
			RegistrationName: &party.Name,
		}
	}

	contact := &Contact{}

	if tID := party.TaxID; tID != nil && party.TaxID.Code != "" {
		code := party.TaxID.String()
		id := tID.GetScheme()
		if id == cbc.CodeEmpty {
			// Peppol default
			id = TaxSchemeVAT
		}

		companyID := &IDType{
			Value: code,
		}
		if string(tID.Country) == "DK" {
			// OIOUBL expects DK VAT numbers with ISO 6523 ICD scheme 0198.
			s := "0198"
			companyID.SchemeID = &s
		}

		taxScheme := PartyTaxScheme{
			CompanyID: companyID,
			TaxScheme: &TaxScheme{
				ID: IDType{Value: id.String()},
			},
		}

		p.PartyTaxScheme = []PartyTaxScheme{taxScheme}
		// Override the company address's country code
		if p.PostalAddress == nil {
			p.PostalAddress = new(PostalAddress)
		}
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
		if n != "" {
			contact.Name = &n
		}
	}

	if contact.Name != nil || contact.Telephone != nil || contact.ElectronicMail != nil {
		p.Contact = contact
	}

	if len(party.Inboxes) > 0 {
		ib := party.Inboxes[0]
		if ib.Email != "" {
			p.EndpointID = &EndpointID{
				SchemeID: SchemeIDEmail,
				Value:    ib.Email,
			}
		} else if ib.Scheme != "" {
			p.EndpointID = &EndpointID{
				SchemeID: normalizeEndpointScheme(ib.Scheme.String()),
				Value:    ib.Code.String(),
			}
		}
	}

	if party.Alias != "" {
		p.PartyName = &PartyName{
			Name: party.Alias,
		}
	}

	if len(party.Identities) > 0 {
		// First pass: Handle legal scope identities
		// First legal identity goes to PartyLegalEntity.CompanyID
		firstLegalIdx := -1
		for i, id := range party.Identities {
			if id.Scope == org.IdentityScopeLegal {
				// Ensure PartyLegalEntity exists before setting CompanyID
				if p.PartyLegalEntity == nil {
					p.PartyLegalEntity = &PartyLegalEntity{}
				}
				code := id.Code.String()
				p.PartyLegalEntity.CompanyID = &IDType{
					Value: code,
				}
				if id.Ext != nil {
					if s := id.Ext[iso.ExtKeySchemeID].String(); s != "" {
						p.PartyLegalEntity.CompanyID.SchemeID = &s
					}
				}
				firstLegalIdx = i
				break
			}
		}

		// Second pass: Handle tax scope identities -> PartyTaxScheme
		for _, id := range party.Identities {
			if id.Scope == org.IdentityScopeTax {
				code := id.Code.String()
				companyID := &IDType{Value: code}
				if id.Ext != nil {
					if s := id.Ext[iso.ExtKeySchemeID].String(); s != "" {
						companyID.SchemeID = &s
					}
				}
				taxScheme := PartyTaxScheme{
					CompanyID: companyID,
					TaxScheme: &TaxScheme{
						ID: IDType{Value: id.Type.String()},
					},
				}
				p.PartyTaxScheme = append(p.PartyTaxScheme, taxScheme)
			}
		}

		// Third pass: Handle remaining identities -> PartyIdentification array
		// This includes non-scoped identities and additional legal identities after the first
		for i, id := range party.Identities {
			// Skip the first legal identity (already in CompanyID)
			if id.Scope == org.IdentityScopeLegal && i == firstLegalIdx {
				continue
			}
			// Skip tax scope identities (already in PartyTaxScheme)
			if id.Scope == org.IdentityScopeTax {
				continue
			}
			// Add to PartyIdentification array
			idType := &IDType{
				Value: id.Code.String(),
			}
			if id.Ext != nil {
				if s := id.Ext[iso.ExtKeySchemeID].String(); s != "" {
					idType.SchemeID = &s
				}
			}
			p.PartyIdentification = append(p.PartyIdentification, Identification{
				ID: idType,
			})
		}
	}

	// OIOUBL requires legal company IDs for Danish parties.
	if p.PartyLegalEntity != nil && p.PartyLegalEntity.CompanyID == nil && party.TaxID != nil && string(party.TaxID.Country) == "DK" {
		s := "0184"
		p.PartyLegalEntity.CompanyID = &IDType{
			SchemeID: &s,
			Value:    party.TaxID.Code.String(),
		}
	}
	return p
}

// newDeliveryParty creates a Party structure for delivery parties
// according to UBL rules:
//   - UBL-CR-394: A UBL invoice should not include the DeliveryParty PostalAddress
//     (it's already in DeliveryLocation)
func newDeliveryParty(party *org.Party) *Party {
	if party == nil {
		return nil
	}

	p := &Party{}
	hasContent := false

	// Only add PartyName if name is not empty
	if party.Name != "" {
		p.PartyName = &PartyName{
			Name: party.Name,
		}
		// Only add PartyLegalEntity if name is not empty
		p.PartyLegalEntity = &PartyLegalEntity{
			RegistrationName: &party.Name,
		}
		hasContent = true
	}

	// Note: Intentionally NOT including PostalAddress per UBL-CR-394
	// The address is already in DeliveryLocation

	contact := &Contact{}

	if len(party.Emails) > 0 {
		contact.ElectronicMail = &party.Emails[0].Address
	}

	if len(party.Telephones) > 0 {
		contact.Telephone = &party.Telephones[0].Number
	}

	if len(party.People) > 0 {
		n := contactName(party.People[0].Name)
		if n != "" {
			contact.Name = &n
		}
	}

	if contact.Name != nil || contact.Telephone != nil || contact.ElectronicMail != nil {
		p.Contact = contact
		hasContent = true
	}

	// Return nil if party would be completely empty to avoid empty XML elements
	if !hasContent {
		return nil
	}

	return p
}

// newPayeeParty creates a minimal Party structure for the Payee
// according to UBL rules which state:
// - BR-17: The Payee name shall be provided
// - UBL-SR-20: Payee identifier shall occur maximum once
// - UBL-CR-272: A UBL invoice should not include the PayeeParty PostalAddress
// - UBL-CR-275: A UBL invoice should not include the PayeeParty PartyLegalEntity RegistrationName
func newPayeeParty(party *org.Party) *Party {
	if party == nil {
		return nil
	}
	p := &Party{
		PartyName: &PartyName{
			Name: party.Name,
		},
	}

	// Add only the first identity with a valid scheme as PartyIdentification (UBL-SR-20: maximum once)
	// Prefer identities with Ext[iso.ExtKeySchemeID] or 4-digit labels (ISO 6523 ICD codes)
	if len(party.Identities) > 0 {
		for _, id := range party.Identities {
			var schemeID *string
			// First check if there's an explicit scheme in Ext
			if id.Ext != nil {
				if s := id.Ext[iso.ExtKeySchemeID].String(); s != "" {
					schemeID = &s
				}
			}
			// If no Ext scheme, check if label looks like a valid ICD code (4 digits)
			if schemeID == nil && id.Label != "" && len(id.Label) == 4 {
				// Assume 4-digit labels are ISO 6523 ICD codes
				schemeID = &id.Label
			}
			// Only add the identity if we have a valid scheme
			if schemeID != nil {
				code := id.Code.String()
				p.PartyIdentification = []Identification{
					{ID: &IDType{
						Value:    code,
						SchemeID: schemeID,
					}},
				}
				break
			}
		}
	}

	// Only add PartyLegalEntity if there's a legal identity, but without RegistrationName
	for _, id := range party.Identities {
		if id.Scope == org.IdentityScopeLegal {
			code := id.Code.String()
			p.PartyLegalEntity = &PartyLegalEntity{
				CompanyID: &IDType{
					Value: code,
				},
			}
			if id.Ext != nil {
				if s := id.Ext[iso.ExtKeySchemeID].String(); s != "" {
					p.PartyLegalEntity.CompanyID.SchemeID = &s
				}
			}
			break
		}
	}

	return p
}

func normalizeEndpointScheme(s string) string {
	switch strings.ToUpper(s) {
	case "GLN":
		return "0088"
	default:
		return s
	}
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
