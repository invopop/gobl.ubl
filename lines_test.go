package ubl_test

import (
	"testing"

	"github.com/invopop/gobl"
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
	assert.Equal(t, cbc.Key("kg"), outInv.Lines[0].Item.Attributes[1].Unit)
	assert.Equal(t, cbc.Code("KGM"), outInv.Lines[0].Item.Attributes[1].Ext.Get(untdid.ExtKeyUnit))
}

// TestItemAttributeUnmappedUnitRoundTrip covers a UNTDID unit code that GOBL
// has no key for, which is preserved in the attribute's extensions.
func TestItemAttributeUnmappedUnitRoundTrip(t *testing.T) {
	env := loadTestEnvelope(t, "invoice-minimal.json")
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	length := num.MakeAmount(25, 1) // 2.5
	inv.Lines[0].Item.Attributes = []*org.Attribute{
		{
			Label:  "Length",
			Amount: &length,
			Ext:    tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyUnit: "X4G"}),
		},
	}
	require.NoError(t, env.Calculate())

	doc, err := ubl.ConvertInvoice(env)
	require.NoError(t, err)

	props := *doc.InvoiceLines[0].Item.AdditionalItemProperty
	require.Len(t, props, 1)
	// With no GOBL key the raw code stands in as the presentation label.
	assert.Equal(t, "2.5 X4G", props[0].Value)
	require.NotNil(t, props[0].ValueQuantity)
	assert.Equal(t, "X4G", props[0].ValueQuantity.UnitCode)

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

	require.Len(t, outInv.Lines[0].Item.Attributes, 1)
	attr := outInv.Lines[0].Item.Attributes[0]
	assert.Equal(t, cbc.KeyEmpty, attr.Unit)
	assert.Equal(t, cbc.Code("X4G"), attr.Ext.Get(untdid.ExtKeyUnit))
}

// TestItemAttributeWithoutUnit covers an attribute whose amount carries no
// unit, which cannot produce a UBL quantity.
func TestItemAttributeWithoutUnit(t *testing.T) {
	env := loadTestEnvelope(t, "invoice-minimal.json")
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	count := num.MakeAmount(25, 1) // 2.5
	inv.Lines[0].Item.Attributes = []*org.Attribute{
		{Label: "Rating", Amount: &count},
	}
	require.NoError(t, env.Calculate())

	doc, err := ubl.ConvertInvoice(env)
	require.NoError(t, err)

	props := *doc.InvoiceLines[0].Item.AdditionalItemProperty
	require.Len(t, props, 1)
	assert.Equal(t, "2.5", props[0].Value)
	assert.Nil(t, props[0].ValueQuantity)
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

const (
	lineStatusGroup       = "GROUP"
	lineStatusDetail      = "DETAIL"
	lineStatusInformation = "INFORMATION"

	fixtureFRExtended = "france-extended/invoice-standard.json"
	amountZero        = "0.00"
)

// breakdownEnvelope loads the French extended fixture and gives its line a
// breakdown: two priced sub-lines, one of them discounted, and one without a
// price.
func breakdownEnvelope(t *testing.T) *gobl.Envelope {
	t.Helper()
	env := loadTestEnvelope(t, fixtureFRExtended)
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)

	design := num.MakeAmount(3000, 2)
	build := num.MakeAmount(6000, 2)
	inv.Lines[0].Breakdown = []*bill.SubLine{
		{Quantity: num.MakeAmount(1, 0), Item: &org.Item{Name: "Design", Price: &design}},
		{
			Quantity:  num.MakeAmount(2, 0),
			Item:      &org.Item{Name: "Build", Price: &build},
			Discounts: []*bill.LineDiscount{{Amount: num.MakeAmount(500, 2), Reason: "Promotion"}},
		},
		{Quantity: num.MakeAmount(1, 0), Item: &org.Item{Name: "Sample, no charge"}},
	}
	require.NoError(t, env.Calculate())
	return env
}

func lineHierarchy(l ubl.InvoiceLine) *ubl.LineBillingReference {
	if len(l.BillingReference) == 0 {
		return nil
	}
	return l.BillingReference[0]
}

func lineStatus(l ubl.InvoiceLine) string {
	br := lineHierarchy(l)
	if br == nil || br.InvoiceDocumentReference == nil {
		return ""
	}
	return br.InvoiceDocumentReference.DocumentStatusCode
}

