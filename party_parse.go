package ubl

import (
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
		p.Name = CleanString(*party.PartyLegalEntity.RegistrationName)
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
			p.Name = CleanString(party.PartyName.Name)
		} else if party.PartyName.Name != p.Name {
			// Only set alias if it's different from the name
			p.Alias = CleanString(party.PartyName.Name)
		}
	}

	if party.Contact != nil && party.Contact.Name != nil {
		p.People = []*org.Person{
			{
				Name: &org.Name{
					Given: CleanString(*party.Contact.Name),
				},
			},
		}
	}

	if party.PostalAddress != nil {
		p.Addresses = []*org.Address{
			ParseAddress(party.PostalAddress),
		}
	}

	if party.Contact != nil {
		if party.Contact.Telephone != nil {
			p.Telephones = []*org.Telephone{
				{
					Number: CleanString(*party.Contact.Telephone),
				},
			}
		}
		if party.Contact.ElectronicMail != nil {
			p.Emails = []*org.Email{
				{
					Address: CleanString(*party.Contact.ElectronicMail),
				},
			}
		}
	}

	HandleLegalEntityIdentity(party, p)
	HandlePartyTaxSchemes(party, p, party.CountryCode(), nil)
	handlePartyIdentifications(party, p, o)

	return p
}

// GoblDeliveryParty creates a GOBL party with only the BTs available
// for the delivery party (BT-70 name). Address is handled separately
// via DeliveryLocation.
func GoblDeliveryParty(party *Party) *org.Party {
	if party == nil {
		return nil
	}
	p := &org.Party{}

	if party.PartyLegalEntity != nil && party.PartyLegalEntity.RegistrationName != nil {
		p.Name = CleanString(*party.PartyLegalEntity.RegistrationName)
	}
	if party.PartyName != nil {
		if p.Name == "" {
			p.Name = CleanString(party.PartyName.Name)
		}
	}

	if p.Name == "" {
		return nil
	}
	return p
}

// ParseAddress builds a GOBL org.Address from a UBL PostalAddress.
func ParseAddress(address *PostalAddress) *org.Address {
	if address == nil {
		return nil
	}

	addr := new(org.Address)
	if address.Country != nil {
		addr.Country = l10n.ISOCountryCode(address.Country.IdentificationCode)
	}
	if address.StreetName != nil {
		addr.Street = CleanString(*address.StreetName)
	}
	if address.AdditionalStreetName != nil {
		addr.StreetExtra = CleanString(*address.AdditionalStreetName)
	}
	if address.CityName != nil {
		addr.Locality = CleanString(*address.CityName)
	}
	if address.PostalZone != nil {
		addr.Code = cbc.Code(CleanString(*address.PostalZone))
	}
	if address.CountrySubentity != nil {
		addr.Region = CleanString(*address.CountrySubentity)
	}
	if address.BuildingNumber != nil {
		addr.Number = CleanString(*address.BuildingNumber)
	}
	// CitySubdivisionName is used by ZATCA to represent the district,
	// which maps to StreetExtra in GOBL.
	if address.CitySubdivisionName != nil && addr.StreetExtra == "" {
		addr.StreetExtra = CleanString(*address.CitySubdivisionName)
	}
	return addr
}

// HandleLegalEntityIdentity restores a party's PartyLegalEntity/CompanyID as a legal-scope GOBL identity.
func HandleLegalEntityIdentity(party *Party, p *org.Party) {
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

// HandlePartyTaxSchemes restores a party's tax ID from its PartyTaxScheme
// entries; mapScheme translates wire scheme IDs (nil is identity).
func HandlePartyTaxSchemes(party *Party, p *org.Party, countryCode string, mapScheme func(string) cbc.Code) {
	if len(party.PartyTaxScheme) == 0 {
		return
	}

	validSchemes := ExtractValidTaxSchemes(party.PartyTaxScheme)

	if len(validSchemes) == 1 {
		SetTaxIDFromScheme(validSchemes[0], p, countryCode, mapScheme)
	} else if len(validSchemes) > 1 {
		HandleMultipleTaxSchemes(validSchemes, p, countryCode, mapScheme)
	}
}

// ExtractValidTaxSchemes filters PartyTaxScheme entries down to those
// carrying both a company ID and a tax scheme.
func ExtractValidTaxSchemes(schemes []PartyTaxScheme) []PartyTaxScheme {
	validSchemes := make([]PartyTaxScheme, 0)
	for _, pts := range schemes {
		if pts.CompanyID != nil && pts.CompanyID.Value != "" && pts.TaxScheme != nil {
			validSchemes = append(validSchemes, pts)
		}
	}
	return validSchemes
}

// SetTaxIDFromScheme builds a GOBL tax identity from a PartyTaxScheme; see
// HandlePartyTaxSchemes for mapScheme.
func SetTaxIDFromScheme(pts PartyTaxScheme, p *org.Party, countryCode string, mapScheme func(string) cbc.Code) {
	p.TaxID = &tax.Identity{
		Country: l10n.TaxCountryCode(countryCode),
		Code:    cbc.Code(pts.CompanyID.Value),
	}
	sc := cbc.Code(pts.TaxScheme.ID.Value)
	if mapScheme != nil {
		sc = mapScheme(pts.TaxScheme.ID.Value)
	}
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

// HandleMultipleTaxSchemes picks the VAT scheme as the party's tax ID; the
// rest become tax-scoped identities.
func HandleMultipleTaxSchemes(validSchemes []PartyTaxScheme, p *org.Party, countryCode string, mapScheme func(string) cbc.Code) {
	// Multiple tax schemes: look for VAT, otherwise use first
	vatIdx := FindVATSchemeIndex(validSchemes, mapScheme)

	// Use VAT if found, otherwise first one
	taxIDIdx := 0
	if vatIdx != -1 {
		taxIDIdx = vatIdx
	}

	// Set TaxID from chosen scheme
	SetTaxIDFromScheme(validSchemes[taxIDIdx], p, countryCode, mapScheme)

	// Rest become identities with tax scope
	AddRemainingTaxSchemesAsIdentities(validSchemes, taxIDIdx, p, countryCode, mapScheme)
}

// FindVATSchemeIndex returns the index of the first VAT scheme among
// schemes, or -1; see HandlePartyTaxSchemes for mapScheme.
func FindVATSchemeIndex(schemes []PartyTaxScheme, mapScheme func(string) cbc.Code) int {
	for i, pts := range schemes {
		sc := cbc.Code(pts.TaxScheme.ID.Value)
		if mapScheme != nil {
			sc = mapScheme(pts.TaxScheme.ID.Value)
		}
		if sc == TaxSchemeVAT {
			return i
		}
	}
	return -1
}

// AddRemainingTaxSchemesAsIdentities appends every validSchemes entry other
// than taxIDIdx to p.Identities as a tax-scoped identity; see
// HandlePartyTaxSchemes for mapScheme.
func AddRemainingTaxSchemesAsIdentities(validSchemes []PartyTaxScheme, taxIDIdx int, p *org.Party, countryCode string, mapScheme func(string) cbc.Code) {
	for i, pts := range validSchemes {
		if i == taxIDIdx {
			continue
		}

		typ := cbc.Code(pts.TaxScheme.ID.Value)
		if mapScheme != nil {
			typ = mapScheme(pts.TaxScheme.ID.Value)
		}
		identity := &org.Identity{
			Country: l10n.ISOCountryCode(countryCode),
			Code:    cbc.Code(pts.CompanyID.Value),
			Scope:   org.IdentityScopeTax,
			Type:    typ,
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
