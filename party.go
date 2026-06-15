package ubl

import (
	"fmt"
	"strconv"
	"strings"

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
	AddressFormatCode    *IDType             `xml:"cbc:AddressFormatCode"`
	Postbox              *string             `xml:"cbc:Postbox,omitempty"`
	StreetName           *string             `xml:"cbc:StreetName"`
	AdditionalStreetName *string             `xml:"cbc:AdditionalStreetName"`
	BuildingNumber       *string             `xml:"cbc:BuildingNumber,omitempty"`
	PlotIdentification   *string             `xml:"cbc:PlotIdentification,omitempty"`
	CitySubdivisionName  *string             `xml:"cbc:CitySubdivisionName,omitempty"`
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

func newParty(party *org.Party, ctx Context) *Party { //nolint:gocyclo
	if party == nil {
		return nil
	}
	p := &Party{
		PostalAddress: newAddress(party.Addresses, ctx),
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
		if ctx.Is(ContextZATCA) {
			code = code[2:]
		}
		id := tID.GetScheme()
		if id == cbc.CodeEmpty {
			// Peppol default
			id = TaxSchemeVAT
		}

		companyID := &IDType{
			Value: code,
		}
		// The DK:SE (0198) scheme on PartyTaxScheme/CompanyID is OIOUBL-specific;
		// emitting it under other contexts (e.g. Peppol) is unintended surface.
		if ctx.Is(ContextOIOUBL21) && string(tID.Country) == "DK" {
			s := icdDKSE
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
		// OIOUBL requires cac:Contact/cbc:ID (F-INV051); source it from the
		// person's identity when present rather than fabricating one.
		if ctx.Is(ContextOIOUBL21) {
			if ids := party.People[0].Identities; len(ids) > 0 && ids[0].Code != "" {
				code := ids[0].Code.String()
				contact.ID = &code
			}
		}
	}

	if contact.Name != nil || contact.Telephone != nil || contact.ElectronicMail != nil || contact.ID != nil {
		p.Contact = contact
	}

	if ep := party.Endpoint(iso6523EndpointScheme); ep != nil {
		if icd, code, ok := splitISO6523Endpoint(ep.URI); ok {
			p.EndpointID = &EndpointID{
				SchemeID: normalizeEndpointScheme(icd),
				Value:    code,
			}
		}
	}
	if p.EndpointID == nil && len(party.Inboxes) > 0 {
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
				if s := id.Ext.Get(iso.ExtKeySchemeID).String(); s != "" {
					p.PartyLegalEntity.CompanyID.SchemeID = &s
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
				if s := id.Ext.Get(iso.ExtKeySchemeID).String(); s != "" {
					companyID.SchemeID = &s
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
			if s := id.Ext.Get(iso.ExtKeySchemeID).String(); s != "" {
				idType.SchemeID = &s
			} else if id.Ext.IsZero() {
				// ZATCA has very specific identities that do not
				// require an ISO extension and are only described with type
				if t := id.Type.String(); t != "" {
					idType.SchemeID = &t
				}
			}
			p.PartyIdentification = append(p.PartyIdentification, Identification{
				ID: idType,
			})
		}
	}

	// Fabricating a DK:CVR (0184) PartyLegalEntity/CompanyID from the tax ID is
	// OIOUBL-specific; under other contexts the legal identity should come from
	// real data, not the tax number.
	if ctx.Is(ContextOIOUBL21) && p.PartyLegalEntity != nil && p.PartyLegalEntity.CompanyID == nil && party.TaxID != nil && string(party.TaxID.Country) == "DK" {
		s := icdDKCVR
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
			if s := id.Ext.Get(iso.ExtKeySchemeID).String(); s != "" {
				schemeID = &s
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
			if s := id.Ext.Get(iso.ExtKeySchemeID).String(); s != "" {
				p.PartyLegalEntity.CompanyID.SchemeID = &s
			}
			break
		}
	}

	return p
}

// iso6523EndpointScheme is the URI scheme used by org.Endpoint for
// Peppol-style participant identifiers (iso6523-actorid-upis::<ICD>:<code>).
const iso6523EndpointScheme = "iso6523-actorid-upis"

// splitISO6523Endpoint extracts the ISO 6523 ICD and participant code from an
// iso6523-actorid-upis endpoint URI.
func splitISO6523Endpoint(uri cbc.URI) (string, string, bool) {
	rest := strings.TrimPrefix(uri.Opaque(), ":")
	icd, code, ok := strings.Cut(rest, ":")
	if !ok || icd == "" || code == "" {
		return "", "", false
	}
	return icd, code, true
}

func normalizeEndpointScheme(s string) string {
	switch strings.ToUpper(s) {
	case oioubl21SchemeGLN:
		return icdGLN
	default:
		return s
	}
}

// oioubl21AddressFormatCode returns the OIOUBL StructuredLax AddressFormatCode
// (codelist addressformatcode-1.1) required on every address (F-LIB025).
// StructuredLax imposes no mandatory sub-fields, matching real NemHandel traffic
// and GOBL's optional address model — we still emit StreetName/BuildingNumber/
// PostalZone whenever GOBL has them.
func oioubl21AddressFormatCode() *IDType {
	listID := "urn:oioubl:codelist:addressformatcode-1.1"
	listAgencyID := "320"
	return &IDType{
		ListID:       &listID,
		ListAgencyID: &listAgencyID,
		Value:        "StructuredLax",
	}
}

func newAddress(addresses []*org.Address, ctx Context) *PostalAddress {
	if len(addresses) == 0 {
		return nil
	}
	// Only return the first a
	a := addresses[0]

	addr := &PostalAddress{}

	if ctx.Is(ContextOIOUBL21) {
		// Every OIOUBL address needs an AddressFormatCode (F-LIB025). Stamping it
		// here covers delivery and payee addresses too, not just the supplier and
		// customer handled by applyOIOUBL21Party.
		addr.AddressFormatCode = oioubl21AddressFormatCode()
		// OIOUBL keeps the street number and PO box in their own elements when
		// GOBL provides them; under StructuredLax these are emitted but not
		// required, so an inline street number is preserved as-is in StreetName.
		if a.Street != "" {
			addr.StreetName = &a.Street
		}
		if a.Number != "" {
			addr.BuildingNumber = &a.Number
		}
		if a.PostOfficeBox != "" {
			addr.Postbox = &a.PostOfficeBox
		}
	} else if a.Street != "" {
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

	if a.Block != "" {
		addr.PlotIdentification = &a.Block
	}

	if a.Country != "" {
		addr.Country = &Country{IdentificationCode: string(a.Country)}
	}

	// OIOUBL forbids cac:LocationCoordinate on an address (F-LIB212).
	if a.Coordinates != nil && !ctx.Is(ContextOIOUBL21) {
		lat := strconv.FormatFloat(*a.Coordinates.Latitude, 'f', -1, 64)
		lon := strconv.FormatFloat(*a.Coordinates.Longitude, 'f', -1, 64)
		addr.LocationCoordinate = &LocationCoordinate{
			LatitudeDegreesMeasure:  &lat,
			LongitudeDegreesMeasure: &lon,
		}
	}

	if ctx.Is(ContextZATCA) {
		l := a.LineTwo()
		addr.CitySubdivisionName = &l
		addr.AdditionalStreetName = nil

		addr.BuildingNumber = &a.Number
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

// OIOUBL symbolic EndpointID schemes (F-LIB179) and the ISO 6523 ICDs they
// correspond to.
const (
	oioubl21SchemeDKCVR = "DK:CVR"
	oioubl21SchemeDKSE  = "DK:SE"
	oioubl21SchemeGLN   = "GLN"
	// oioubl21SchemeZZZ is the OIOUBL "other" company-ID scheme, the only
	// PartyTaxScheme/PartyLegalEntity scheme valid for a non-Danish identifier
	// (F-LIB195 allows {DK:SE, ZZZ}; F-LIB189 allows {DK:CVR, DK:CPR, ZZZ}).
	oioubl21SchemeZZZ = "ZZZ"
	icdGLN            = "0088"
	icdDKCVR          = "0184"
	icdDKSE           = "0198"
)

// oioubl21EndpointSchemes maps ISO 6523 ICDs / Peppol EAS codes to the
// symbolic OIOUBL EndpointID schemeID codelist (F-LIB179) — numeric scheme
// IDs are rejected on the NemHandel wire. Only codes with an unambiguous
// symbolic counterpart are mapped; anything else passes through numerically.
var oioubl21EndpointSchemes = map[string]string{
	"0007":   "SE:ORGNR",
	"0009":   "FR:SIRET",
	"0037":   "FI:OVT",
	"0060":   "DUNS",
	icdGLN:   oioubl21SchemeGLN,
	"0096":   "DK:P",
	icdDKCVR: oioubl21SchemeDKCVR,
	"0192":   "NO:ORGNR",
	"0196":   "IS:KT",
	icdDKSE:  oioubl21SchemeDKSE,
	"0212":   "FI:ORGNR",
	"0213":   "FI:VAT",
	"9902":   oioubl21SchemeDKCVR, // legacy EAS for DK:CVR
	"9906":   "IT:VAT",
	"9907":   "IT:CF",
	"9909":   "NO:VAT",
	"9910":   "HU:VAT",
	"9912":   "EU:VAT",
	"9913":   "EU:REID",
	"9914":   "AT:VAT",
	"9915":   "AT:GOV",
	"9917":   "IS:KT", // legacy EAS for IS:KT
	"9918":   "IBAN",
	"9919":   "AT:KUR",
	"9920":   "ES:VAT",
	"9922":   "AD:VAT",
	"9923":   "AL:VAT",
	"9924":   "BA:VAT",
	"9925":   "BE:VAT",
	"9926":   "BG:VAT",
	"9927":   "CH:VAT",
	"9928":   "CY:VAT",
	"9929":   "CZ:VAT",
	"9930":   "DE:VAT",
	"9931":   "EE:VAT",
	"9932":   "GB:VAT",
	"9933":   "GR:VAT",
	"9934":   "HR:VAT",
	"9935":   "IE:VAT",
	"9936":   "LI:VAT",
	"9937":   "LT:VAT",
	"9938":   "LU:VAT",
	"9939":   "LV:VAT",
	"9940":   "MC:VAT",
	"9941":   "ME:VAT",
	"9942":   "MK:VAT",
	"9943":   "MT:VAT",
	"9944":   "NL:VAT",
	"9945":   "PL:VAT",
	"9946":   "PT:VAT",
	"9947":   "RO:VAT",
	"9948":   "RS:VAT",
	"9949":   "SI:VAT",
	"9950":   "SK:VAT",
	"9951":   "SM:VAT",
	"9952":   "TR:VAT",
	"9953":   "VA:VAT",
	"9955":   "SE:VAT",
}

// oioubl21EndpointICDs restores wire EndpointIDs to ISO 6523 endpoints on
// parse. Inverse of oioubl21EndpointSchemes; symbolic schemes fed by several
// codes restore to the lowest (canonical, non-legacy) one.
var oioubl21EndpointICDs = func() map[string]string {
	m := make(map[string]string, len(oioubl21EndpointSchemes))
	for icd, scheme := range oioubl21EndpointSchemes {
		if cur, ok := m[scheme]; !ok || icd < cur {
			m[scheme] = icd
		}
	}
	return m
}()

// applyOIOUBL21Party rewrites an assembled party into OIOUBL 2.1 form: symbolic
// endpoint scheme + DK-prefixed CVR (F-LIB179/F-LIB180), a fallback PartyName,
// the StructuredLax address format, and the DK:SE/DK:CVR company-ID schemes.
func applyOIOUBL21Party(p *Party) {
	if p == nil {
		return
	}
	if p.EndpointID != nil {
		if mapped, ok := oioubl21EndpointSchemes[p.EndpointID.SchemeID]; ok {
			p.EndpointID.SchemeID = mapped
		}
		// OIOUBL CVR endpoints must carry the DK-prefixed form (F-LIB180).
		if p.EndpointID.SchemeID == oioubl21SchemeDKCVR && !strings.HasPrefix(p.EndpointID.Value, "DK") {
			p.EndpointID.Value = "DK" + p.EndpointID.Value
		}
	}
	if p.PartyName == nil && len(p.PartyIdentification) == 0 {
		if p.PartyLegalEntity != nil && p.PartyLegalEntity.RegistrationName != nil {
			p.PartyName = &PartyName{
				Name: *p.PartyLegalEntity.RegistrationName,
			}
		}
	}
	if p.PostalAddress != nil && p.PostalAddress.AddressFormatCode == nil {
		// Covers a party that has a tax identity but no address (newAddress
		// returns nil, so the bare PostalAddress is created without a format code).
		p.PostalAddress.AddressFormatCode = oioubl21AddressFormatCode()
	}
	// DK:SE/DK:CVR (with the DK-prefixed value the schematron mandates) apply
	// only to Danish parties; a foreign party's tax and legal identifiers carry
	// the OIOUBL "other" scheme ZZZ, with the value left as-is. Forcing DK:SE/
	// DK:CVR + a DK prefix onto a foreign identifier is wire-fatal (F-LIB196/190)
	// or silently corrupting.
	danish := partyIsDanish(p)
	if p.PartyTaxScheme != nil {
		for i := range p.PartyTaxScheme {
			pts := &p.PartyTaxScheme[i]
			if pts.CompanyID != nil {
				if danish {
					scheme := oioubl21SchemeDKSE
					pts.CompanyID.SchemeID = &scheme
					if !strings.HasPrefix(pts.CompanyID.Value, "DK") {
						pts.CompanyID.Value = "DK" + pts.CompanyID.Value
					}
				} else {
					scheme := oioubl21SchemeZZZ
					pts.CompanyID.SchemeID = &scheme
				}
			}
			applyOIOUBL21TaxScheme(pts.TaxScheme)
		}
	}
	if p.PartyLegalEntity != nil && p.PartyLegalEntity.CompanyID != nil {
		if danish {
			scheme := oioubl21SchemeDKCVR
			p.PartyLegalEntity.CompanyID.SchemeID = &scheme
			if !strings.HasPrefix(p.PartyLegalEntity.CompanyID.Value, "DK") {
				p.PartyLegalEntity.CompanyID.Value = "DK" + p.PartyLegalEntity.CompanyID.Value
			}
		} else {
			scheme := oioubl21SchemeZZZ
			p.PartyLegalEntity.CompanyID.SchemeID = &scheme
		}
	}
	applyOIOUBL21PartyIdentifications(p)
}

// applyOIOUBL21PartyIdentifications normalises each PartyIdentification/ID scheme
// to the symbolic OIOUBL PartyID codelist (F-LIB183) — a numeric ICD maps to its
// symbolic scheme, anything unmappable becomes ZZZ — and DK-prefixes DK:CVR/DK:SE
// values (F-LIB184), mirroring the company-ID handling.
func applyOIOUBL21PartyIdentifications(p *Party) {
	for i := range p.PartyIdentification {
		id := p.PartyIdentification[i].ID
		if id == nil || id.SchemeID == nil {
			continue
		}
		scheme := *id.SchemeID
		if mapped, ok := oioubl21EndpointSchemes[scheme]; ok {
			scheme = mapped
		} else if isNumericICDScheme(scheme) {
			scheme = oioubl21SchemeZZZ
		}
		id.SchemeID = &scheme
		if (scheme == oioubl21SchemeDKCVR || scheme == oioubl21SchemeDKSE) && !strings.HasPrefix(id.Value, "DK") {
			id.Value = "DK" + id.Value
		}
	}
}

// isNumericICDScheme reports whether a scheme is a bare 4-digit ISO 6523 ICD
// (e.g. "0184") rather than a symbolic OIOUBL scheme.
func isNumericICDScheme(s string) bool {
	if len(s) != 4 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// partyIsDanish reports whether an assembled OIOUBL party is Danish, the signal
// that decides DK:SE/DK:CVR vs the ZZZ "other" scheme. newParty stamps the tax
// identity's country onto the postal address (party.go), so the country code is
// the reliable marker even when an identifier value carries no country prefix.
func partyIsDanish(p *Party) bool {
	return p.PostalAddress != nil &&
		p.PostalAddress.Country != nil &&
		p.PostalAddress.Country.IdentificationCode == "DK"
}
