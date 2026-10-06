package ubl_test

import (
	"strings"
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func lineStatus(l ubl.InvoiceLine) string {
	if l.BillingReference == nil || l.BillingReference.InvoiceDocumentReference == nil {
		return ""
	}
	return l.BillingReference.InvoiceDocumentReference.DocumentStatusCode
}

func lineParent(l ubl.InvoiceLine) string {
	if l.BillingReference == nil || l.BillingReference.BillingReferenceLine == nil {
		return ""
	}
	return l.BillingReference.BillingReferenceLine.ID.Value
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
		assert.Equal(t, "FAC-2024-001", group.BillingReference.InvoiceDocumentReference.ID.Value)
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
		assert.Nil(t, doc.InvoiceLines[0].BillingReference)
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

func parseSubLines(t *testing.T, xml string) *bill.Invoice {
	t.Helper()
	doc, err := ubl.Parse([]byte(xml))
	require.NoError(t, err)
	ui, ok := doc.(*ubl.Invoice)
	require.True(t, ok)
	env, err := ui.Convert()
	require.NoError(t, err)
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)
	return inv
}

// TestParseSubInvoiceLines covers the sub-invoice lines of EXTENDED-CTC-FR.
// Only DETAIL lines count towards the totals: a GROUP line restates the sum
// of its DETAIL lines and an INFORMATION line carries no amount. GROUP and
// INFORMATION lines may come without a price or a tax category.
func TestParseSubInvoiceLines(t *testing.T) {
	e := parseXMLInvoice(t, "sub-invoice-lines.xml")
	inv, ok := e.Extract().(*bill.Invoice)
	require.True(t, ok)
	require.Len(t, inv.Lines, 5)

	// The first group mixes tax rates, which a breakdown cannot, so it is read
	// flat with the GROUP line at a zero price.
	names := []string{"Kit", "Part A", "Part B", "Service bundle", "Assembly instructions"}
	totals := []string{amountZero, "100.00", "50.00", "20.00", amountZero}
	for i, l := range inv.Lines {
		assert.Equal(t, names[i], l.Item.Name)
		require.NotNil(t, l.Total)
		assert.Equal(t, totals[i], l.Total.String(), "line %d", i+1)
	}
	assert.Empty(t, inv.Lines[0].Taxes)
	assert.Empty(t, inv.Lines[0].Breakdown)

	// The second shares one rate and becomes a breakdown.
	bundle := inv.Lines[3]
	require.Len(t, bundle.Breakdown, 1)
	assert.Equal(t, "Installation", bundle.Breakdown[0].Item.Name)
	require.Len(t, bundle.Taxes, 1)
	assert.Equal(t, "20%", bundle.Taxes[0].Percent.String())

	// The calculation reproduces the declared totals without being bypassed.
	assert.False(t, inv.HasTags(tax.TagBypass))
	assert.Equal(t, "170.00", inv.Totals.Sum.String())
	assert.Equal(t, "170.00", inv.Totals.Total.String())
	assert.Equal(t, "29.00", inv.Totals.Tax.String())
	assert.Equal(t, "199.00", inv.Totals.Payable.String())

	require.NoError(t, e.Validate())
}

// TestParseSubInvoiceLinesFlat covers the groups a breakdown cannot hold, which
// are read flat, next to the ones it can.
func TestParseSubInvoiceLinesFlat(t *testing.T) {
	e := parseXMLInvoice(t, "sub-invoice-lines-flat.xml")
	inv, ok := e.Extract().(*bill.Invoice)
	require.True(t, ok)

	want := []struct {
		name      string
		total     string
		breakdown int
	}{
		// Nested groups.
		{"Display", amountZero, 0},
		{"Roast", "30.00", 0},
		{"Bundle", amountZero, 0},
		{"Colombia", "90.00", 0},
		// DETAIL lines ahead of their GROUP line still fold into it.
		{"Hardware", "550.00", 2},
		// A quantity that does not divide by the GROUP line's.
		{"Odd lot", amountZero, 0},
		{"Thing", "10.00", 0},
		// A GROUP line declaring more than its DETAIL lines add up to.
		{"Mismatch", amountZero, 0},
		{"Part", "40.00", 0},
		// INFORMATION lines describe a line that keeps its own price.
		{"Safety kit", "450.00", 2},
	}
	require.Len(t, inv.Lines, len(want))
	for i, w := range want {
		l := inv.Lines[i]
		assert.Equal(t, w.name, l.Item.Name, "line %d", i+1)
		assert.Equal(t, w.total, l.Total.String(), "line %d", i+1)
		assert.Len(t, l.Breakdown, w.breakdown, "line %d", i+1)
	}

	hardware := inv.Lines[4]
	assert.Equal(t, "Laser printer", hardware.Breakdown[0].Item.Name)
	assert.Equal(t, "-1", hardware.Breakdown[1].Quantity.String())

	kit := inv.Lines[9]
	assert.Equal(t, "45.00", kit.Item.Price.String())
	for _, sl := range kit.Breakdown {
		assert.Nil(t, sl.Item.Price)
		assert.Equal(t, "1", sl.Quantity.String())
	}

	assert.False(t, inv.HasTags(tax.TagBypass))
	assert.Equal(t, "1170.00", inv.Totals.Sum.String())
	assert.Equal(t, "234.00", inv.Totals.Tax.String())
	assert.Equal(t, "1404.00", inv.Totals.Payable.String())
	require.NoError(t, e.Validate())
}

// TestParseSubInvoiceLinesUnreconciled covers a group whose DETAIL lines
// declare amounts their prices do not produce: the breakdown would not come to
// the declared amount, so the group is read flat.
func TestParseSubInvoiceLinesUnreconciled(t *testing.T) {
	data, err := testLoadXML("sub-invoice-lines.xml")
	require.NoError(t, err)
	// The bundle's only DETAIL line declares 25.00 at a 20.00 price.
	xml := strings.Replace(string(data),
		`<cbc:LineExtensionAmount currencyID="EUR">20.00</cbc:LineExtensionAmount>`,
		`<cbc:LineExtensionAmount currencyID="EUR">25.00</cbc:LineExtensionAmount>`, 2)

	inv := parseSubLines(t, xml)
	require.Len(t, inv.Lines, 6)
	assert.Equal(t, "Service bundle", inv.Lines[3].Item.Name)
	assert.Empty(t, inv.Lines[3].Breakdown)
	assert.Equal(t, "Installation", inv.Lines[4].Item.Name)
}

// TestParseSubInvoiceLinesCycle covers lines naming each other as parents,
// which never lead back to a top-level line: they are still read.
func TestParseSubInvoiceLinesCycle(t *testing.T) {
	data, err := testLoadXML("sub-invoice-lines.xml")
	require.NoError(t, err)
	xml := strings.Replace(string(data),
		"<cbc:DocumentStatusCode>GROUP</cbc:DocumentStatusCode>\n      </cac:InvoiceDocumentReference>\n",
		"<cbc:DocumentStatusCode>GROUP</cbc:DocumentStatusCode>\n      </cac:InvoiceDocumentReference>\n      <cac:BillingReferenceLine>\n        <cbc:ID>2</cbc:ID>\n      </cac:BillingReferenceLine>\n", 1)
	require.Contains(t, xml, "<cbc:ID>2</cbc:ID>\n      </cac:BillingReferenceLine>")

	inv := parseSubLines(t, xml)
	assert.Len(t, inv.Lines, 5)
	assert.Equal(t, "170.00", inv.Totals.Sum.String())
}
