package utog

import (
	"github.com/invopop/gobl.ubl/document"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

func (c *Converter) getParty(party *document.Party) *org.Party {
	p := &org.Party{}

	if party.PartyLegalEntity != nil && party.PartyLegalEntity.RegistrationName != nil {
		p.Name = *party.PartyLegalEntity.RegistrationName
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

	if party.PartyTaxScheme != nil {
		for _, taxReg := range party.PartyTaxScheme {
			if taxReg.CompanyID != nil {
				switch taxReg.TaxScheme.ID {
				// Source https://ec.europa.eu/digital-building-blocks/sites/download/attachments/467108974/EN16931%20code%20lists%20values%20v13%20-%20used%20from%202024-05-15.xlsx?version=2&modificationDate=1712937109681&api=v2
				case "VAT":
					// Parse the country code from the vat
					if identity, err := tax.ParseIdentity(*taxReg.CompanyID); err == nil {
						p.TaxID = identity
					} else {
						// Fallback to preserve the tax id
						p.TaxID = &tax.Identity{
							Country: l10n.TaxCountryCode(party.PostalAddress.Country.IdentificationCode),
							Code:    cbc.Code(*taxReg.CompanyID),
						}
					}
				default:
					id := &org.Identity{
						Country: l10n.ISOCountryCode(party.PostalAddress.Country.IdentificationCode),
						Code:    cbc.Code(*taxReg.CompanyID),
					}
					if p.Identities == nil {
						p.Identities = make([]*org.Identity, 0)
					}
					p.Identities = append(p.Identities, id)
				}
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

func parseAddress(address *document.PostalAddress) *org.Address {
	if address == nil {
		return nil
	}

	addr := &org.Address{
		Country: l10n.ISOCountryCode(address.Country.IdentificationCode),
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
