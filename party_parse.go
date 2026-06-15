package ubl

import (
	"strings"

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
			// OIOUBL participants are restored as ISO 6523 endpoints, the
			// going-forward GOBL routing model. Symbolic schemes without an
			// ICD equivalent fall back to an inbox so no identifier is lost.
			if icd, ok := oioubl21EndpointICDs[eID.SchemeID]; ok {
				code := eID.Value
				if eID.SchemeID == oioubl21SchemeDKCVR {
					// Reverse the wire-only DK prefix (F-LIB180).
					code = strings.TrimPrefix(code, "DK")
				}
				p.Endpoints = append(p.Endpoints, &org.Endpoint{
					URI: cbc.URI(iso6523EndpointScheme + "::" + icd + ":" + code),
				})
			} else {
				p.Inboxes = append(p.Inboxes, &org.Inbox{
					Scheme: cbc.Code(eID.SchemeID),
					Code:   cbc.Code(eID.Value),
				})
			}
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
	handlePartyTaxSchemes(party, p)
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
	if address.BuildingNumber != nil {
		addr.Number = cleanString(*address.BuildingNumber)
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

// applyOIOUBL21AddressFormatParse restores the OIOUBL address format declared on
// the wire (cbc:AddressFormatCode) to the GOBL party extensions the emit side
// reads (see applyOIOUBL21AddressFormat), so the format round-trips. StructuredLax
// is the default form newAddress emits for an address without a declared format,
// so it carries no extension. The StructuredID identifier (cbc:ID) and the
// StructuredRegion district (cbc:District) are not modelled by org.Address and are
// read back onto the party extension.
func applyOIOUBL21AddressFormatParse(address *PostalAddress, p *org.Party) {
	if address == nil || address.AddressFormatCode == nil {
		return
	}
	format := address.AddressFormatCode.Value
	if format == "" || format == oioubl21AddressStructuredLax {
		return
	}
	exts := cbc.CodeMap{oioubl21AddressFormatKey: cbc.Code(format)}
	switch format {
	case oioubl21AddressStructuredID:
		// F-LIB038: the identifier lives in cbc:ID, which GOBL does not model on
		// the address.
		if address.ID != nil {
			if id := cleanString(address.ID.Value); id != "" {
				exts[oioubl21AddressIDKey] = cbc.Code(id)
			}
		}
	case oioubl21AddressStructuredRegion:
		// F-LIB040: cbc:District is not modelled by GOBL; the region is parsed
		// into org.Address.Region by parseAddress.
		if address.District != nil {
			if d := cleanString(*address.District); d != "" {
				exts[oioubl21AddressDistrictKey] = cbc.Code(d)
			}
		}
	}
	p.Ext = tax.ExtensionsOf(exts)
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

func handlePartyTaxSchemes(party *Party, p *org.Party) {
	if len(party.PartyTaxScheme) == 0 {
		return
	}

	validSchemes := extractValidTaxSchemes(party.PartyTaxScheme)

	if len(validSchemes) == 1 {
		setTaxIDFromScheme(validSchemes[0], p, party.CountryCode())
	} else if len(validSchemes) > 1 {
		handleMultipleTaxSchemes(validSchemes, p, party.CountryCode())
	}
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
		if pts.TaxScheme.TaxTypeCode != "" {
			scheme = cbc.Code(pts.TaxScheme.TaxTypeCode)
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
			identity := &org.Identity{
				Code: cbc.Code(partyID.ID.Value),
			}
			if partyID.ID.SchemeID != nil {
				s := *partyID.ID.SchemeID
				if o.context.Is(ContextZATCA) {
					identity.Type = cbc.Code(s)
				} else {
					identity.Ext = tax.ExtensionsOf(cbc.CodeMap{
						iso.ExtKeySchemeID: cbc.Code(s),
					})
				}
			}
			if p.Identities == nil {
				p.Identities = make([]*org.Identity, 0)
			}
			p.Identities = append(p.Identities, identity)
		}
	}
}
