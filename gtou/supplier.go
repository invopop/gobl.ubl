package gtou

import (
	"fmt"

	"github.com/invopop/gobl/org"
)

func (c *Conversor) newSupplier(supplier *org.Party) error {
	if supplier == nil {
		return nil
	}

	supplierParty := SupplierParty{
		Party: Party{
			// PartyIdentification:
			PartyName: &PartyName{
				Name: supplier.Name,
			},
			PostalAddress:    newAddress(supplier.Addresses),
			PartyTaxScheme:   createPartyTaxScheme(supplier),
			PartyLegalEntity: createPartyLegalEntity(supplier),
			Contact:          createContact(supplier),
		},
	}
	if supplier.TaxID != nil {
		supplierParty.Party.PartyTaxScheme = []PartyTaxScheme{
			{
				CompanyID: supplier.TaxID.String(),
			},
		}
	}
	if supplier.Name != "" {
		supplierParty.Party.PartyLegalEntity = &PartyLegalEntity{
			RegistrationName: supplier.Name,
		}
	}
	c.doc.AccountingSupplierParty = supplierParty

	return nil
}

func (c *Conversor) createPartyName(supplier *org.Party) {
	c.doc.AccountingSupplierParty.Party.PartyName = &PartyName{
		Name: supplier.Name,
	}
}

func (c *Conversor) createPartyTaxScheme(supplier *org.Party) {
	c.doc.AccountingSupplierParty.Party.PartyTaxScheme = []PartyTaxScheme{
		{
			CompanyID: supplier.TaxID.String(),
		},
	}
}

func (c *Conversor) createPartyLegalEntity(supplier *org.Party) {
	c.doc.AccountingSupplierParty.Party.PartyLegalEntity = &PartyLegalEntity{
		RegistrationName: supplier.Name,
	}

}

func (c *Conversor) createContact(supplier *org.Party) {
	c.doc.AccountingSupplierParty.Party.Contact = &Contact{
		Name:           contactName(supplier.People[0].Name),
		Telephone:      supplier.Telephones[0].Number,
		ElectronicMail: supplier.Emails[0].Address,
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
