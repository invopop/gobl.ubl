package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTotals(t *testing.T) {
	t.Run("peppol-1-advance.json", func(t *testing.T) {
		doc := testInvoiceFrom(t, "peppol/peppol-1-advance.json")

		assert.Equal(t, "1620.00", doc.LegalMonetaryTotal.LineExtensionAmount.Value)
		assert.Equal(t, "1620.00", doc.LegalMonetaryTotal.TaxExclusiveAmount.Value)
		assert.Equal(t, "1960.20", doc.LegalMonetaryTotal.TaxInclusiveAmount.Value)
		assert.NotNil(t, doc.LegalMonetaryTotal.PrepaidAmount)
		assert.Equal(t, "196.02", doc.LegalMonetaryTotal.PrepaidAmount.Value)
		assert.NotNil(t, doc.LegalMonetaryTotal.PayableAmount)
		assert.Equal(t, "1764.18", doc.LegalMonetaryTotal.PayableAmount.Value)

		assert.Equal(t, "340.20", doc.TaxTotal[0].TaxAmount.Value)
		assert.Equal(t, "VAT", doc.TaxTotal[0].TaxSubtotal[0].TaxCategory.TaxScheme.ID.Value)
		assert.Equal(t, "21.0", *doc.TaxTotal[0].TaxSubtotal[0].TaxCategory.Percent)
	})

	t.Run("standard_invoice_no_exemption_reason", func(t *testing.T) {
		doc := testInvoiceFrom(t, "peppol/invoice-minimal.json")

		require.Len(t, doc.TaxTotal, 1)
		require.Len(t, doc.TaxTotal[0].TaxSubtotal, 1)
		tc := doc.TaxTotal[0].TaxSubtotal[0].TaxCategory
		assert.Nil(t, tc.TaxExemptionReasonCode)
		assert.Nil(t, tc.TaxExemptionReason)
	})

	t.Run("reverse_charge_exemption_from_tax_notes", func(t *testing.T) {
		doc := testInvoiceFrom(t, "peppol/peppol-reverse-charge.json")

		require.Len(t, doc.TaxTotal, 1)
		require.Len(t, doc.TaxTotal[0].TaxSubtotal, 1)
		tc := doc.TaxTotal[0].TaxSubtotal[0].TaxCategory

		assert.Equal(t, "AE", tc.ID.Value)
		assert.Equal(t, "0", *tc.Percent)
		require.NotNil(t, tc.TaxExemptionReasonCode)
		assert.Equal(t, "VATEX-EU-AE", *tc.TaxExemptionReasonCode)
		require.NotNil(t, tc.TaxExemptionReason)
		assert.Equal(t, "Reverse Charge / Umkehr der Steuerschuld.", *tc.TaxExemptionReason)
	})
}

func TestOIOUBL21DualCurrencyTotals(t *testing.T) {
	doc := testInvoiceFrom(t, "oioubl21/invoice-minimal.json")

	// OIOUBL carries the accounting-currency tax inside the single TaxTotal,
	// not as a second TaxTotal block (F-INV018 / F-CRN013).
	require.Len(t, doc.TaxTotal, 1)
	require.NotNil(t, doc.TaxTotal[0].TaxAmount.CurrencyID)
	assert.Equal(t, "EUR", *doc.TaxTotal[0].TaxAmount.CurrencyID)

	require.Len(t, doc.TaxTotal[0].TaxSubtotal, 1)
	tcta := doc.TaxTotal[0].TaxSubtotal[0].TransactionCurrencyTaxAmount
	require.NotNil(t, tcta)
	assert.Equal(t, "2551.32", tcta.Value)
	require.NotNil(t, tcta.CurrencyID)
	// F-INV339: the amount's currencyID equals the TaxCurrencyCode (DKK).
	assert.Equal(t, "DKK", *tcta.CurrencyID)

	// ProfileID carries OIOUBL scheme attributes natively (no post-serialize hack).
	require.NotNil(t, doc.ProfileID)
	require.NotNil(t, doc.ProfileID.SchemeID)
	assert.Equal(t, "urn:oioubl:id:profileid-1.2", *doc.ProfileID.SchemeID)
}

// TestOIOUBL21MixedCategoryExchangeRate locks the exchange-rate reconstruction
// for a foreign-currency invoice that mixes StandardRated and ZeroRated lines.
// OIOUBL emits TransactionCurrencyTaxAmount only on StandardRated subtotals
// (F-LIB373), so the parse numerator covers only standard-rated tax; the
// denominator is the document tax amount, to which zero-rated lines contribute
// nothing (zero tax). The reconstructed rate must therefore equal the
// pure-standard-rated rate and be unaffected by the zero-rated line.
func TestOIOUBL21MixedCategoryExchangeRate(t *testing.T) {
	doc, err := ubl.Parse([]byte(mixedCategoryDualCurrencyXML))
	require.NoError(t, err)
	inv, ok := doc.(*ubl.Invoice)
	require.True(t, ok)

	env, err := inv.Convert()
	require.NoError(t, err)
	gi, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	require.Len(t, gi.ExchangeRates, 1)
	rate := gi.ExchangeRates[0]
	assert.Equal(t, "EUR", string(rate.From))
	assert.Equal(t, "DKK", string(rate.To))
	// 2551.32 DKK (standard-rated tax) / 342.00 EUR (document tax) = 7.46.
	assert.Equal(t, "7.46", rate.Amount.String())
}

