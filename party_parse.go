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
		p.Name = *party.PartyLegalEntity.RegistrationName
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
			p.Name = party.PartyName.Name
		} else {
			p.Alias = party.PartyName.Name
		}
	}

	if party.Contact != nil && party.Contact.Name != nil {
		p.People = []*org.Person{
			{
				Name: &org.Name{
					Given: *party.Contact.Name,
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
					Number: *party.Contact.Telephone,
				},
			}
		}
		if party.Contact.ElectronicMail != nil {
			p.Emails = []*org.Email{
				{
					Address: *party.Contact.ElectronicMail,
				},
			}
		}
	}

	if party.PartyLegalEntity != nil && party.PartyLegalEntity.CompanyID != nil {
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

	if len(party.PartyTaxScheme) > 0 {
		// Handle multiple party tax schemes
		// If multiple schemes exist, look for VAT first, otherwise use first valid one
		// Remaining schemes become identities with tax scope

		validSchemes := make([]PartyTaxScheme, 0)
		for _, pts := range party.PartyTaxScheme {
			if pts.CompanyID != nil && *pts.CompanyID != "" && pts.TaxScheme != nil {
				validSchemes = append(validSchemes, pts)
			}
		}

		if len(validSchemes) == 1 {
			// Single tax scheme -> becomes TaxID
			pts := validSchemes[0]
			p.TaxID = &tax.Identity{
				Country: l10n.TaxCountryCode(party.CountryCode()),
				Code:    cbc.Code(*pts.CompanyID),
			}
			sc := cbc.Code(pts.TaxScheme.ID)
			if p.TaxID.GetScheme() != sc {
				var scheme cbc.Code
				if pts.TaxScheme.TaxTypeCode != "" {
					scheme = cbc.Code(pts.TaxScheme.TaxTypeCode)
				} else {
					scheme = cbc.Code(pts.TaxScheme.ID)
				}
				p.TaxID.Scheme = scheme
			}
		} else if len(validSchemes) > 1 {
			// Multiple tax schemes: look for VAT, otherwise use first
			vatIdx := -1
			for i, pts := range validSchemes {
				if pts.TaxScheme.ID == "VAT" {
					vatIdx = i
					break
				}
			}

			// Use VAT if found, otherwise first one
			taxIDIdx := 0
			if vatIdx != -1 {
				taxIDIdx = vatIdx
			}

			// Set TaxID from chosen scheme
			pts := validSchemes[taxIDIdx]
			p.TaxID = &tax.Identity{
				Country: l10n.TaxCountryCode(party.CountryCode()),
				Code:    cbc.Code(*pts.CompanyID),
			}
			sc := cbc.Code(pts.TaxScheme.ID)
			if p.TaxID.GetScheme() != sc {
				var scheme cbc.Code
				if pts.TaxScheme.TaxTypeCode != "" {
					scheme = cbc.Code(pts.TaxScheme.TaxTypeCode)
				} else {
					scheme = cbc.Code(pts.TaxScheme.ID)
				}
				p.TaxID.Scheme = scheme
			}

			// Rest become identities with tax scope
			for i, pts := range validSchemes {
				if i == taxIDIdx {
					continue
				}

				identity := &org.Identity{
					Country: l10n.ISOCountryCode(party.CountryCode()),
					Code:    cbc.Code(*pts.CompanyID),
					Scope:   org.IdentityScopeTax,
					Type:    cbc.Code(pts.TaxScheme.ID),
				}

				if p.Identities == nil {
					p.Identities = make([]*org.Identity, 0)
				}
				p.Identities = append(p.Identities, identity)

				// If this non-VAT scheme is becoming an identity and we don't have a TaxID yet,
				// create an empty TaxID with just the country
				if pts.TaxScheme.ID != "VAT" && p.TaxID == nil {
					p.TaxID = &tax.Identity{
						Country: l10n.TaxCountryCode(party.CountryCode()),
					}
				}
			}
		}
	}

	// Handle multiple PartyIdentifications
	for _, partyID := range party.PartyIdentification {
		if partyID.ID != nil && partyID.ID.SchemeID != nil {
			s := *partyID.ID.SchemeID
			identity := &org.Identity{
				Ext: tax.Extensions{
					iso.ExtKeySchemeID: cbc.Code(s),
				},
				Code: cbc.Code(partyID.ID.Value),
			}
			if p.Identities == nil {
				p.Identities = make([]*org.Identity, 0)
			}
			p.Identities = append(p.Identities, identity)
		}
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
		addr.Street = *address.StreetName
	}
	if address.AdditionalStreetName != nil {
		addr.StreetExtra = *address.AdditionalStreetName
	}
	if address.CityName != nil {
		addr.Locality = *address.CityName
	}
	if address.PostalZone != nil {
		addr.Code = cbc.Code(*address.PostalZone)
	}
	if address.CountrySubentity != nil {
		addr.Region = *address.CountrySubentity
	}
	return addr
}
