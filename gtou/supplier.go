package gtou

import (
	"github.com/invopop/gobl/bill"
)

// createSupplierParty creates the SupplierParty part of a UBL invoice
func (c *Conversor) createSupplierParty(inv *bill.Invoice) error {
	if inv.Supplier == nil {
		return nil
	}

	supplier := inv.Supplier
	c.doc.AccountingSupplierParty = SupplierParty{
		Party: Party{
			PartyIdentification: createPartyIdentification(supplier),
			PartyName:           createPartyName(supplier),
			PostalAddress:       createPostalAddress(supplier),
			PartyTaxScheme:      createPartyTaxScheme(supplier),
			PartyLegalEntity:    createPartyLegalEntity(supplier),
			Contact:             createContact(supplier),
		},
	}

	return nil
}

func createPartyIdentification(supplier *bill.Supplier) []Identification {
	return []Identification{
		{ID: supplier.ID},
	}
}

func createPartyName(supplier *bill.Supplier) *PartyName {
	return &PartyName{Name: supplier.Name}
}

func createPostalAddress(supplier *bill.Supplier) *PostalAddress {
	return &PostalAddress{
		StreetName: supplier.Address.Street,
		CityName:   supplier.Address.City,
		PostalZone: supplier.Address.PostalCode,
		Country:    &Country{IdentificationCode: supplier.Address.CountryCode},
	}
}

func createPartyTaxScheme(supplier *bill.Supplier) []PartyTaxScheme {
	return []PartyTaxScheme{
		{CompanyID: supplier.TaxID},
	}
}

func createPartyLegalEntity(supplier *bill.Supplier) *PartyLegalEntity {
	return &PartyLegalEntity{
		RegistrationName: supplier.LegalName,
		CompanyID:        supplier.CompanyID,
	}
}

func createContact(supplier *bill.Supplier) *Contact {
	return &Contact{
		Name:           supplier.ContactName,
		Telephone:      supplier.ContactPhone,
		ElectronicMail: supplier.ContactEmail,
	}
}
