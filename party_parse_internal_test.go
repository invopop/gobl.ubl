package ubl

import (
	"fmt"
	"testing"

	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/xmlctx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoblPartyLegalForm covers BT-33 on the way in: the legal form becomes
// the party's registration, but only when there is something to keep.
func TestGoblPartyLegalForm(t *testing.T) {
	parse := func(t *testing.T, legalForm string) *org.Party {
		t.Helper()
		in := `<cac:Party xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">
			<cac:PartyLegalEntity>
				<cbc:RegistrationName>Tvrtka A d.o.o.</cbc:RegistrationName>
				` + legalForm + `
			</cac:PartyLegalEntity>
		</cac:Party>`
		party := new(Party)
		require.NoError(t, xmlctx.Unmarshal([]byte(in), party, xmlctx.WithNamespaces(map[string]string{
			"cac": NamespaceCAC,
			"cbc": NamespaceCBC,
		})))
		return goblParty(party)
	}

	t.Run("sets the registration", func(t *testing.T) {
		p := parse(t, `<cbc:CompanyLegalForm>Osnovano na trgovačkom sudu, temeljni kapital</cbc:CompanyLegalForm>`)

		require.NotNil(t, p.Registration)
		assert.Equal(t, "Osnovano na trgovačkom sudu, temeljni kapital", p.Registration.Other)
	})

	for name, legalForm := range map[string]string{
		"absent":             ``,
		"empty":              `<cbc:CompanyLegalForm/>`,
		"whitespace":         `<cbc:CompanyLegalForm> </cbc:CompanyLegalForm>`,
		"replacement runes":  "<cbc:CompanyLegalForm>\uFFFD\uFFFD</cbc:CompanyLegalForm>",
		"padded replacement": "<cbc:CompanyLegalForm> \uFFFD </cbc:CompanyLegalForm>",
	} {
		t.Run("no registration when "+name, func(t *testing.T) {
			p := parse(t, legalForm)

			assert.Nil(t, p.Registration)
		})
	}

	t.Run("no registration when blank in a party built in code", func(t *testing.T) {
		// Unlike the XML decoder, a caller of ParseParty may pass untrimmed text.
		form := " \uFFFD "
		p := goblParty(&Party{PartyLegalEntity: &PartyLegalEntity{CompanyLegalForm: &form}})

		assert.Nil(t, p.Registration)
	})
}

// TestGoblPartyTaxRegistration covers BT-32 on the way in: a non-VAT scheme
// is a tax registration identifier, never the party's VAT identifier.
func TestGoblPartyTaxRegistration(t *testing.T) {
	const partyTemplate = `<cac:Party xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">
		<cac:PostalAddress><cac:Country><cbc:IdentificationCode>FR</cbc:IdentificationCode></cac:Country></cac:PostalAddress>
		%s
	</cac:Party>`

	const registration = `<cac:PartyTaxScheme>
			<cbc:CompanyID>828701557</cbc:CompanyID>
			<cac:TaxScheme><cbc:ID>FC</cbc:ID></cac:TaxScheme>
		</cac:PartyTaxScheme>`

	const vat = `<cac:PartyTaxScheme>
			<cbc:CompanyID>FR18828701557</cbc:CompanyID>
			<cac:TaxScheme><cbc:ID>VAT</cbc:ID></cac:TaxScheme>
		</cac:PartyTaxScheme>`

	parse := func(t *testing.T, in string) *org.Party {
		t.Helper()
		party := new(Party)
		require.NoError(t, xmlctx.Unmarshal([]byte(in), party, xmlctx.WithNamespaces(map[string]string{
			"cac": NamespaceCAC,
			"cbc": NamespaceCBC,
		})))
		return goblParty(party)
	}

	t.Run("alongside a VAT identifier", func(t *testing.T) {
		p := parse(t, fmt.Sprintf(partyTemplate, registration+vat))

		require.NotNil(t, p.TaxID)
		assert.Equal(t, cbc.Code("FR18828701557"), p.TaxID.Code)
		require.Len(t, p.Identities, 1)
		assert.Equal(t, org.IdentityScopeTax, p.Identities[0].Scope)
		assert.Equal(t, cbc.Code("828701557"), p.Identities[0].Code)
		assert.Equal(t, l10n.ISOCountryCode("FR"), p.Identities[0].Country)
		assert.Empty(t, p.Identities[0].Type)
	})

	t.Run("without a VAT identifier", func(t *testing.T) {
		p := parse(t, fmt.Sprintf(partyTemplate, registration))

		assert.Nil(t, p.TaxID)
		require.Len(t, p.Identities, 1)
		assert.Equal(t, org.IdentityScopeTax, p.Identities[0].Scope)
		assert.Equal(t, cbc.Code("828701557"), p.Identities[0].Code)
	})

	t.Run("a scheme the country does not use is a registration", func(t *testing.T) {
		p := parse(t, fmt.Sprintf(partyTemplate, `<cac:PartyTaxScheme>
			<cbc:CompanyID>Godkänd för F-skatt</cbc:CompanyID>
			<cac:TaxScheme><cbc:ID>LOC</cbc:ID></cac:TaxScheme>
		</cac:PartyTaxScheme>`))

		assert.Nil(t, p.TaxID)
		require.Len(t, p.Identities, 1)
		assert.Equal(t, org.IdentityScopeTax, p.Identities[0].Scope)
	})

	t.Run("only VAT sets the tax ID", func(t *testing.T) {
		// GST is Singapore's own tax scheme, but only VAT is BT-31.
		p := parse(t, `<cac:Party xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">
			<cac:PostalAddress><cac:Country><cbc:IdentificationCode>SG</cbc:IdentificationCode></cac:Country></cac:PostalAddress>
			<cac:PartyTaxScheme>
				<cbc:CompanyID>123457890SC</cbc:CompanyID>
				<cac:TaxScheme><cbc:ID>GST</cbc:ID></cac:TaxScheme>
			</cac:PartyTaxScheme>
		</cac:Party>`)

		assert.Nil(t, p.TaxID)
		require.Len(t, p.Identities, 1)
		assert.Equal(t, org.IdentityScopeTax, p.Identities[0].Scope)
		assert.Equal(t, cbc.Code("123457890SC"), p.Identities[0].Code)
	})
}

// TestGoblPartyVATCountry covers BT-31 on a party with no postal address, such
// as the payee (UBL-CR-272 forbids its address): the tax ID country can only
// come from the ISO prefix BR-CO-09 requires on the VAT identifier.
func TestGoblPartyVATCountry(t *testing.T) {
	const partyTemplate = `<cac:Party xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">
		<cac:PartyTaxScheme>
			<cbc:CompanyID>%s</cbc:CompanyID>
			<cac:TaxScheme><cbc:ID>VAT</cbc:ID></cac:TaxScheme>
		</cac:PartyTaxScheme>
	</cac:Party>`

	parse := func(t *testing.T, code string) *org.Party {
		t.Helper()
		party := new(Party)
		require.NoError(t, xmlctx.Unmarshal(fmt.Appendf(nil, partyTemplate, code), party, xmlctx.WithNamespaces(map[string]string{
			"cac": NamespaceCAC,
			"cbc": NamespaceCBC,
		})))
		return goblParty(party)
	}

	tests := []struct {
		name    string
		code    string
		country l10n.TaxCountryCode
	}{
		{"takes the VAT prefix", "FR53341200068", "FR"},
		{"takes a tax-only prefix", "EL094259216", "EL"},
		{"takes a lower-case prefix", "fr53341200068", "FR"},
		// BR-CO-09 makes the prefix mandatory, so a code without a usable one
		// keeps an empty country and still fails validation.
		{"no prefix", "123456789", ""},
		{"an unknown prefix", "ZZ123456789", ""},
		{"only a prefix", "FR", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parse(t, tt.code)

			require.NotNil(t, p.TaxID)
			assert.Equal(t, tt.country, p.TaxID.Country)
			assert.Equal(t, cbc.Code(tt.code), p.TaxID.Code)
			assert.Empty(t, p.Identities)
		})
	}
}
