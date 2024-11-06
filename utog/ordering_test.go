package utog

import (
	"testing"

	"github.com/invopop/gobl/cbc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrdering(t *testing.T) {
	t.Run("ubl-example2.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("ubl-example2.xml")
		require.NoError(t, err)

		converter := NewConverter()
		err = converter.NewInvoice(doc)
		require.NoError(t, err)
		ordering := converter.GetInvoice().Ordering
		assert.NotNil(t, ordering)

		assert.Equal(t, cbc.Code("123"), ordering.Code)
		assert.Equal(t, "2013-06-01", ordering.Period.Start.String())
		assert.Equal(t, "2013-06-30", ordering.Period.End.String())
		assert.Equal(t, cbc.Code("Contract321"), ordering.Contracts[0].Code)
	})

	t.Run("ubl-example5.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("ubl-example5.xml")
		require.NoError(t, err)

		converter := NewConverter()
		err = converter.NewInvoice(doc)
		require.NoError(t, err)
		ordering := converter.GetInvoice().Ordering
		assert.NotNil(t, ordering)

		assert.Equal(t, cbc.Code("PO4711"), ordering.Code)
		assert.Equal(t, "2013-03-10", ordering.Period.Start.String())
		assert.Equal(t, "2013-04-10", ordering.Period.End.String())
		assert.Equal(t, cbc.Code("2013-05"), ordering.Contracts[0].Code)
		assert.Equal(t, cbc.Code("3544"), ordering.Receiving[0].Code)
		assert.Equal(t, cbc.Code("5433"), ordering.Despatch[0].Code)
	})

}
