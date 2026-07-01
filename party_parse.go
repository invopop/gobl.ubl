package ubl

import (
	"strings"

	oioubl "github.com/invopop/gobl.dk.oioubl/addon"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

func goblParty(party *Party, o *options) *org.Party {
	if party == nil {
		return nil
	}
	p := &org.Party{}

	if party.PartyLegalEntity != nil && party.PartyLegalEntity.RegistrationName != nil {
		p.Name = cleanString(*party.PartyLegalEntity.RegistrationName)
	}

	if eID := party.EndpointID; eID != nil {
		switch {
		case eID.SchemeID == "EM": // email
			p.Inboxes = append(p.Inboxes, &org.Inbox{Email: eID.Value})
		case o.context.Is(ContextOIOUBL21):
			// OIOUBL participants are restored as org.Endpoints under the OIOUBL
			// endpoint-identifier scheme (org.Inbox is deprecated). The symbolic
			// scheme and code round-trip verbatim; only the wire-only DK prefix
			// (F-LIB180) on a Danish identifier is reversed.
			code := eID.Value
			if eID.SchemeID == oioubl21SchemeDKCVR || eID.SchemeID == oioubl21SchemeDKSE {
				code = strings.TrimPrefix(code, "DK")
			}
			p.Endpoints = append(p.Endpoints, &org.Endpoint{
				URI: cbc.URI(oioubl.OIOUBLEndpointURI(eID.SchemeID, code)),
			})
		default:
			p.Inboxes = append(p.Inboxes, &org.Inbox{
				Scheme: cbc.Code(eID.SchemeID),
				Code:   cbc.Code(eID.Value),
			})
		}
	}

	if party.PartyName != nil {
		if p.Name == "" {
			p.Name = cleanString(party.PartyName.Name)
		} else if party.PartyName.Name != p.Name {
			// Only set alias if it's different from the name
			p.Alias = cleanString(party.PartyName.Name)
		}
	}

	if c := party.Contact; c != nil {
		person := new(org.Person)
		if c.Name != nil {
			person.Name = &org.Name{
				Given: cleanString(*c.Name),
			}
		}
		// OIOUBL carries the contact reference in cac:Contact/cbc:ID; restore it
		// to the person's identities so the round-trip stays lossless (the
		// outbound side sources Contact/ID from person.Identities for F-INV051).
		if c.ID != nil && o.context.Is(ContextOIOUBL21) {
			if code := cleanString(*c.ID); code != "" {
				person.Identities = []*org.Identity{{Code: cbc.Code(code)}}
			}
		}
		if person.Name != nil || len(person.Identities) > 0 {
			p.People = []*org.Person{person}
		}
	}

	if party.PostalAddress != nil {
		p.Addresses = []*org.Address{
			parseAddress(party.PostalAddress),
		}
		if o.context.Is(ContextOIOUBL21) {
			applyOIOUBL21AddressFormatParse(party.PostalAddress, p)
		}
	}

	if party.Contact != nil {
		if party.Contact.Telephone != nil {
			p.Telephones = []*org.Telephone{
				{
					Number: cleanString(*party.Contact.Telephone),
				},
			}
		}
		if party.Contact.ElectronicMail != nil {
			p.Emails = []*org.Email{
				{
					Address: cleanString(*party.Contact.ElectronicMail),
				},
			}
		}
	}

	handleLegalEntityIdentity(party, p)
	handlePartyTaxSchemes(party, p, o)
	handlePartyIdentifications(party, p, o)

	return p
}

// goblDeliveryParty creates a GOBL party with only the BTs available
// for the delivery party (BT-70 name). Address is handled separately
// via DeliveryLocation.
func goblDeliveryParty(party *Party) *org.Party {
	if party == nil {
		return nil
	}
	p := &org.Party{}

	if party.PartyLegalEntity != nil && party.PartyLegalEntity.RegistrationName != nil {
		p.Name = cleanString(*party.PartyLegalEntity.RegistrationName)
	}
	if party.PartyName != nil {
		if p.Name == "" {
			p.Name = cleanString(party.PartyName.Name)
		}
	}

	if p.Name == "" {
		return nil
	}
	return p
}

func parseAddress(address *PostalAddress) *org.Address {
	if address == nil {
		return nil
	}

	addr := new(org.Address)
	if address.Country != nil {
		addr.Country = l10n.ISOCountryCode(address.Country.IdentificationCode)
	}
	if address.StreetName != nil {
		addr.Street = cleanString(*address.StreetName)
	}
	if address.AdditionalStreetName != nil {
		addr.StreetExtra = cleanString(*address.AdditionalStreetName)
	}
	if address.CityName != nil {
		addr.Locality = cleanString(*address.CityName)
	}
	if address.PostalZone != nil {
		addr.Code = cbc.Code(cleanString(*address.PostalZone))
	}
	if address.CountrySubentity != nil {
		addr.Region = cleanString(*address.CountrySubentity)
	}
	// A StructuredRegion address carries its region in cbc:Region rather than
	// cbc:CountrySubentity (F-LIB040). No other profile emits cbc:Region.
	if address.Region != nil && addr.Region == "" {
		addr.Region = cleanString(*address.Region)
	}
	// A StructuredRegion address carries the locality in cbc:District (F-LIB040);
	// org.Address.Locality is its district-level field ("village, town, district,
	// or city").
	if address.District != nil && addr.Locality == "" {
		addr.Locality = cleanString(*address.District)
	}
	if address.BuildingNumber != nil {
		addr.Number = cleanString(*address.BuildingNumber)
	}
	// A StructuredID address is reduced to a single register identifier (a GLN) in
	// cbc:ID (F-LIB037/038). GOBL has no address-identifier field, so the value
	// rides org.Address.Number (idle in this format, which clears all postal
	// fields); the emit side re-reads it from there.
	if address.AddressFormatCode != nil &&
		address.AddressFormatCode.Value == oioubl21AddressStructuredID &&
		address.ID != nil {
		addr.Number = cleanString(address.ID.Value)
	}
	if address.Postbox != nil {
		addr.PostOfficeBox = cleanString(*address.Postbox)
	}
	// CitySubdivisionName is used by ZATCA to represent the district,
	// which maps to StreetExtra in GOBL.
	if address.CitySubdivisionName != nil && addr.StreetExtra == "" {
		addr.StreetExtra = cleanString(*address.CitySubdivisionName)
	}
	// Unstructured addresses (OIOUBL AddressFormatCode "Unstructured") carry
	// their content as free-text cac:AddressLine rather than the structured
	// fields above. Fall back to it so the content survives the parse: the first
	// line becomes the street, any remaining lines the street extra.
	if addr.Street == "" && len(address.AddressLine) > 0 {
		var lines []string
		for _, l := range address.AddressLine {
			if s := cleanString(l.Line); s != "" {
				lines = append(lines, s)
			}
		}
		if len(lines) > 0 {
			addr.Street = lines[0]
			if len(lines) > 1 && addr.StreetExtra == "" {
				addr.StreetExtra = strings.Join(lines[1:], ", ")
			}
		}
	}
	return addr
}

// applyOIOUBL21AddressFormatParse restores the wire cbc:AddressFormatCode to the
// dk-oioubl-address-format extension so the format round-trips. StructuredLax is
// the default and carries no extension; the StructuredID id and StructuredRegion
// region/district round-trip through org.Address fields (Number, Region,
// Locality), so those formats need only the format extension.
//
// An unrecognized value carries no extension: an alternative codelist such as
// UN/ECE 3477 (the interop AddressFormatCode used by non-OIOUBL senders,
// OIOUBL_GUIDE_PARTIES §3.1.6) has no GOBL representation, so the address still
// imports through its structured fields but is left as the lax default.
func applyOIOUBL21AddressFormatParse(address *PostalAddress, p *org.Party) {
	if address == nil || address.AddressFormatCode == nil {
		return
	}
	format := address.AddressFormatCode.Value
	if format == oioubl21AddressStructuredLax || !isOIOUBLAddressFormat(format) {
		return
	}
	p.Ext = tax.ExtensionsOf(cbc.CodeMap{oioubl21AddressFormatKey: cbc.Code(format)})
}

// isOIOUBLAddressFormat reports whether the value is one of OIOUBL's own
// addressformatcode-1.1 codes, as opposed to an alternative codelist (§3.1.6).
func isOIOUBLAddressFormat(format string) bool {
	switch format {
	case oioubl21AddressStructuredDK, oioubl21AddressStructuredLax,
		oioubl21AddressStructuredID, oioubl21AddressStructuredRegion,
		oioubl21AddressUnstructured:
		return true
	}
	return false
}

func handleLegalEntityIdentity(party *Party, p *org.Party) {
	if party.PartyLegalEntity == nil || party.PartyLegalEntity.CompanyID == nil {
		return
	}
	if p.Identities == nil {
		p.Identities = make([]*org.Identity, 0)
	}
	identity := &org.Identity{
		Code:  cbc.Code(party.PartyLegalEntity.CompanyID.Value),
		Scope: org.IdentityScopeLegal,
	}
	if party.PartyLegalEntity.CompanyID.SchemeID != nil {
		identity.Ext = tax.ExtensionsOf(cbc.CodeMap{
			iso.ExtKeySchemeID: cbc.Code(*party.PartyLegalEntity.CompanyID.SchemeID),
		})
	}
	p.Identities = append(p.Identities, identity)
}

func handlePartyTaxSchemes(party *Party, p *org.Party, o *options) {
	if len(party.PartyTaxScheme) == 0 {
		return
	}

	cc := party.resolveCountry(o.context)
	validSchemes := extractValidTaxSchemes(party.PartyTaxScheme)

	if len(validSchemes) == 1 {
		setTaxIDFromScheme(validSchemes[0], p, cc)
	} else if len(validSchemes) > 1 {
		handleMultipleTaxSchemes(validSchemes, p, cc)
	}
}

// resolveCountry returns the party country for tax-identity parsing. An OIOUBL
// StructuredID address carries only an identifier (F-LIB038), so the postal
// address has no country to derive it from; fall back to the DK:SE/DK:CVR
// company-ID scheme, which only a Danish party carries, so the tax-id country
// and the DK-prefix strip still resolve.
func (p *Party) resolveCountry(ctx Context) string {
	if c := p.CountryCode(); c != "" {
		return c
	}
	if ctx.Is(ContextOIOUBL21) && p.hasDanishCompanyScheme() {
		return "DK"
	}
	return ""
}

// hasDanishCompanyScheme reports whether any tax-scheme or legal-entity company
// ID carries a Danish OIOUBL scheme (DK:SE/DK:CVR).
func (p *Party) hasDanishCompanyScheme() bool {
	for _, pts := range p.PartyTaxScheme {
		if id := pts.CompanyID; id != nil && id.SchemeID != nil &&
			(*id.SchemeID == oioubl21SchemeDKSE || *id.SchemeID == oioubl21SchemeDKCVR) {
			return true
		}
	}
	if le := p.PartyLegalEntity; le != nil && le.CompanyID != nil && le.CompanyID.SchemeID != nil &&
		(*le.CompanyID.SchemeID == oioubl21SchemeDKSE || *le.CompanyID.SchemeID == oioubl21SchemeDKCVR) {
		return true
	}
	return false
}

func extractValidTaxSchemes(schemes []PartyTaxScheme) []PartyTaxScheme {
	validSchemes := make([]PartyTaxScheme, 0)
	for _, pts := range schemes {
		if pts.CompanyID != nil && pts.CompanyID.Value != "" && pts.TaxScheme != nil {
			validSchemes = append(validSchemes, pts)
		}
	}
	return validSchemes
}

func setTaxIDFromScheme(pts PartyTaxScheme, p *org.Party, countryCode string) {
	p.TaxID = &tax.Identity{
		Country: l10n.TaxCountryCode(countryCode),
		Code:    cbc.Code(pts.CompanyID.Value),
	}
	sc := goblTaxSchemeCategory(pts.TaxScheme.ID.Value)
	if p.TaxID.GetScheme() != sc {
		var scheme cbc.Code
		if pts.TaxScheme.TaxTypeCode != nil && pts.TaxScheme.TaxTypeCode.Value != "" {
			scheme = cbc.Code(pts.TaxScheme.TaxTypeCode.Value)
		} else {
			scheme = sc
		}
		p.TaxID.Scheme = scheme
	}
}

func handleMultipleTaxSchemes(validSchemes []PartyTaxScheme, p *org.Party, countryCode string) {
	// Multiple tax schemes: look for VAT, otherwise use first
	vatIdx := findVATSchemeIndex(validSchemes)

	// Use VAT if found, otherwise first one
	taxIDIdx := 0
	if vatIdx != -1 {
		taxIDIdx = vatIdx
	}

	// Set TaxID from chosen scheme
	setTaxIDFromScheme(validSchemes[taxIDIdx], p, countryCode)

	// Rest become identities with tax scope
	addRemainingTaxSchemesAsIdentities(validSchemes, taxIDIdx, p, countryCode)
}

func findVATSchemeIndex(schemes []PartyTaxScheme) int {
	for i, pts := range schemes {
		if goblTaxSchemeCategory(pts.TaxScheme.ID.Value) == cbc.Code(TaxSchemeVAT) {
			return i
		}
	}
	return -1
}

func addRemainingTaxSchemesAsIdentities(validSchemes []PartyTaxScheme, taxIDIdx int, p *org.Party, countryCode string) {
	for i, pts := range validSchemes {
		if i == taxIDIdx {
			continue
		}

		identity := &org.Identity{
			Country: l10n.ISOCountryCode(countryCode),
			Code:    cbc.Code(pts.CompanyID.Value),
			Scope:   org.IdentityScopeTax,
			Type:    goblTaxSchemeCategory(pts.TaxScheme.ID.Value),
		}

		if p.Identities == nil {
			p.Identities = make([]*org.Identity, 0)
		}
		p.Identities = append(p.Identities, identity)
	}
}

func handlePartyIdentifications(party *Party, p *org.Party, o *options) {
	for _, partyID := range party.PartyIdentification {
		if partyID.ID != nil {
			code := partyID.ID.Value
			identity := &org.Identity{}
			if partyID.ID.SchemeID != nil {
				s := *partyID.ID.SchemeID
				if o.context.Is(ContextZATCA) {
					identity.Type = cbc.Code(s)
				} else {
					identity.Ext = tax.ExtensionsOf(cbc.CodeMap{
						iso.ExtKeySchemeID: cbc.Code(s),
					})
				}
				if o.context.Is(ContextOIOUBL21) && (s == oioubl21SchemeDKCVR || s == oioubl21SchemeDKSE) {
					// Reverse the wire-only DK prefix (F-LIB180), matching the
					// endpoint parse and gobl's canonical country-prefix-free codes.
					code = strings.TrimPrefix(code, "DK")
				}
			}
			identity.Code = cbc.Code(code)
			if p.Identities == nil {
				p.Identities = make([]*org.Identity, 0)
			}
			p.Identities = append(p.Identities, identity)
		}
	}
}
