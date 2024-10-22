package utog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCtoGPayment(t *testing.T) {
	// Read the XML file
	doc, err := LoadTestXMLDoc("invoice-test-4.xml")
	require.NoError(t, err)

	payment := ctog.ParseCtoGPayment(&doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement)

	assert.NotNil(t, payment)

}
