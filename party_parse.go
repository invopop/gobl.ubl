package ubl

import (
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

func goblParty(party *Party) *org.Party {
	if party == nil {
		return nil
	}
	p := &org.Party{}

	if party.PartyLegalEntity != nil && party.PartyLegalEntity.RegistrationName != nil {
		p.Name = cleanString(*party.PartyLegalEntity.RegistrationName)
	}

	if eID := party.EndpointID; eID != nil {
		oi := new(org.Inbox)
		switch eID.SchemeID {
		case "EM": // email
			oi.Email = eID.Value
		default:
			oi.Scheme = cbc.Code(eID.SchemeID)
			oi.Code = cbc.Code(eID.Value)
		}
		p.Inboxes = append(p.Inboxes, oi)
	}

	if party.PartyName != nil {
		if p.Name == "" {
			p.Name = cleanString(party.PartyName.Name)
		} else if party.PartyName.Name != p.Name {
			// Only set alias if it's different from the name
			p.Alias = cleanString(party.PartyName.Name)
		}
	}

	if party.Contact != nil && party.Contact.Name != nil {
		p.People = []*org.Person{
			{
				Name: &org.Name{
					Given: cleanString(*party.Contact.Name),
				},
			},
		}
	}

	if party.PostalAddress != nil {
		p.Addresses = []*org.Address{
			parseAddress(party.PostalAddress),
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
	handlePartyIdentifications(party, p)

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
	return addr
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
		identity.Ext = tax.Extensions{
			iso.ExtKeySchemeID: cbc.Code(*party.PartyLegalEntity.CompanyID.SchemeID),
		}
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
	sc := cbc.Code(pts.TaxScheme.ID.Value)
	if p.TaxID.GetScheme() != sc {
		var scheme cbc.Code
		if pts.TaxScheme.TaxTypeCode != "" {
			scheme = cbc.Code(pts.TaxScheme.TaxTypeCode)
		} else {
			scheme = cbc.Code(pts.TaxScheme.ID.Value)
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
		if pts.TaxScheme.ID.Value == TaxSchemeVAT {
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
			Type:    cbc.Code(pts.TaxScheme.ID.Value),
		}

		if p.Identities == nil {
			p.Identities = make([]*org.Identity, 0)
		}
		p.Identities = append(p.Identities, identity)
	}
}

func handlePartyIdentifications(party *Party, p *org.Party) {
	for _, partyID := range party.PartyIdentification {
		if partyID.ID != nil {
			identity := &org.Identity{
				Code: cbc.Code(partyID.ID.Value),
			}
			if partyID.ID.SchemeID != nil {
				s := *partyID.ID.SchemeID
				identity.Ext = tax.Extensions{
					iso.ExtKeySchemeID: cbc.Code(s),
				}
			}
			if p.Identities == nil {
				p.Identities = make([]*org.Identity, 0)
			}
			p.Identities = append(p.Identities, identity)
		}
	}
}