func lineParent(l ubl.InvoiceLine) string {
	br := lineHierarchy(l)
	if br == nil || br.BillingReferenceLine == nil {
		return ""
	}
	return br.BillingReferenceLine.ID.Value
}

func TestSubLinesConvert(t *testing.T) {
	t.Run("extended profile writes sub-invoice lines", func(t *testing.T) {
		doc, err := ubl.ConvertInvoice(breakdownEnvelope(t), ubl.WithContext(ubl.ContextPeppolFranceExtended))
		require.NoError(t, err)
		lines := doc.InvoiceLines
		require.Len(t, lines, 4)

		// The parent quantity is 10, and each sub-line counts per unit of it.
		group := lines[0]
		assert.Equal(t, "1", group.ID)
		assert.Equal(t, lineStatusGroup, lineStatus(group))
		assert.Empty(t, lineParent(group))
		assert.Equal(t, "FAC-2024-001", lineHierarchy(group).InvoiceDocumentReference.ID.Value)
		assert.Nil(t, group.Item.ClassifiedTaxCategory)
		assert.Equal(t, "145.00", group.Price.PriceAmount.Value)
		assert.Equal(t, "10", group.InvoicedQuantity.Value)
		assert.Equal(t, "1450.00", group.LineExtensionAmount.Value)

		design := lines[1]
		assert.Equal(t, "1.1", design.ID)
		assert.Equal(t, "1", lineParent(design))
		assert.Equal(t, lineStatusDetail, lineStatus(design))
		assert.Equal(t, "Design", design.Item.Name)
		assert.Equal(t, "10", design.InvoicedQuantity.Value)
		assert.Equal(t, "30.00", design.Price.PriceAmount.Value)
		assert.Equal(t, "300.00", design.LineExtensionAmount.Value)
		require.NotNil(t, design.Item.ClassifiedTaxCategory)
		assert.Equal(t, "S", design.Item.ClassifiedTaxCategory.ID.Value)

		build := lines[2]
		assert.Equal(t, "1.2", build.ID)
		assert.Equal(t, "20", build.InvoicedQuantity.Value)
		require.Len(t, build.AllowanceCharge, 1)
		assert.Equal(t, "50.00", build.AllowanceCharge[0].Amount.Value)
		assert.Equal(t, "1150.00", build.LineExtensionAmount.Value)

		info := lines[3]
		assert.Equal(t, "1.3", info.ID)
		assert.Equal(t, lineStatusInformation, lineStatus(info))
		assert.Equal(t, "1", lineParent(info))
		assert.Equal(t, amountZero, info.Price.PriceAmount.Value)
		assert.Equal(t, amountZero, info.LineExtensionAmount.Value)
	})

	t.Run("other profiles write the line alone", func(t *testing.T) {
		doc, err := ubl.ConvertInvoice(breakdownEnvelope(t), ubl.WithContext(ubl.ContextPeppolFranceCIUS))
		require.NoError(t, err)
		require.Len(t, doc.InvoiceLines, 1)
		assert.Empty(t, doc.InvoiceLines[0].BillingReference)
		assert.Equal(t, "1450.00", doc.InvoiceLines[0].LineExtensionAmount.Value)
	})

	t.Run("a line with its own discount is written alone", func(t *testing.T) {
		env := breakdownEnvelope(t)
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		inv.Lines[0].Discounts = []*bill.LineDiscount{{Amount: num.MakeAmount(1000, 2), Reason: "Loyalty"}}
		require.NoError(t, env.Calculate())

		doc, err := ubl.ConvertInvoice(env, ubl.WithContext(ubl.ContextPeppolFranceExtended))
		require.NoError(t, err)
		require.Len(t, doc.InvoiceLines, 1)
	})

	t.Run("a line whose sub-lines round apart is written alone", func(t *testing.T) {
		env := loadTestEnvelope(t, fixtureFRExtended)
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		// Each sub-line comes to 0.33 x 1.5 = 0.495, so 0.50 written, while
		// the line is 0.66 x 1.5 = 0.99.
		price := num.MakeAmount(33, 2)
		inv.Lines[0].Quantity = num.MakeAmount(15, 1)
		inv.Lines[0].Breakdown = []*bill.SubLine{
			{Quantity: num.MakeAmount(1, 0), Item: &org.Item{Name: "Part A", Price: &price}},
			{Quantity: num.MakeAmount(1, 0), Item: &org.Item{Name: "Part B", Price: &price}},
		}
		require.NoError(t, env.Calculate())

		doc, err := ubl.ConvertInvoice(env, ubl.WithContext(ubl.ContextPeppolFranceExtended))
		require.NoError(t, err)
		require.Len(t, doc.InvoiceLines, 1)
		assert.Empty(t, doc.InvoiceLines[0].BillingReference)
		assert.Equal(t, "0.99", doc.InvoiceLines[0].LineExtensionAmount.Value)
	})

	t.Run("unpriced sub-lines describe a line that keeps its price", func(t *testing.T) {
		env := loadTestEnvelope(t, fixtureFRExtended)
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		inv.Lines[0].Breakdown = []*bill.SubLine{
			{Quantity: num.MakeAmount(1, 0), Item: &org.Item{Name: "Helmet"}},
		}
		require.NoError(t, env.Calculate())

		doc, err := ubl.ConvertInvoice(env, ubl.WithContext(ubl.ContextPeppolFranceExtended))
		require.NoError(t, err)
		lines := doc.InvoiceLines
		require.Len(t, lines, 2)
		assert.Equal(t, lineStatusDetail, lineStatus(lines[0]))
		assert.NotNil(t, lines[0].Item.ClassifiedTaxCategory)
		assert.Equal(t, "1000.00", lines[0].LineExtensionAmount.Value)
		assert.Equal(t, lineStatusInformation, lineStatus(lines[1]))
	})
}

