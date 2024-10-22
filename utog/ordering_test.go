package utog

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUtoGOrdering(t *testing.T) {
	t.Run("UBL_example1.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("UBL_example1.xml")
		require.NoError(t, err)

		invoice := &bill.Invoice{}
		ordering := ParseUtoGOrdering(invoice, doc)

		require.NotNil(t, ordering, "Ordering should not be nil")
		assert.Equal(t, "AEG012345", string(ordering.Code), "Order reference should match")
	})

	t.Run("UBL_example2.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("UBL_example2.xml")
		require.NoError(t, err)

		invoice := &bill.Invoice{}
		ordering := ParseUtoGOrdering(invoice, doc)

		require.NotNil(t, ordering, "Ordering should not be nil")
		assert.Equal(t, "5009567", string(ordering.Code), "Order reference should match")
		require.NotNil(t, ordering.Period, "OrderingPeriod should not be nil")
		assert.Equal(t, "2005-06-20", ordering.Period.Start.String(), "OrderingPeriod start date should match")
		assert.Equal(t, "2005-06-21", ordering.Period.End.String(), "OrderingPeriod end date should match")
	})
}
