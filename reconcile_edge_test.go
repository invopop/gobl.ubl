package ubl_test

import (
	"fmt"
	"os"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These cover reconciliation cases the fixture corpus cannot reach, so each
// document is built here rather than stored: a line with no price, a document
// with no BT-112, and totals that disagree only on the optional fields.

const edgeHeader = `<?xml version="1.0" encoding="UTF-8"?>
<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2" xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">
  <cbc:UBLVersionID>2.1</cbc:UBLVersionID>
  <cbc:CustomizationID>urn:cen.eu:en16931:2017</cbc:CustomizationID>
  <cbc:ID>EDGE-1</cbc:ID>
  <cbc:IssueDate>2026-01-01</cbc:IssueDate>
  <cbc:DueDate>2026-02-01</cbc:DueDate>
  <cbc:InvoiceTypeCode>380</cbc:InvoiceTypeCode>
  <cbc:DocumentCurrencyCode>EUR</cbc:DocumentCurrencyCode>
  <cbc:BuyerReference>REF-1</cbc:BuyerReference>
  <cac:AccountingSupplierParty><cac:Party>
    <cbc:EndpointID schemeID="0002">381511039</cbc:EndpointID>
    <cac:PostalAddress><cbc:CityName>Paris</cbc:CityName>
      <cac:Country><cbc:IdentificationCode>FR</cbc:IdentificationCode></cac:Country></cac:PostalAddress>
    <cac:PartyTaxScheme><cbc:CompanyID>FR59381511039</cbc:CompanyID>
      <cac:TaxScheme><cbc:ID>VAT</cbc:ID></cac:TaxScheme></cac:PartyTaxScheme>
    <cac:PartyLegalEntity><cbc:RegistrationName>Seller SA</cbc:RegistrationName></cac:PartyLegalEntity>
  </cac:Party></cac:AccountingSupplierParty>
  <cac:AccountingCustomerParty><cac:Party>
    <cbc:EndpointID schemeID="0002">820731115</cbc:EndpointID>
    <cac:PostalAddress><cbc:CityName>Paris</cbc:CityName>
      <cac:Country><cbc:IdentificationCode>FR</cbc:IdentificationCode></cac:Country></cac:PostalAddress>
    <cac:PartyLegalEntity><cbc:RegistrationName>Buyer SA</cbc:RegistrationName></cac:PartyLegalEntity>
  </cac:Party></cac:AccountingCustomerParty>`

// edgeLine renders an invoice line, omitting cac:Price entirely when price is
// empty so the converter drops it.
func edgeLine(id, qty, amount, price string) string {
	out := fmt.Sprintf(`
  <cac:InvoiceLine>
    <cbc:ID>%s</cbc:ID>
    <cbc:InvoicedQuantity unitCode="EA">%s</cbc:InvoicedQuantity>
    <cbc:LineExtensionAmount currencyID="EUR">%s</cbc:LineExtensionAmount>
    <cac:Item><cbc:Name>Widget %s</cbc:Name>
      <cac:ClassifiedTaxCategory><cbc:ID>S</cbc:ID><cbc:Percent>20.00</cbc:Percent>
        <cac:TaxScheme><cbc:ID>VAT</cbc:ID></cac:TaxScheme></cac:ClassifiedTaxCategory></cac:Item>`, id, qty, amount, id)
	if price != "" {
		out += fmt.Sprintf("\n    <cac:Price><cbc:PriceAmount currencyID=\"EUR\">%s</cbc:PriceAmount></cac:Price>", price)
	}
	return out + "\n  </cac:InvoiceLine>"
}

func edgeInvoice(t *testing.T, taxTotal, monetaryTotal string, lines ...string) *bill.Invoice {
	t.Helper()

	body := edgeHeader + taxTotal + monetaryTotal
	for _, l := range lines {
		body += l
	}
	body += "\n</Invoice>"

	doc, err := ubl.Parse([]byte(body))
	require.NoError(t, err)
	inv, ok := doc.(*ubl.Invoice)
	require.True(t, ok)

	env, err := inv.Convert()
	require.NoError(t, err)
	out, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)
	return out
}

