package ubl

import (
	"fmt"
	"strconv"
	"strings"

	oioubl "github.com/invopop/gobl.dk.oioubl/addon"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
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
	ID                   *IDType             `xml:"cbc:ID,omitempty"`
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
	Region               *string             `xml:"cbc:Region,omitempty"`
	District             *string             `xml:"cbc:District,omitempty"`
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
	TaxTypeCode *IDType `xml:"cbc:TaxTypeCode,omitempty"`
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

func newParty(party *org.Party, ctx Context) *Party {
	if party == nil {
		return nil
	}
	p := &Party{
		PostalAddress: newAddress(party.Addresses, ctx),
	}
	if party.Name != "" {
		p.PartyName = &PartyName{Name: party.Name}
		p.PartyLegalEntity = &PartyLegalEntity{RegistrationName: &party.Name}
	}
	addPartyTaxScheme(p, party, ctx)
	p.Contact = newPartyContact(party, ctx)
	addPartyEndpoint(p, party, ctx)
	if party.Alias != "" {
		p.PartyName = &PartyName{Name: party.Alias}
	}
	addPartyIdentities(p, party)
	return p
}

// addPartyTaxScheme maps the party's primary tax identity to a PartyTaxScheme and
// stamps its country onto the postal address.
func addPartyTaxScheme(p *Party, party *org.Party, ctx Context) {
	tID := party.TaxID
	if tID == nil || tID.Code == "" {
		return
	}
	code := tID.String()
	// Norwegian VAT numbers require the MVA suffix on the wire
	// (PEPPOL-EN16931 NO-R-001), which GOBL normalization may strip.
	if tID.Country.Code() == l10n.NO && !strings.HasSuffix(code, "MVA") {
		code += "MVA"
	}
	if ctx.Is(ContextZATCA) {
		code = code[2:]
	}
	id := tID.GetScheme()
	if id == cbc.CodeEmpty {
		id = TaxSchemeVAT // Peppol default
	}
	p.PartyTaxScheme = []PartyTaxScheme{{
		CompanyID: &IDType{Value: code},
		TaxScheme: &TaxScheme{ID: IDType{Value: id.String()}},
	}}
	// Override the company address's country code.
	if p.PostalAddress == nil {
		p.PostalAddress = new(PostalAddress)
	}
	p.PostalAddress.Country = &Country{IdentificationCode: tID.Country.String()}
}

// newPartyContact builds the cac:Contact from the party's emails, phones and first
// person, returning nil when none are present. For OIOUBL it sources the mandatory
// cbc:ID (F-INV051) from the person's identity rather than fabricating one.
func newPartyContact(party *org.Party, ctx Context) *Contact {
	contact := &Contact{}
	if len(party.Emails) > 0 {
		contact.ElectronicMail = &party.Emails[0].Address
	}
	if len(party.Telephones) > 0 {
		contact.Telephone = &party.Telephones[0].Number
	}
	if len(party.People) > 0 {
		if n := contactName(party.People[0].Name); n != "" {
			contact.Name = &n
		}
		if ctx.Is(ContextOIOUBL21) {
			if ids := party.People[0].Identities; len(ids) > 0 && ids[0].Code != "" {
				code := ids[0].Code.String()
				contact.ID = &code
			}
		}
	}
	if contact.Name == nil && contact.Telephone == nil && contact.ElectronicMail == nil && contact.ID == nil {
		return nil
	}
	return contact
}

// addPartyEndpoint derives the cbc:EndpointID. For OIOUBL it prefers the ISO 6523
// participant endpoint and lets an explicit dk-oioubl-address-scheme extension
// override the derived scheme (the manual path for a foreign participant); other
// contexts fall back to the first inbox using its raw scheme.
func addPartyEndpoint(p *Party, party *org.Party, ctx Context) {
	if ctx.Is(ContextOIOUBL21) {
		for _, ep := range party.Endpoints {
			if ep == nil {
				continue
			}
			// The participant scheme (DK:CVR/DK:SE/GLN/…) and code are carried in the
			// OIOUBL endpoint URI and emitted 1:1 as the EndpointID schemeID + value.
			if scheme, value, ok := oioubl.ParseOIOUBLEndpoint(ep.URI.String()); ok {
				p.EndpointID = &EndpointID{SchemeID: scheme, Value: value}
				break
			}
		}
	}
	if p.EndpointID == nil && len(party.Inboxes) > 0 {
		ib := party.Inboxes[0]
		if ib.Email != "" {
			p.EndpointID = &EndpointID{SchemeID: SchemeIDEmail, Value: ib.Email}
		} else if ib.Scheme != "" {
			p.EndpointID = &EndpointID{SchemeID: ib.Scheme.String(), Value: ib.Code.String()}
		}
	}
}

// addPartyIdentities classifies the party identities: the first legal-scope one
// becomes PartyLegalEntity.CompanyID, tax-scope ones become additional
// PartyTaxScheme entries, and the rest become PartyIdentification entries.
func addPartyIdentities(p *Party, party *org.Party) {
	firstLegalIdx := -1
	for i, id := range party.Identities {
		if id.Scope != org.IdentityScopeLegal {
			continue
		}
		if p.PartyLegalEntity == nil {
			p.PartyLegalEntity = &PartyLegalEntity{}
		}
		p.PartyLegalEntity.CompanyID = &IDType{Value: id.Code.String()}
		if s := id.Ext.Get(iso.ExtKeySchemeID).String(); s != "" {
			p.PartyLegalEntity.CompanyID.SchemeID = &s
		}
		firstLegalIdx = i
		break
	}
	for _, id := range party.Identities {
		if id.Scope != org.IdentityScopeTax {
			continue
		}
		companyID := &IDType{Value: id.Code.String()}
		if s := id.Ext.Get(iso.ExtKeySchemeID).String(); s != "" {
			companyID.SchemeID = &s
		}
		p.PartyTaxScheme = append(p.PartyTaxScheme, PartyTaxScheme{
			CompanyID: companyID,
			TaxScheme: &TaxScheme{ID: IDType{Value: id.Type.String()}},
		})
	}
	for i, id := range party.Identities {
		if (id.Scope == org.IdentityScopeLegal && i == firstLegalIdx) || id.Scope == org.IdentityScopeTax {
			continue
		}
		idType := &IDType{Value: id.Code.String()}
		if s := id.Ext.Get(iso.ExtKeySchemeID).String(); s != "" {
			idType.SchemeID = &s
		} else if id.Ext.IsZero() {
			// ZATCA has very specific identities that do not require an ISO
			// extension and are only described with type.
			if t := id.Type.String(); t != "" {
				idType.SchemeID = &t
			}
		}
		p.PartyIdentification = append(p.PartyIdentification, Identification{ID: idType})
	}
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

// oioubl21AddressFormatCode builds the cbc:AddressFormatCode (codelist
// addressformatcode-1.1) required on every OIOUBL address (F-LIB025).
func oioubl21AddressFormatCode(value string) *IDType {
	listID := "urn:oioubl:codelist:addressformatcode-1.1"
	listAgencyID := "320"
	return &IDType{
		ListID:       &listID,
		ListAgencyID: &listAgencyID,
		Value:        value,
	}
}

// OIOUBL address extension keys and values, sourced from the dk-oioubl addon (the
// single source of truth) so the converter and addon never drift. The converter
// reads them as plain party extensions (GOBL has no address-level extension).
const (
	oioubl21AddressFormatKey = oioubl.ExtKeyAddressFormat

	oioubl21AddressStructuredDK     = string(oioubl.ExtValueAddressFormatStructuredDK)
	oioubl21AddressStructuredLax    = string(oioubl.ExtValueAddressFormatStructuredLax)
	oioubl21AddressUnstructured     = string(oioubl.ExtValueAddressFormatUnstructured)
	oioubl21AddressStructuredID     = string(oioubl.ExtValueAddressFormatStructuredID)
	oioubl21AddressStructuredRegion = string(oioubl.ExtValueAddressFormatStructuredRegion)

	// oioubl21AddressIDScheme (GLN) and its GS1 agency are wire-serialization
	// attributes OIOUBL mandates on a StructuredID address ID (F-LIB028/029).
	oioubl21AddressIDScheme = string(oioubl.SchemeGLN)
	oioubl21GLNAgencyID     = "9"
)

// applyOIOUBL21AddressFormat reshapes a party's postal address to its declared
// dk-oioubl-address-format, dropping the elements each restricted format forbids
// (F-LIB031/038/040). Must run after applyOIOUBL21Party, which needs the address
// country before the restricted formats drop it.
func applyOIOUBL21AddressFormat(addr *PostalAddress, party *org.Party) {
	if addr == nil || party == nil {
		return
	}
	format := party.Ext.Get(oioubl21AddressFormatKey)
	if format == "" {
		return
	}
	addr.AddressFormatCode = oioubl21AddressFormatCode(format.String())
	switch format.String() {
	case oioubl21AddressUnstructured:
		// F-LIB031: an Unstructured address carries only AddressLine.
		lines := oioubl21AddressLines(party)
		clearStructuredAddress(addr)
		addr.AddressLine = lines
	case oioubl21AddressStructuredID:
		// F-LIB038: a StructuredID address carries only the identifier. GOBL has no
		// address-identifier field, so the register GLN rides org.Address.Number
		// (mapped to BuildingNumber by newAddress and idle in this format, which
		// clears every postal element). Re-emit it as cbc:ID with the mandatory GLN
		// schemeID (F-LIB028/029), the scheme OIOUBL uses for every GLN identifier.
		id := ""
		if addr.BuildingNumber != nil {
			id = *addr.BuildingNumber
		}
		clearStructuredAddress(addr)
		if id != "" {
			scheme := oioubl21AddressIDScheme
			agency := oioubl21GLNAgencyID
			addr.ID = &IDType{Value: id, SchemeID: &scheme, SchemeAgencyID: &agency}
		}
	case oioubl21AddressStructuredRegion:
		// F-LIB040: a StructuredRegion address carries only Region, District and
		// Country. newAddress mapped the GOBL region to CountrySubentity and the
		// locality to CityName; move the region to cbc:Region and reinterpret the
		// locality (org.Address defines it as "village, town, district, or city")
		// as the district OIOUBL requires here.
		region := addr.CountrySubentity
		country := addr.Country
		district := addr.CityName
		clearStructuredAddress(addr)
		addr.Region = region
		addr.Country = country
		addr.District = district
	}
	// StructuredDK and StructuredLax keep the structured fields as built.
}

// clearStructuredAddress blanks every postal element except the
// AddressFormatCode, leaving a canvas for a format-specific rebuild.
func clearStructuredAddress(addr *PostalAddress) {
	addr.ID = nil
	addr.Postbox = nil
	addr.StreetName = nil
	addr.AdditionalStreetName = nil
	addr.BuildingNumber = nil
	addr.PlotIdentification = nil
	addr.CitySubdivisionName = nil
	addr.CityName = nil
	addr.PostalZone = nil
	addr.CountrySubentity = nil
	addr.Region = nil
	addr.District = nil
	addr.AddressLine = nil
	addr.Country = nil
	addr.LocationCoordinate = nil
}

// oioubl21AddressLines renders a GOBL address as OIOUBL free-text AddressLine
// elements for an Unstructured address.
func oioubl21AddressLines(party *org.Party) []AddressLine {
	if len(party.Addresses) == 0 {
		return nil
	}
	a := party.Addresses[0]
	var lines []AddressLine
	if one := a.LineOne(); one != "" {
		lines = append(lines, AddressLine{Line: one})
	} else if a.PostOfficeBox != "" {
		lines = append(lines, AddressLine{Line: a.PostOfficeBox})
	}
	if two := a.LineTwo(); two != "" {
		lines = append(lines, AddressLine{Line: two})
	}
	if loc := strings.TrimSpace(a.Code.String() + " " + a.Locality); loc != "" {
		lines = append(lines, AddressLine{Line: loc})
	}
	return lines
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
		addr.AddressFormatCode = oioubl21AddressFormatCode("StructuredLax")
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

// OIOUBL symbolic schemes (F-LIB179), defined by the dk-oioubl addon (the single
// source of truth). The ICD<->scheme codelist also lives in the addon, reached via
// oioubl.SchemeForICD (convert) and oioubl.ICDForScheme (parse).
const (
	oioubl21SchemeDKCVR = oioubl.SchemeDKCVR
	oioubl21SchemeDKSE  = oioubl.SchemeDKSE
	oioubl21SchemeZZZ   = oioubl.SchemeZZZ
)

// dkPrefixed adds the "DK" country prefix the OIOUBL schematron mandates on
// DK:CVR/DK:SE identifier values (F-LIB180/F-LIB184), only when absent.
func dkPrefixed(value string) string {
	if strings.HasPrefix(value, "DK") {
		return value
	}
	return "DK" + value
}

// applyOIOUBL21CompanyID stamps a CompanyID's OIOUBL scheme: a Danish party gets
// the given Danish scheme with the DK-prefixed value the schematron mandates
// (F-LIB190/196); a foreign party gets the "other" scheme ZZZ with its value left
// as-is, since forcing a DK scheme + prefix onto a foreign identifier is wire-fatal.
// A nil CompanyID is ignored.
func applyOIOUBL21CompanyID(id *IDType, danishScheme string, danish bool) {
	if id == nil {
		return
	}
	if danish {
		id.SchemeID = &danishScheme
		id.Value = dkPrefixed(id.Value)
		return
	}
	scheme := oioubl21SchemeZZZ
	id.SchemeID = &scheme
}

// applyOIOUBL21Party rewrites an assembled party into OIOUBL 2.1 form: symbolic
// endpoint scheme + DK-prefixed CVR (F-LIB179/F-LIB180), a fallback PartyName,
// the StructuredLax address format, and the DK:SE/DK:CVR company-ID schemes.
func applyOIOUBL21Party(p *Party) {
	if p == nil {
		return
	}
	if p.EndpointID != nil && p.EndpointID.SchemeID == oioubl21SchemeDKCVR {
		// The schemeID is the dk-oioubl-address-scheme extension value (set in
		// newParty), emitted 1:1. OIOUBL CVR endpoints must carry the DK-prefixed
		// form (F-LIB180).
		p.EndpointID.Value = dkPrefixed(p.EndpointID.Value)
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
		p.PostalAddress.AddressFormatCode = oioubl21AddressFormatCode("StructuredLax")
	}
	danish := partyIsDanish(p)
	for i := range p.PartyTaxScheme {
		pts := &p.PartyTaxScheme[i]
		applyOIOUBL21CompanyID(pts.CompanyID, oioubl21SchemeDKSE, danish)
		applyOIOUBL21TaxScheme(pts.TaxScheme)
	}
	if p.PartyLegalEntity != nil {
		applyOIOUBL21CompanyID(p.PartyLegalEntity.CompanyID, oioubl21SchemeDKCVR, danish)
	}
	applyOIOUBL21PartyIdentifications(p)
}

// applyOIOUBL21TaxRepParty drops the elements OIOUBL forbids on a
// cac:TaxRepresentativeParty (EndpointID, PartyIdentification, PartyLegalEntity,
// Contact) and runs the standard OIOUBL party pass on what remains.
func applyOIOUBL21TaxRepParty(p *Party) {
	if p == nil {
		return
	}
	p.EndpointID = nil
	p.PartyIdentification = nil
	p.PartyLegalEntity = nil
	p.Contact = nil
	applyOIOUBL21Party(p)
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
		if mapped := oioubl.SchemeForICD(scheme); mapped != "" {
			scheme = mapped.String()
		} else if isNumericICDScheme(scheme) {
			scheme = oioubl21SchemeZZZ
		}
		id.SchemeID = &scheme
		if scheme == oioubl21SchemeDKCVR || scheme == oioubl21SchemeDKSE {
			id.Value = dkPrefixed(id.Value)
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
