package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseDeclaredTotals covers reconciliation against the declared amounts.
// BT-131 is mandatory and the totals are summed from it, so it decides.
func TestParseDeclaredTotals(t *testing.T) {
	// A multiplier its base (BT-137) reproduces: the line reconciles.
	t.Run("line allowance with a declared base", func(t *testing.T) {
		e := parseXMLInvoice(t, "en16931/line-allowance-base.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)
		assert.False(t, inv.HasTags(tax.TagBypass))

		require.Len(t, inv.Lines, 1)
		// 200 x 10.00, less 10% of the declared 1000.00 base.
		assert.Equal(t, "2000.00", inv.Lines[0].Sum.String())
		assert.Equal(t, "1900.00", inv.Lines[0].Total.String())
		assert.Equal(t, "1900.00", inv.Totals.Sum.String())
		assert.Equal(t, "2280.00", inv.Totals.Payable.String())
	})

	// Reproducible under no reading: the sender's figures are kept.
	t.Run("line totals that cannot be reproduced", func(t *testing.T) {
		e := parseXMLInvoice(t, "en16931/line-totals-mismatch.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)
		assert.True(t, inv.HasTags(tax.TagBypass))

		require.Len(t, inv.Lines, 1)
		assert.Equal(t, "1614.87", inv.Lines[0].Total.String())
		assert.Equal(t, "1614.87", inv.Totals.Sum.String())
		assert.Equal(t, "1937.84", inv.Totals.Payable.String())
	})

	// The type tags are what regenerate BT-3, so bypassing the totals must
	// not drop them, or the document type quietly falls back to 380.
	for _, tt := range []struct {
		code cbc.Code
		tag  cbc.Key
	}{
		{"389", tax.TagSelfBilled},
		{"393", tax.TagFactoring},
	} {
		t.Run("totals that cannot be reproduced keep "+tt.code.String(), func(t *testing.T) {
			inv := parseWithTypeCode(t, "en16931/line-totals-mismatch.xml",
				"<cbc:InvoiceTypeCode>380</cbc:InvoiceTypeCode>",
				"<cbc:InvoiceTypeCode>"+tt.code.String()+"</cbc:InvoiceTypeCode>")

			assert.True(t, inv.HasTags(tax.TagBypass))
			assert.True(t, inv.HasTags(tt.tag), "missing tag %s", tt.tag)
			assert.Equal(t, tt.code, inv.Tax.Ext.Get(untdid.ExtKeyDocumentType))
		})
	}

	// Format import funcs run before the totals are reconciled, so the tags
	// they set must survive the bypass too.
	importWithTags := func(t *testing.T, tags []cbc.Key) *bill.Invoice {
		t.Helper()
		f := ubl.FormatEN16931
		f.ImportFuncs = []ubl.ImportFunc{
			func(_ *ubl.Format, _ ubl.Document, env *gobl.Envelope) error {
				env.Extract().(*bill.Invoice).SetTags(tags...)
				return nil
			},
		}
		data, err := testLoadXML("en16931/line-totals-mismatch.xml")
		require.NoError(t, err)
		doc, err := ubl.Decode(data)
		require.NoError(t, err)
		env, err := ubl.Import(doc, ubl.WithFormat(f))
		require.NoError(t, err)
		return env.Extract().(*bill.Invoice)
	}

	t.Run("totals that cannot be reproduced keep a format's tags", func(t *testing.T) {
		// Spare capacity: adding the bypass tag must not write into it.
		tags := make([]cbc.Key, 1, 4)
		tags[0] = tax.TagPrepayment
		inv := importWithTags(t, tags)

		assert.Equal(t, []cbc.Key{tax.TagPrepayment, tax.TagBypass}, inv.GetTags())
		assert.Equal(t, cbc.Code("386"), inv.Tax.Ext.Get(untdid.ExtKeyDocumentType))
		assert.Empty(t, tags[:2][1], "the format's slice was written to")
	})
}

// TestLineAllowanceBaseRoundTrip guards PEPPOL-EN16931-R040: the amount must
// equal base x percentage, so the sender's base has to survive the round trip.
func TestLineAllowanceBaseRoundTrip(t *testing.T) {
	e := parseXMLInvoice(t, "en16931/line-allowance-base.xml")

	inv, ok := e.Extract().(*bill.Invoice)
	require.True(t, ok)
	require.Len(t, inv.Lines, 1)
	require.Len(t, inv.Lines[0].Discounts, 1)

	d := inv.Lines[0].Discounts[0]
	require.NotNil(t, d.Base)
	// The declared basis, not the 2000.00 line sum.
	assert.Equal(t, "1000.00", d.Base.String())
	require.NotNil(t, d.Percent)
	assert.Equal(t, "10.00%", d.Percent.String())
	assert.Equal(t, "100.00", d.Amount.String())

	out, err := ubl.ExportInvoice(e)
	require.NoError(t, err)
	require.Len(t, out.InvoiceLines, 1)
	require.Len(t, out.InvoiceLines[0].AllowanceCharge, 1)

	ac := out.InvoiceLines[0].AllowanceCharge[0]
	require.NotNil(t, ac.BaseAmount)
	assert.Equal(t, "1000.00", ac.BaseAmount.Value)
	require.NotNil(t, ac.MultiplierFactorNumeric)
	assert.Equal(t, "10.00", *ac.MultiplierFactorNumeric)
	assert.Equal(t, "100.00", ac.Amount.Value)
}
