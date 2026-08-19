package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLines(t *testing.T) {
	t.Run("invoice-without-buyers-tax-id.json", func(t *testing.T) {
		doc := testInvoiceFrom(t, "invoice-without-buyers-tax-id.json")

		assert.NotNil(t, doc.InvoiceLines)
		assert.Len(t, doc.InvoiceLines, 1)
		assert.Equal(t, "1", doc.InvoiceLines[0].ID)
		assert.Equal(t, "1800.00", doc.InvoiceLines[0].LineExtensionAmount.Value)
		assert.Equal(t, "Development services", doc.InvoiceLines[0].Item.Name)
		assert.Equal(t, "HUR", doc.InvoiceLines[0].InvoicedQuantity.UnitCode)
		assert.Equal(t, "VAT", doc.InvoiceLines[0].Item.ClassifiedTaxCategory.TaxScheme.ID.Value)
		assert.Equal(t, "19", *doc.InvoiceLines[0].Item.ClassifiedTaxCategory.Percent)
		assert.True(t, doc.InvoiceLines[0].AllowanceCharge[0].ChargeIndicator)
		assert.Equal(t, "Testing", *doc.InvoiceLines[0].AllowanceCharge[0].AllowanceChargeReason)
		assert.Equal(t, "12.00", doc.InvoiceLines[0].AllowanceCharge[0].Amount.Value)
		assert.False(t, doc.InvoiceLines[0].AllowanceCharge[1].ChargeIndicator)
		assert.Equal(t, "Damage", *doc.InvoiceLines[0].AllowanceCharge[1].AllowanceChargeReason)
		assert.Equal(t, "12.00", doc.InvoiceLines[0].AllowanceCharge[1].Amount.Value)
		assert.Equal(t, "0088", *doc.InvoiceLines[0].Item.StandardItemIdentification.ID.SchemeID)
		assert.Equal(t, "1234567890128", doc.InvoiceLines[0].Item.StandardItemIdentification.ID.Value)
	})

	t.Run("invoice-with-line-order.json", func(t *testing.T) {
		doc := testInvoiceFrom(t, "invoice-with-line-order.json")

		assert.NotNil(t, doc.InvoiceLines)
		assert.Len(t, doc.InvoiceLines, 1)
		assert.Equal(t, "1", doc.InvoiceLines[0].ID)
		assert.NotNil(t, doc.InvoiceLines[0].OrderLineReference)
		assert.Equal(t, "123", doc.InvoiceLines[0].OrderLineReference.LineID)
		assert.Equal(t, "DEVSERV001", doc.InvoiceLines[0].Item.SellersItemIdentification.ID.Value)

		// First identity with extension maps to StandardItemIdentification
		assert.NotNil(t, doc.InvoiceLines[0].Item.StandardItemIdentification)
		assert.Equal(t, "0088", *doc.InvoiceLines[0].Item.StandardItemIdentification.ID.SchemeID)
		assert.Equal(t, "1234567890128", doc.InvoiceLines[0].Item.StandardItemIdentification.ID.Value)

		// First identity without extension maps to BuyersItemIdentification
		assert.NotNil(t, doc.InvoiceLines[0].Item.BuyersItemIdentification)
		assert.Nil(t, doc.InvoiceLines[0].Item.BuyersItemIdentification.ID.SchemeID)
		assert.Equal(t, "1234567890128", doc.InvoiceLines[0].Item.BuyersItemIdentification.ID.Value)
	})

	t.Run("invoice-zero-quantity.json", func(t *testing.T) {
		doc := testInvoiceFrom(t, "invoice-zero-quantity.json")

		assert.NotNil(t, doc.InvoiceLines)
		assert.Len(t, doc.InvoiceLines, 1)
		assert.Equal(t, "1", doc.InvoiceLines[0].ID)
		assert.Equal(t, "0.00", doc.InvoiceLines[0].LineExtensionAmount.Value)
		assert.Equal(t, "Development services", doc.InvoiceLines[0].Item.Name)

		// Quantity should always be set, even when zero (mandatory field)
		assert.NotNil(t, doc.InvoiceLines[0].InvoicedQuantity)
		assert.Equal(t, "0", doc.InvoiceLines[0].InvoicedQuantity.Value)
		assert.Equal(t, "HUR", doc.InvoiceLines[0].InvoicedQuantity.UnitCode)
	})

	// BR-DEC-24 / UBL-DT-01: when prices_include=VAT, RemoveIncludedTaxes
	// leaves discount amounts with extra precision (1.76 / 1.21 = 1.4545).
	// The converter must round line allowance amounts to 2 decimals to satisfy
	// EN16931 / Peppol BIS 3.0, even though the item net price (BT-146) is
	// allowed to keep its higher precision.
	t.Run("invoice-prices-include-vat.json", func(t *testing.T) {
		doc := testInvoiceFrom(t, "invoice-prices-include-vat.json")

		require.Len(t, doc.InvoiceLines, 2)

		l1 := doc.InvoiceLines[0]
		require.Len(t, l1.AllowanceCharge, 1)
		assert.False(t, l1.AllowanceCharge[0].ChargeIndicator)
		assert.Equal(t, "1.45", l1.AllowanceCharge[0].Amount.Value)
		assert.Equal(t, "8.1818", l1.Price.PriceAmount.Value)

		l2 := doc.InvoiceLines[1]
		require.Len(t, l2.AllowanceCharge, 1)
		assert.Equal(t, "11.02", l2.AllowanceCharge[0].Amount.Value)
		assert.Equal(t, "61.9008", l2.Price.PriceAmount.Value)
	})

}