func TestParseTaxNotes(t *testing.T) {
	t.Run("reverse_charge", func(t *testing.T) {
		env := parseXMLInvoice(t, "peppol/nbio-stuck-ubl.xml")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.NotNil(t, inv.Tax)
		require.Len(t, inv.Tax.Notes, 1)

		note := inv.Tax.Notes[0]
		assert.Equal(t, cbc.Code("VAT"), note.Category)
		assert.Equal(t, cbc.Key("reverse-charge"), note.Key)
		assert.Equal(t, "Reverse charge Article 20", note.Text)
		assert.Equal(t, cbc.Code("AE"), note.Ext.Get(untdid.ExtKeyTaxCategory))
	})

	t.Run("standard_no_tax_notes", func(t *testing.T) {
		env := parseXMLInvoice(t, "peppol/base-example.xml")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		if inv.Tax != nil {
			assert.Empty(t, inv.Tax.Notes)
		}
	})
}

// mixedCategoryDualCurrencyXML is a foreign-currency (EUR document, DKK tax)
// OIOUBL invoice mixing a StandardRated line (carrying TransactionCurrencyTaxAmount)
// with a ZeroRated line (which does not). Used to lock the exchange-rate
// reconstruction against the mixed-category case.
const mixedCategoryDualCurrencyXML = `<?xml version="1.0" encoding="UTF-8"?>
<Invoice xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2" xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2">
  <cbc:UBLVersionID>2.1</cbc:UBLVersionID>
  <cbc:CustomizationID>OIOUBL-2.1</cbc:CustomizationID>
  <cbc:ProfileID schemeAgencyID="320" schemeID="urn:oioubl:id:profileid-1.4">urn:www.nesubl.eu:profiles:profile5:ver2.0</cbc:ProfileID>
  <cbc:ID>SAMPLE-001</cbc:ID>
  <cbc:IssueDate>2024-05-15</cbc:IssueDate>
  <cbc:InvoiceTypeCode listAgencyID="320" listID="urn:oioubl:codelist:invoicetypecode-1.1">380</cbc:InvoiceTypeCode>
  <cbc:DocumentCurrencyCode>EUR</cbc:DocumentCurrencyCode>
  <cbc:TaxCurrencyCode>DKK</cbc:TaxCurrencyCode>
  <cac:AccountingSupplierParty>
    <cac:Party>
      <cbc:EndpointID schemeID="GLN">5790000436101</cbc:EndpointID>
      <cac:PartyName>
        <cbc:Name>Provide One GmbH</cbc:Name>
      </cac:PartyName>
      <cac:PostalAddress>
        <cbc:StreetName>Dietmar-Hopp-Allee 16</cbc:StreetName>
        <cbc:CityName>Walldorf</cbc:CityName>
        <cbc:PostalZone>69190</cbc:PostalZone>
        <cac:Country>
          <cbc:IdentificationCode>DK</cbc:IdentificationCode>
        </cac:Country>
      </cac:PostalAddress>
      <cac:PartyLegalEntity>
        <cbc:RegistrationName>Provide One GmbH</cbc:RegistrationName>
        <cbc:CompanyID schemeID="DK:CVR">DK37990485</cbc:CompanyID>
      </cac:PartyLegalEntity>
    </cac:Party>
  </cac:AccountingSupplierParty>
  <cac:AccountingCustomerParty>
    <cac:Party>
      <cbc:EndpointID schemeID="GLN">5790000436057</cbc:EndpointID>
      <cac:PartyName>
        <cbc:Name>Sample Consumer</cbc:Name>
      </cac:PartyName>
      <cac:PostalAddress>
        <cbc:StreetName>Werner-Heisenberg-Allee 25</cbc:StreetName>
        <cbc:CityName>München</cbc:CityName>
        <cbc:PostalZone>80939</cbc:PostalZone>
        <cac:Country>
          <cbc:IdentificationCode>DK</cbc:IdentificationCode>
        </cac:Country>
      </cac:PostalAddress>
      <cac:PartyLegalEntity>
        <cbc:RegistrationName>Sample Consumer</cbc:RegistrationName>
        <cbc:CompanyID schemeID="DK:CVR">DK47458714</cbc:CompanyID>
      </cac:PartyLegalEntity>
    </cac:Party>
  </cac:AccountingCustomerParty>
  <cac:TaxTotal>
    <cbc:TaxAmount currencyID="EUR">342.00</cbc:TaxAmount>
    <cac:TaxSubtotal>
      <cbc:TaxableAmount currencyID="EUR">1800.00</cbc:TaxableAmount>
      <cbc:TaxAmount currencyID="EUR">342.00</cbc:TaxAmount>
      <cbc:TransactionCurrencyTaxAmount currencyID="DKK">2551.32</cbc:TransactionCurrencyTaxAmount>
      <cac:TaxCategory>
        <cbc:ID schemeAgencyID="320" schemeID="urn:oioubl:id:taxcategoryid-1.1">StandardRated</cbc:ID>
        <cbc:Percent>19</cbc:Percent>
        <cac:TaxScheme>
          <cbc:ID schemeAgencyID="320" schemeID="urn:oioubl:id:taxschemeid-1.2">63</cbc:ID>
          <cbc:Name>Moms</cbc:Name>
        </cac:TaxScheme>
      </cac:TaxCategory>
    </cac:TaxSubtotal>
    <cac:TaxSubtotal>
      <cbc:TaxableAmount currencyID="EUR">1000.00</cbc:TaxableAmount>
      <cbc:TaxAmount currencyID="EUR">0.00</cbc:TaxAmount>
      <cac:TaxCategory>
        <cbc:ID schemeAgencyID="320" schemeID="urn:oioubl:id:taxcategoryid-1.1">ZeroRated</cbc:ID>
        <cbc:Percent>0</cbc:Percent>
        <cac:TaxScheme>
          <cbc:ID schemeAgencyID="320" schemeID="urn:oioubl:id:taxschemeid-1.2">63</cbc:ID>
          <cbc:Name>Moms</cbc:Name>
        </cac:TaxScheme>
      </cac:TaxCategory>
    </cac:TaxSubtotal>
  </cac:TaxTotal>
  <cac:LegalMonetaryTotal>
    <cbc:LineExtensionAmount currencyID="EUR">2800.00</cbc:LineExtensionAmount>
    <cbc:TaxExclusiveAmount currencyID="EUR">342.00</cbc:TaxExclusiveAmount>
    <cbc:TaxInclusiveAmount currencyID="EUR">3142.00</cbc:TaxInclusiveAmount>
    <cbc:PayableAmount currencyID="EUR">3142.00</cbc:PayableAmount>
  </cac:LegalMonetaryTotal>
  <cac:InvoiceLine>
    <cbc:ID>1</cbc:ID>
    <cbc:InvoicedQuantity unitCode="HUR">20</cbc:InvoicedQuantity>
    <cbc:LineExtensionAmount currencyID="EUR">1800.00</cbc:LineExtensionAmount>
    <cac:Item>
      <cbc:Name>Development services</cbc:Name>
      <cac:ClassifiedTaxCategory>
        <cbc:ID schemeAgencyID="320" schemeID="urn:oioubl:id:taxcategoryid-1.1">StandardRated</cbc:ID>
        <cbc:Percent>19</cbc:Percent>
        <cac:TaxScheme>
          <cbc:ID schemeAgencyID="320" schemeID="urn:oioubl:id:taxschemeid-1.2">63</cbc:ID>
          <cbc:Name>Moms</cbc:Name>
        </cac:TaxScheme>
      </cac:ClassifiedTaxCategory>
    </cac:Item>
    <cac:Price>
      <cbc:PriceAmount currencyID="EUR">90.00</cbc:PriceAmount>
    </cac:Price>
  </cac:InvoiceLine>
  <cac:InvoiceLine>
    <cbc:ID>2</cbc:ID>
    <cbc:InvoicedQuantity unitCode="HUR">10</cbc:InvoicedQuantity>
    <cbc:LineExtensionAmount currencyID="EUR">1000.00</cbc:LineExtensionAmount>
    <cac:Item>
      <cbc:Name>Exported goods</cbc:Name>
      <cac:ClassifiedTaxCategory>
        <cbc:ID schemeAgencyID="320" schemeID="urn:oioubl:id:taxcategoryid-1.1">ZeroRated</cbc:ID>
        <cbc:Percent>0</cbc:Percent>
        <cac:TaxScheme>
          <cbc:ID schemeAgencyID="320" schemeID="urn:oioubl:id:taxschemeid-1.2">63</cbc:ID>
          <cbc:Name>Moms</cbc:Name>
        </cac:TaxScheme>
      </cac:ClassifiedTaxCategory>
    </cac:Item>
    <cac:Price>
      <cbc:PriceAmount currencyID="EUR">100.00</cbc:PriceAmount>
    </cac:Price>
  </cac:InvoiceLine>
</Invoice>`
