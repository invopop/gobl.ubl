package ubl_test

import (
	"strings"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define tests for the ParseXMLLines function
func TestParseLines(t *testing.T) {
	t.Run("ubl-example1.xml", func(t *testing.T) {
		e := parseXMLInvoice(t, "en16931/ubl-example1.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		lines := inv.Lines
		assert.NotNil(t, lines)

		assert.Len(t, lines, 20)

		line := lines[0]
		assert.Equal(t, "PATAT FRITES 10MM 10KG", line.Item.Name)
		assert.Equal(t, "2", line.Quantity.String())
		assert.Equal(t, cbc.Key("item"), line.Item.Unit)
		assert.Equal(t, "9.95", line.Item.Price.String())
		assert.Equal(t, cbc.Code("VAT"), line.Taxes[0].Category)
		assert.Equal(t, "6%", line.Taxes[0].Percent.String())

		line = lines[19]
		assert.Equal(t, "FRITUUR VET 10 KG RETOUR", line.Item.Name)
		assert.Equal(t, "6", line.Quantity.String())
		assert.Equal(t, cbc.Key("item"), line.Item.Unit)
		assert.Equal(t, "18.33", line.Item.Price.String())
		assert.Equal(t, cbc.Code("VAT"), line.Taxes[0].Category)
		assert.Equal(t, "6%", line.Taxes[0].Percent.String())
	})

	// Line Charges and Discounts
	t.Run("ubl-example2.xml", func(t *testing.T) {
		e := parseXMLInvoice(t, "en16931/ubl-example2.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		lines := inv.Lines
		assert.NotNil(t, lines)
		assert.Len(t, lines, 5)

		// Check the first line
		line := lines[0]
		assert.Equal(t, "Laptop computer", line.Item.Name)
		assert.Equal(t, "2", line.Quantity.String())
		assert.Equal(t, "JB007", line.Item.Ref.String())
		assert.Equal(t, "Scratch on box", line.Notes[0].Text)
		assert.Equal(t, "Processor: Intel Core 2 Duo SU9400 LV (1.4GHz). RAM: 3MB. Screen 1440x900", line.Item.Description)
		assert.Equal(t, cbc.Key("item"), line.Item.Unit)
		assert.Equal(t, l10n.ISOCountryCode("DE"), line.Item.Origin)
		assert.Equal(t, "1273.00", line.Item.Price.String())
		assert.Equal(t, cbc.Code("VAT"), line.Taxes[0].Category)
		assert.Equal(t, "25%", line.Taxes[0].Percent.String())

		assert.Len(t, line.Charges, 1)
		charge := line.Charges[0]
		assert.Equal(t, "12.00", charge.Amount.String())
		assert.Equal(t, "Testing", charge.Reason)

		assert.Len(t, line.Discounts, 1)
		discount := line.Discounts[0]
		assert.Equal(t, "12.00", discount.Amount.String())
		assert.Equal(t, "Damage", discount.Reason)

		assert.Len(t, line.Item.Identities, 3)
		assert.Equal(t, cbc.Code("1234567890128"), line.Item.Identities[0].Code)
		assert.Equal(t, "0088", line.Item.Identities[0].Ext.Get(iso.ExtKeySchemeID).String())
		assert.Equal(t, cbc.Code("12344321"), line.Item.Identities[1].Code)
		assert.Equal(t, "ZZZ", line.Item.Identities[1].Label)
		assert.Equal(t, cbc.Code("65434568"), line.Item.Identities[2].Code)
		assert.Equal(t, "STI", line.Item.Identities[2].Label)

		require.Len(t, line.Item.Attributes, 1)
		assert.Equal(t, "Color", line.Item.Attributes[0].Label)
		assert.Equal(t, "Black", line.Item.Attributes[0].Text)

		// Check the second line
		line = lines[1]
		assert.Equal(t, "Returned \"Advanced computing\" book", line.Item.Name)
		assert.Equal(t, "-1", line.Quantity.String())
		assert.Equal(t, cbc.Key("item"), line.Item.Unit)
		assert.Equal(t, "3.96", line.Item.Price.String())
		assert.Equal(t, cbc.Code("VAT"), line.Taxes[0].Category)
		assert.Equal(t, "15%", line.Taxes[0].Percent.String())
	})

	// Test OrderLineReference parsing
	t.Run("partial-invoice.xml with OrderLineReference", func(t *testing.T) {
		e := parseXMLInvoice(t, "peppol/partial-invoice.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		lines := inv.Lines
		assert.NotNil(t, lines)
		assert.Len(t, lines, 2)

		// Check the first line has order reference
		line := lines[0]
		assert.Equal(t, cbc.Code("123"), line.Order)

		// Check the second line has order reference
		line = lines[1]
		assert.Equal(t, cbc.Code("123"), line.Order)
	})

	// Test BaseQuantity logic
	t.Run("BaseQuantity price calculation", func(t *testing.T) {
		e := parseXMLInvoice(t, "peppol/Allowance-example.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		lines := inv.Lines
		assert.NotNil(t, lines)
		assert.Len(t, lines, 3)

		// Check the second line which has BaseQuantity = 2 and PriceAmount = 200
		// Expected unit price should be 200/2 = 100 (precision: 2 + ceil(log10(2)) = 2 + 1 = 3)
		line := lines[1]
		assert.Equal(t, "100.00", line.Item.Price.String(), "Price should be divided by BaseQuantity (200/2=100)")

		// Check the first line which has BaseQuantity = 1 and PriceAmount = 410
		// Expected unit price should be 410/1 = 410 (precision: 2 + ceil(log10(1)) = 2 + 0 = 2)
		line = lines[0]
		assert.Equal(t, "410.00", line.Item.Price.String(), "Price should be divided by BaseQuantity (410/1=410)")

		// Test the convert amount from String error handling
		data, err := testLoadXML("peppol/Allowance-example.xml")
		require.NoError(t, err)
		invalidXML := strings.ReplaceAll(string(data), `<cbc:BaseAmount currencyID="EUR">1000</cbc:BaseAmount>`, `<cbc:BaseAmount currencyID="EUR">invalid-amount</cbc:BaseAmount>`)
		doc, err := ubl.Parse([]byte(invalidXML))
		assert.NoError(t, err)

		ui, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		_, err = ui.Convert()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid major number")
	})
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

// TestParseSubInvoiceLinesOtherInvoice covers line billing references naming
// another invoice: EXTENDED-CTC-FR only reads a line's type and parent from
// the one naming the invoice itself.
func TestParseSubInvoiceLinesOtherInvoice(t *testing.T) {
	data, err := testLoadXML("sub-invoice-lines.xml")
	require.NoError(t, err)
	xml := string(data)

	// The bundle's DETAIL line also refers to an earlier invoice.
	other := "    <cac:BillingReference>\n      <cac:InvoiceDocumentReference>\n        <cbc:ID>FAC-2023-099</cbc:ID>\n      </cac:InvoiceDocumentReference>\n    </cac:BillingReference>\n"
	i := strings.Index(xml, "<cbc:ID>5</cbc:ID>")
	j := i + strings.Index(xml[i:], "    <cac:BillingReference>")
	xml = xml[:j] + other + xml[j:]

	// The INFORMATION status names another invoice, so the line is an ordinary
	// one, and with no price it is not converted.
	k := strings.LastIndex(xml, "<cbc:ID>FAC-2024-001</cbc:ID>")
	xml = xml[:k] + "<cbc:ID>FAC-2023-099</cbc:ID>" + xml[k+len("<cbc:ID>FAC-2024-001</cbc:ID>"):]

	inv := parseSubLines(t, xml)
	require.Len(t, inv.Lines, 4)
	bundle := inv.Lines[3]
	assert.Equal(t, "Service bundle", bundle.Item.Name)
	require.Len(t, bundle.Breakdown, 1)
	assert.Equal(t, "Installation", bundle.Breakdown[0].Item.Name)
	assert.False(t, inv.HasTags(tax.TagBypass))
}
