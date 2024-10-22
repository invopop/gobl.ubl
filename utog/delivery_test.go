package utog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUtoGDelivery(t *testing.T) {
	t.Run("ubl-example2.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("ubl-example2.xml")
		require.NoError(t, err)

		conversor := NewConversor()
		err = conversor.NewInvoice(doc)
		require.NoError(t, err)

		inv := conversor.GetInvoice()
		assert.NotNil(t, inv.Delivery)
	})

	t.Run("ubl-example4.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("ubl-example4.xml")
		require.NoError(t, err)

		conversor := NewConversor()
		err = conversor.NewInvoice(doc)
		require.NoError(t, err)

		inv := conversor.GetInvoice()
		assert.NotNil(t, inv.Delivery)
	})

	t.Run("ubl-example5.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("ubl-example5.xml")
		require.NoError(t, err)

		conversor := NewConversor()
		err = conversor.NewInvoice(doc)
		require.NoError(t, err)

		inv := conversor.GetInvoice()
		assert.NotNil(t, inv.Delivery)
	})

}
