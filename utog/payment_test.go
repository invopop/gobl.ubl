package utog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPayment(t *testing.T) {
	// Read the XML file
	doc, err := LoadTestXMLDoc("invoice-test-4.xml")
	require.NoError(t, err)

	conversor := NewConversor()
	err = conversor.NewInvoice(doc)
	require.NoError(t, err)
	payment := conversor.GetInvoice().Payment
	assert.NotNil(t, payment)

}