// TestSubLinesRoundTrip writes a breakdown out as sub-invoice lines and reads
// it back: the breakdown and every total must survive.
func TestSubLinesRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		env  func(t *testing.T) *gobl.Envelope
	}{
		{"invoice-sub-lines", func(t *testing.T) *gobl.Envelope {
			return loadTestEnvelope(t, "france-extended/invoice-sub-lines.json")
		}},
		{"breakdown", breakdownEnvelope},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.env(t)
			inv, ok := env.Extract().(*bill.Invoice)
			require.True(t, ok)

			doc, err := ubl.ConvertInvoice(env, ubl.WithContext(ubl.ContextPeppolFranceExtended))
			require.NoError(t, err)
			data, err := ubl.Bytes(doc)
			require.NoError(t, err)

			parsed, err := ubl.Parse(data)
			require.NoError(t, err)
			pd, ok := parsed.(*ubl.Invoice)
			require.True(t, ok)
			penv, err := pd.Convert()
			require.NoError(t, err)
			out, ok := penv.Extract().(*bill.Invoice)
			require.True(t, ok)

			assert.False(t, out.HasTags(tax.TagBypass))
			require.Len(t, out.Lines, len(inv.Lines))
			for i, l := range inv.Lines {
				got := out.Lines[i]
				assert.Equal(t, l.Quantity.String(), got.Quantity.String(), "line %d quantity", i+1)
				assert.Equal(t, l.Total.String(), got.Total.String(), "line %d total", i+1)
				require.Len(t, got.Breakdown, len(l.Breakdown), "line %d breakdown", i+1)
				for j, sl := range l.Breakdown {
					gs := got.Breakdown[j]
					assert.Equal(t, sl.Item.Name, gs.Item.Name)
					assert.True(t, sl.Quantity.Equals(gs.Quantity), "sub-line %d.%d quantity", i+1, j+1)
					if sl.Item.Price == nil {
						assert.Nil(t, gs.Item.Price)
						assert.Nil(t, gs.Total)
						continue
					}
					require.NotNil(t, gs.Item.Price)
					assert.True(t, sl.Item.Price.Equals(*gs.Item.Price), "sub-line %d.%d price", i+1, j+1)
					assert.Equal(t, sl.Total.String(), gs.Total.String(), "sub-line %d.%d total", i+1, j+1)
					assert.Len(t, gs.Discounts, len(sl.Discounts))
				}
			}
			assert.Equal(t, inv.Totals.Sum.String(), out.Totals.Sum.String())
			assert.Equal(t, inv.Totals.Tax.String(), out.Totals.Tax.String())
			assert.Equal(t, inv.Totals.Payable.String(), out.Totals.Payable.String())
		})
	}
}