func TestLineNoteSubjectCodeRoundTrip(t *testing.T) {
	env := loadTestEnvelope(t, "invoice-minimal.json")
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	inv.Lines[0].Notes = []*org.Note{
		{
			Text: "Handle with care",
			Ext:  tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyTextSubject: "AAI"}),
		},
	}

	doc, err := ubl.ConvertInvoice(env)
	require.NoError(t, err)

	require.NotEmpty(t, doc.InvoiceLines[0].Note)
	assert.Equal(t, "#AAI#Handle with care", doc.InvoiceLines[0].Note[0])

	data, err := ubl.Bytes(doc)
	require.NoError(t, err)

	parsed, err := ubl.Parse(data)
	require.NoError(t, err)
	out, ok := parsed.(*ubl.Invoice)
	require.True(t, ok)
	outEnv, err := out.Convert()
	require.NoError(t, err)
	outInv, ok := outEnv.Extract().(*bill.Invoice)
	require.True(t, ok)

	require.NotEmpty(t, outInv.Lines[0].Notes)
	n := outInv.Lines[0].Notes[0]
	assert.Equal(t, "Handle with care", n.Text)
	assert.Equal(t, cbc.Code("AAI"), n.Ext.Get(untdid.ExtKeyTextSubject))
}

func TestItemAttributeRoundTrip(t *testing.T) {
	env := loadTestEnvelope(t, "invoice-minimal.json")
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	weight := num.MakeAmount(25, 1) // 2.5
	inv.Lines[0].Item.Attributes = []*org.Attribute{
		{Label: "Color", Text: "Black"},
		{Label: "Weight", Amount: &weight, Unit: "kg"},
	}
	require.NoError(t, env.Calculate())

	doc, err := ubl.ConvertInvoice(env)
	require.NoError(t, err)

	require.NotNil(t, doc.InvoiceLines[0].Item.AdditionalItemProperty)
	props := *doc.InvoiceLines[0].Item.AdditionalItemProperty
	require.Len(t, props, 2)

	assert.Equal(t, "Color", props[0].Name)
	assert.Equal(t, "Black", props[0].Value)
	assert.Nil(t, props[0].ValueQuantity)

	assert.Equal(t, "Weight", props[1].Name)
	// BR-54 requires a plain Value even when ValueQuantity is also provided.
	assert.Equal(t, "2.5 kg", props[1].Value)
	require.NotNil(t, props[1].ValueQuantity)
	assert.Equal(t, "2.5", props[1].ValueQuantity.Value)
	assert.Equal(t, "KGM", props[1].ValueQuantity.UnitCode)

	data, err := ubl.Bytes(doc)
	require.NoError(t, err)

	parsed, err := ubl.Parse(data)
	require.NoError(t, err)
	out, ok := parsed.(*ubl.Invoice)
	require.True(t, ok)
	outEnv, err := out.Convert()
	require.NoError(t, err)
	outInv, ok := outEnv.Extract().(*bill.Invoice)
	require.True(t, ok)

	require.Len(t, outInv.Lines[0].Item.Attributes, 2)
	assert.Equal(t, "Color", outInv.Lines[0].Item.Attributes[0].Label)
	assert.Equal(t, "Black", outInv.Lines[0].Item.Attributes[0].Text)
	assert.Equal(t, "Weight", outInv.Lines[0].Item.Attributes[1].Label)
	require.NotNil(t, outInv.Lines[0].Item.Attributes[1].Amount)
	assert.Equal(t, "2.5", outInv.Lines[0].Item.Attributes[1].Amount.String())
	assert.Equal(t, org.Unit("kg"), outInv.Lines[0].Item.Attributes[1].Unit)
}

func TestLineSellerRoundTrip(t *testing.T) {
	env := loadTestEnvelope(t, "invoice-minimal.json")
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	inv.Lines[0].Seller = &org.Party{
		Inboxes: []*org.Inbox{
			{Email: "seller@example.com"},
		},
	}
	require.NoError(t, env.Calculate())

	doc, err := ubl.ConvertInvoice(env)
	require.NoError(t, err)

	require.NotNil(t, doc.InvoiceLines[0].Item.ManufacturerParty)
	require.NotNil(t, doc.InvoiceLines[0].Item.ManufacturerParty.EndpointID)
	assert.Equal(t, ubl.SchemeIDEmail, doc.InvoiceLines[0].Item.ManufacturerParty.EndpointID.SchemeID)
	assert.Equal(t, "seller@example.com", doc.InvoiceLines[0].Item.ManufacturerParty.EndpointID.Value)

	data, err := ubl.Bytes(doc)
	require.NoError(t, err)

	parsed, err := ubl.Parse(data)
	require.NoError(t, err)
	out, ok := parsed.(*ubl.Invoice)
	require.True(t, ok)
	outEnv, err := out.Convert()
	require.NoError(t, err)
	outInv, ok := outEnv.Extract().(*bill.Invoice)
	require.True(t, ok)

	require.NotNil(t, outInv.Lines[0].Seller)
	require.Len(t, outInv.Lines[0].Seller.Inboxes, 1)
	assert.Equal(t, "seller@example.com", outInv.Lines[0].Seller.Inboxes[0].Email)
}
