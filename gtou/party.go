package gtou

import (
	"fmt"

	"github.com/invopop/gobl/org"
)

func (c *Conversor) newParty(party *org.Party) Party {
	if party == nil {
		return Party{}
	}

	p := Party{
		// PartyIdentification:
		PartyName: &PartyName{
			Name: party.Name,
		},
		PostalAddress: newAddress(party.Addresses),
		PartyTaxScheme: []PartyTaxScheme{
			{
				CompanyID: nil,
				TaxScheme: &TaxScheme{
					ID: party.TaxID.Code.String(),
				},
			},
		},
		Contact: createContact(party),
	}

	if party.TaxID != nil {
		p.PartyTaxScheme = []PartyTaxScheme{
			{
				CompanyID: party.TaxID.String(),
			},
		}
	}

	if len(party.Emails) > 0 {
		p.Contact = &Contact{
			ElectronicMail: party.Emails[0].Address,
		}
	}

	if party.Name != "" {
		p.Party.PartyLegalEntity = &PartyLegalEntity{
			RegistrationName: party.Name,
		}
	}
	if party.LegalEntity != nil {
		p.Party.PartyLegalEntity = &PartyLegalEntity{
			RegistrationName:      party.LegalEntity.RegistrationName,
			CompanyID:             party.LegalEntity.CompanyID,
			CompanyType:           party.LegalEntity.CompanyType,
			CorporateRegistration: party.LegalEntity.CorporateRegistration,
		}
	}
	return p
}

func (c *Conversor) createPartyName(party *org.Party) {
	c.doc.AccountingSupplierParty.Party.PartyName = &PartyName{
		Name: party.Name,
	}
}

func newAddress(addresses []*org.Address) *PostalAddress {
	if len(addresses) == 0 {
		return nil
	}
	// Only return the first address
	address := addresses[0]

	postalTradeAddress := &PostalAddress{
		StreetName:           address.Street,
		AdditionalStreetName: address.StreetExtra,
		CityName:             address.Locality,
		PostalZone:           address.Code,
		CountrySubentity:     address.Region,
		Country:              &Country{IdentificationCode: string(address.Country)},
	}

	return postalTradeAddress
}

func contactName(personName *org.Name) string {
	given := personName.Given
	surname := personName.Surname

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
