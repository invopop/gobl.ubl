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
		// id := getIdentity(party.PartyLegalEntity.CompanyID)
		p.Identities = append(p.Identities, &org.Identity{
			Label: "CompanyID",
			Code:  cbc.Code(party.PartyLegalEntity.CompanyID.Value),
		})
	}

	if len(party.PartyTaxScheme) > 0 {
		// There may be more than one party tax scheme, Peppol allows two
		// for example. We take the first valid entry that has a Tax Scheme
		// as the source of truth, and store the rest as identities.
		for _, pts := range party.PartyTaxScheme {
			if pts.CompanyID == nil || *pts.CompanyID == "" {
				continue
			}
			if pts.TaxScheme != nil && p.TaxID == nil {
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
			} else {
				id := &org.Identity{
					Country: l10n.ISOCountryCode(party.CountryCode()),
					Code:    cbc.Code(*pts.CompanyID),
				}
				if p.Identities == nil {
					p.Identities = make([]*org.Identity, 0)
				}
				p.Identities = append(p.Identities, id)
			}
		}
	}

	if party.PartyIdentification != nil &&
		party.PartyIdentification.ID != nil &&
		party.PartyIdentification.ID.SchemeID != nil {
		s := *party.PartyIdentification.ID.SchemeID
		identity := &org.Identity{
			Ext: tax.Extensions{
				iso.ExtKeySchemeID: cbc.Code(s),
			},
			Code: cbc.Code(party.PartyIdentification.ID.Value),
		}
		if p.Identities == nil {
			p.Identities = make([]*org.Identity, 0)
		}
		p.Identities = append(p.Identities, identity)
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
