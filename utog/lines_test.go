package utog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define tests for the ParseXMLLines function
func TestGetLines(t *testing.T) {
	// Basic Invoice 1
	t.Run("UBL_example1.xml", func(t *testing.T) {
		doc, err := LoadTestXMLDoc("UBL_example1.xml")
		require.NoError(t, err)

		conversor := NewConversor()
		err = conversor.NewInvoice(doc)
		require.NoError(t, err)

		inv := conversor.GetInvoice()
		lines := inv.Lines
		assert.NotNil(t, lines)
		require.Len(t, lines, 2)

	})

}
