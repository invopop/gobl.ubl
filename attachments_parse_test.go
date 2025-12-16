package ubl_test

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAttachments(t *testing.T) {
	t.Run("ubl-example2.xml", func(t *testing.T) {
		e, err := testParseInvoice("ubl-example2.xml")
		require.NoError(t, err)

		inv, ok := e.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.NotNil(t, inv.Attachments)
		// Only external reference attachments are parsed without a binaryHandler
		// The embedded document (Doc2) is skipped
		require.Len(t, inv.Attachments, 1)

		// First attachment - external reference
		// Note: When FileName is not present in ExternalReference, Code is moved to Name
		att1 := inv.Attachments[0]
		assert.Equal(t, cbc.Code("Doc1"), att1.Code)
		assert.Equal(t, "Timesheet", att1.Description)
		assert.Equal(t, "http://www.suppliersite.eu/sheet001.html", att1.URL)
	})
}
