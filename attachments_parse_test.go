package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
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

func TestExtractBinaryAttachments(t *testing.T) {
	t.Run("ubl-example2.xml", func(t *testing.T) {
		data, err := testLoadXML("ubl-example2.xml")
		require.NoError(t, err)

		doc, err := ubl.Parse(data)
		require.NoError(t, err)

		inv, ok := doc.(*ubl.Invoice)
		require.True(t, ok, "Expected an Invoice document")

		// Extract binary attachments
		attachments := inv.ExtractBinaryAttachments()
		require.Len(t, attachments, 1)

		// Verify the binary attachment
		att := attachments[0]
		assert.Equal(t, "Doc2", att.ID)
		assert.Equal(t, "EHF specification", att.Description)
		assert.Equal(t, "application/pdf", att.MimeCode)
		assert.Equal(t, "test.pdf", att.Filename)

		// Verify the data is correctly decoded
		// "VGVzdGluZyBCYXNlNjQgZW5jb2Rpbmc=" decodes to "Testing Base64 encoding"
		expectedData := []byte("Testing Base64 encoding")
		assert.Equal(t, expectedData, att.Data)
	})
}
