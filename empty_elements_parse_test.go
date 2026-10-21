package ubl_test

import (
	"strings"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEmptyElements(t *testing.T) {
	data, err := testLoadXML("en16931/ubl-example2.xml")
	require.NoError(t, err)
	doc := string(data)
	inject := func(anchor, extra string) {
		t.Helper()
		require.Equal(t, 1, strings.Count(doc, anchor), anchor)
		doc = strings.Replace(doc, anchor, anchor+extra, 1)
	}
	inject("<cbc:InvoiceTypeCode>380</cbc:InvoiceTypeCode>", `<cbc:Note></cbc:Note>`)
	inject("<cbc:AccountingCost>Project cost code 123</cbc:AccountingCost>",
		`<cac:OrderReference><cbc:ID/></cac:OrderReference>`+
			`<cac:BillingReference><cac:InvoiceDocumentReference><cbc:ID/></cac:InvoiceDocumentReference></cac:BillingReference>`+
			`<cac:DespatchDocumentReference><cbc:ID/></cac:DespatchDocumentReference>`)
	inject("<cac:TaxRepresentativeParty>",
		`<cbc:EndpointID schemeID="0088"/>`+
			`<cac:PartyIdentification><cbc:ID schemeID="0088"></cbc:ID></cac:PartyIdentification>`+
			`<cac:PartyLegalEntity><cbc:CompanyID schemeID="0007"/></cac:PartyLegalEntity>`+
			`<cac:Contact><cbc:Name/><cbc:Telephone/><cbc:ElectronicMail/></cac:Contact>`)
	inject("</cac:Delivery>", `<cac:DeliveryTerms><cbc:ID/></cac:DeliveryTerms>`)
	inject("<cbc:Name>Laptop computer</cbc:Name>",
		`<cac:BuyersItemIdentification><cbc:ID/></cac:BuyersItemIdentification>`+
			`<cac:CommodityClassification><cbc:ItemClassificationCode listID="ZZZ"/></cac:CommodityClassification>`)
	inject("<cbc:AllowanceChargeReason>Damage</cbc:AllowanceChargeReason>",
		`<cbc:AllowanceChargeReasonCode/><cbc:MultiplierFactorNumeric/><cbc:BaseAmount currencyID="NOK"/>`)

	parsed, err := ubl.Decode([]byte(doc))
	require.NoError(t, err)
	env, err := ubl.Import(parsed.(*ubl.Invoice))
	require.NoError(t, err)
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	require.Len(t, inv.Notes, 1)
	assert.Empty(t, inv.Preceding)
	assert.Empty(t, inv.Ordering.Purchases)
	assert.Empty(t, inv.Ordering.Despatch)

	rep := inv.Ordering.Seller
	require.NotNil(t, rep)
	assert.Empty(t, rep.Inboxes)
	assert.Empty(t, rep.Identities)
	assert.Empty(t, rep.People)
	assert.Empty(t, rep.Telephones)
	assert.Empty(t, rep.Emails)

	require.Len(t, inv.Delivery.Identities, 1)
	assert.Equal(t, "6754238987643", inv.Delivery.Identities[0].Code.String())

	line := inv.Lines[0]
	require.Len(t, line.Item.Identities, 3)
	for _, id := range line.Item.Identities {
		assert.NotEmpty(t, id.Code)
	}
	require.Len(t, line.Discounts, 1)
	assert.True(t, line.Discounts[0].Ext.IsZero())
	assert.Nil(t, line.Discounts[0].Percent)
	assert.Nil(t, line.Discounts[0].Base)
}