// A line with no price is dropped during conversion. Anything pairing source
// lines with converted ones has to drop it the same way, or every later line
// takes the wrong declared amount.
func TestReconcilePairsLinesPastADroppedLine(t *testing.T) {
	inv := edgeInvoice(t,
		`
  <cac:TaxTotal><cbc:TaxAmount currencyID="EUR">20.00</cbc:TaxAmount>
    <cac:TaxSubtotal><cbc:TaxableAmount currencyID="EUR">100.00</cbc:TaxableAmount>
      <cbc:TaxAmount currencyID="EUR">20.00</cbc:TaxAmount>
      <cac:TaxCategory><cbc:ID>S</cbc:ID><cbc:Percent>20.00</cbc:Percent>
        <cac:TaxScheme><cbc:ID>VAT</cbc:ID></cac:TaxScheme></cac:TaxCategory></cac:TaxSubtotal></cac:TaxTotal>`,
		`
  <cac:LegalMonetaryTotal>
    <cbc:LineExtensionAmount currencyID="EUR">100.00</cbc:LineExtensionAmount>
    <cbc:TaxExclusiveAmount currencyID="EUR">100.00</cbc:TaxExclusiveAmount>
    <cbc:TaxInclusiveAmount currencyID="EUR">120.00</cbc:TaxInclusiveAmount>
    <cbc:PayableAmount currencyID="EUR">120.00</cbc:PayableAmount>
  </cac:LegalMonetaryTotal>`,
		edgeLine("1", "1", "999.00", ""),       // no price: dropped
		edgeLine("2", "10", "100.00", "10.00"), // the only converted line
	)

	// Only the priced line survives, and it must carry its own amount - not
	// the 999.00 belonging to the line that was dropped.
	require.Len(t, inv.Lines, 1)
	assert.Equal(t, "10.00", inv.Lines[0].Item.Price.String())
	assert.Equal(t, "100.00", inv.Lines[0].Total.String())
	assert.False(t, inv.HasTags(tax.TagBypass), "the surviving line reconciles")
}

// BT-112 is mandatory but not every document carries one. Without it the
// calculated total must not survive beside declared components it no longer
// agrees with.
func TestReconcileDerivesMissingTaxInclusiveTotal(t *testing.T) {
	inv := edgeInvoice(t,
		`
  <cac:TaxTotal><cbc:TaxAmount currencyID="EUR">17.50</cbc:TaxAmount>
    <cac:TaxSubtotal><cbc:TaxableAmount currencyID="EUR">90.00</cbc:TaxableAmount>
      <cbc:TaxAmount currencyID="EUR">17.50</cbc:TaxAmount>
      <cac:TaxCategory><cbc:ID>S</cbc:ID><cbc:Percent>20.00</cbc:Percent>
        <cac:TaxScheme><cbc:ID>VAT</cbc:ID></cac:TaxScheme></cac:TaxCategory></cac:TaxSubtotal></cac:TaxTotal>`,
		// No TaxInclusiveAmount, and a tax-exclusive total below the line sum.
		`
  <cac:LegalMonetaryTotal>
    <cbc:LineExtensionAmount currencyID="EUR">100.00</cbc:LineExtensionAmount>
    <cbc:TaxExclusiveAmount currencyID="EUR">90.00</cbc:TaxExclusiveAmount>
    <cbc:PayableAmount currencyID="EUR">107.50</cbc:PayableAmount>
  </cac:LegalMonetaryTotal>`,
		edgeLine("1", "10", "100.00", "10.00"),
	)

	require.True(t, inv.HasTags(tax.TagBypass))
	assert.Equal(t, "90.00", inv.Totals.Total.String())
	assert.Equal(t, "17.50", inv.Totals.Tax.String())
	// Derived, not the 120.00 the calculation left behind.
	assert.Equal(t, "107.50", inv.Totals.TotalWithTax.String())
	assert.Equal(t, "107.50", inv.Totals.Payable.String())
}

// The optional summary totals are preserved on bypass, so they have to be
// checked too: here every core total agrees and only BT-108 differs.
func TestReconcileChecksOptionalDeclaredTotals(t *testing.T) {
	inv := edgeInvoice(t,
		`
  <cac:TaxTotal><cbc:TaxAmount currencyID="EUR">20.00</cbc:TaxAmount>
    <cac:TaxSubtotal><cbc:TaxableAmount currencyID="EUR">100.00</cbc:TaxableAmount>
      <cbc:TaxAmount currencyID="EUR">20.00</cbc:TaxAmount>
      <cac:TaxCategory><cbc:ID>S</cbc:ID><cbc:Percent>20.00</cbc:Percent>
        <cac:TaxScheme><cbc:ID>VAT</cbc:ID></cac:TaxScheme></cac:TaxCategory></cac:TaxSubtotal></cac:TaxTotal>`,
		// Every core total reconciles; only the declared charge total is wrong.
		`
  <cac:LegalMonetaryTotal>
    <cbc:LineExtensionAmount currencyID="EUR">100.00</cbc:LineExtensionAmount>
    <cbc:TaxExclusiveAmount currencyID="EUR">100.00</cbc:TaxExclusiveAmount>
    <cbc:TaxInclusiveAmount currencyID="EUR">120.00</cbc:TaxInclusiveAmount>
    <cbc:ChargeTotalAmount currencyID="EUR">25.00</cbc:ChargeTotalAmount>
    <cbc:PayableAmount currencyID="EUR">120.00</cbc:PayableAmount>
  </cac:LegalMonetaryTotal>`,
		edgeLine("1", "10", "100.00", "10.00"),
	)

	require.True(t, inv.HasTags(tax.TagBypass), "a declared charge total we cannot reproduce must not be silently replaced")
	require.NotNil(t, inv.Totals.Charge)
	assert.Equal(t, "25.00", inv.Totals.Charge.String())
}

