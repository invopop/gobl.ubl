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

	conversor := NewConversor()
	inv, err := conversor.NewInvoice(doc)
	require.NoError(t, err)
	payment := inv.Payment
	assert.NotNil(t, payment)

}
