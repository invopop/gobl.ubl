package utog

import (
	"regexp"

	"github.com/invopop/gobl.ubl/structs"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

var onlyDigitsRegex = regexp.MustCompile(`\D`)

// Parses the XML information for a Party object
func ParseUtoGParty(party *structs.Party) *org.Party {
	p := &org.Party{
		Name: party.PartyName.Name,
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

	if party.PartyTaxScheme != nil {
		for _, taxReg := range *party.PartyTaxScheme {
			if taxReg.CompanyID != nil {
				switch *taxReg.TaxScheme.ID {
				//Source https://ec.europa.eu/digital-building-blocks/sites/download/attachments/467108974/EN16931%20code%20lists%20values%20v13%20-%20used%20from%202024-05-15.xlsx?version=2&modificationDate=1712937109681&api=v2
				case "VAT":
					p.TaxID = &tax.Identity{
						Country: l10n.TaxCountryCode(party.PostalAddress.Country.IdentificationCode),
						Code:    cbc.Code(onlyDigitsRegex.ReplaceAllString(*taxReg.CompanyID, "")),
					}
				default:
					identity := &org.Identity{
						Country: l10n.ISOCountryCode(party.PostalAddress.Country.IdentificationCode),
						Code:    cbc.Code(*taxReg.CompanyID),
					}
					p.Identities = append(p.Identities, identity)
				}
			}
		}
	}

	return p
}

func parseAddress(address *structs.PostalAddress) *org.Address {
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
		addr.Code = *address.PostalZone
	}

	if address.CountrySubentity != nil {
		addr.Region = *address.CountrySubentity
	}

	return addr
}
