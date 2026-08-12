package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOrdering(t *testing.T) {
	t.Run("ubl-example2.xml", func(t *testing.T) {
		e := parseXMLInvoice(t, "en16931/ubl-example2.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		ordering := inv.Ordering
		assert.NotNil(t, ordering)

		assert.Equal(t, "2013-06-01", ordering.Period.Start.String())
		assert.Equal(t, "2013-06-30", ordering.Period.End.String())
		assert.Equal(t, cbc.Code("Contract321"), ordering.Contracts[0].Code)
	})

	t.Run("ubl-example5.xml", func(t *testing.T) {
		e := parseXMLInvoice(t, "en16931/ubl-example5.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		ordering := inv.Ordering
		assert.NotNil(t, ordering)

		assert.Equal(t, cbc.Code("123"), ordering.Code)
		assert.Equal(t, "2013-03-10", ordering.Period.Start.String())
		assert.Equal(t, "2013-04-10", ordering.Period.End.String())
		assert.Equal(t, cbc.Code("2013-05"), ordering.Contracts[0].Code)
		assert.Equal(t, cbc.Code("PO4711"), ordering.Purchases[0].Code)
		assert.Equal(t, cbc.Code("3544"), ordering.Receiving[0].Code)
		assert.Equal(t, cbc.Code("5433"), ordering.Despatch[0].Code)
	})

}

func TestParseOrderingNotApplicable(t *testing.T) {
	t.Run("NA is not a purchase order", func(t *testing.T) {
		e := parseXMLInvoice(t, "peppol/sales-order-example.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.NotNil(t, inv.Ordering)
		assert.Empty(t, inv.Ordering.Purchases)
	})

	t.Run("a sales order alongside NA survives", func(t *testing.T) {
		e := parseXMLInvoice(t, "peppol/sales-order-example.xml")

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.NotNil(t, inv.Ordering)
		require.Len(t, inv.Ordering.Sales, 1)
		assert.Equal(t, cbc.Code("123456"), inv.Ordering.Sales[0].Code)
	})

	t.Run("an invoice with no ordering gains none on a round trip", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")

		doc, err := ubl.ConvertInvoice(env, ubl.WithContext(ubl.ContextPeppol))
		require.NoError(t, err)
		require.Equal(t, "NA", doc.OrderReference.ID)

		data, err := ubl.Bytes(doc)
		require.NoError(t, err)

		out, err := ubl.Parse(data)
		require.NoError(t, err)
		parsed, ok := out.(*ubl.Invoice)
		require.True(t, ok)

		env2, err := parsed.Convert()
		require.NoError(t, err)
		inv, ok := env2.Extract().(*bill.Invoice)
		require.True(t, ok)

		if inv.Ordering != nil {
			assert.Empty(t, inv.Ordering.Purchases)
		}
	})
}