// GOBL requires a percentage wherever a base is set. Dropping a multiplier that
// disagrees with the declared amount must drop the base with it, or the
// document fails validation at signing.
func TestDeclaredAmountWinsWithoutLeavingABase(t *testing.T) {
	inv := edgeInvoice(t,
		`
  <cac:TaxTotal><cbc:TaxAmount currencyID="EUR">16.00</cbc:TaxAmount>
    <cac:TaxSubtotal><cbc:TaxableAmount currencyID="EUR">80.00</cbc:TaxableAmount>
      <cbc:TaxAmount currencyID="EUR">16.00</cbc:TaxAmount>
      <cac:TaxCategory><cbc:ID>S</cbc:ID><cbc:Percent>20.00</cbc:Percent>
        <cac:TaxScheme><cbc:ID>VAT</cbc:ID></cac:TaxScheme></cac:TaxCategory></cac:TaxSubtotal></cac:TaxTotal>`,
		`
  <cac:LegalMonetaryTotal>
    <cbc:LineExtensionAmount currencyID="EUR">80.00</cbc:LineExtensionAmount>
    <cbc:TaxExclusiveAmount currencyID="EUR">80.00</cbc:TaxExclusiveAmount>
    <cbc:TaxInclusiveAmount currencyID="EUR">96.00</cbc:TaxInclusiveAmount>
    <cbc:PayableAmount currencyID="EUR">96.00</cbc:PayableAmount>
  </cac:LegalMonetaryTotal>`,
		`
  <cac:InvoiceLine>
    <cbc:ID>1</cbc:ID>
    <cbc:InvoicedQuantity unitCode="EA">10</cbc:InvoicedQuantity>
    <cbc:LineExtensionAmount currencyID="EUR">80.00</cbc:LineExtensionAmount>
    <cac:AllowanceCharge>
      <cbc:ChargeIndicator>false</cbc:ChargeIndicator>
      <cbc:MultiplierFactorNumeric>50.00</cbc:MultiplierFactorNumeric>
      <cbc:BaseAmount currencyID="EUR">200.00</cbc:BaseAmount>
      <cbc:Amount currencyID="EUR">20.00</cbc:Amount>
    </cac:AllowanceCharge>
    <cac:Item><cbc:Name>Widget</cbc:Name>
      <cac:ClassifiedTaxCategory><cbc:ID>S</cbc:ID><cbc:Percent>20.00</cbc:Percent>
        <cac:TaxScheme><cbc:ID>VAT</cbc:ID></cac:TaxScheme></cac:ClassifiedTaxCategory></cac:Item>
    <cac:Price><cbc:PriceAmount currencyID="EUR">10.00</cbc:PriceAmount></cac:Price>
  </cac:InvoiceLine>`,
	)

	// 50% of 200.00 is 100.00, not the declared 20.00, so the multiplier goes -
	// and the base has to go with it.
	require.Len(t, inv.Lines, 1)
	require.Len(t, inv.Lines[0].Discounts, 1)
	d := inv.Lines[0].Discounts[0]
	assert.Equal(t, "20.00", d.Amount.String())
	assert.Nil(t, d.Percent)
	assert.Nil(t, d.Base, "a base with no percent is rejected by GOBL")
}

// PEPPOL-EN16931-R040 wants a document-level allowance or charge amount to
// equal its base times its percentage, so the basis the sender calculated on
// has to survive the round trip rather than being replaced by the invoice sum.
func TestDocumentChargeBaseRoundTrip(t *testing.T) {
	data, err := os.ReadFile("test/data/parse/peppol/Allowance-example.xml")
	require.NoError(t, err)

	doc, err := ubl.Parse(data)
	require.NoError(t, err)
	in, ok := doc.(*ubl.Invoice)
	require.True(t, ok)
	env, err := in.Convert()
	require.NoError(t, err)

	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	var withBase *bill.Charge
	for _, c := range inv.Charges {
		if c.Base != nil && c.Percent != nil {
			withBase = c
			break
		}
	}
	require.NotNil(t, withBase, "the example should carry a charge with its own base")

	out, err := ubl.ConvertInvoice(env)
	require.NoError(t, err)

	var found bool
	for _, ac := range out.AllowanceCharge {
		if ac.MultiplierFactorNumeric == nil || ac.BaseAmount == nil {
			continue
		}
		if *ac.MultiplierFactorNumeric != withBase.Percent.StringWithoutSymbol() {
			continue
		}
		found = true
		assert.Equal(t, withBase.Base.Rescale(2).String(), ac.BaseAmount.Value,
			"the declared basis must be emitted, not the invoice sum")
	}
	assert.True(t, found, "the charge should round-trip with its multiplier")
}
