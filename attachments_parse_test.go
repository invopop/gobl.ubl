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
		e := parseXMLInvoice(t, "en16931/ubl-example2.xml")

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
		data, err := testLoadXML("en16931/ubl-example2.xml")
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

func TestParseEmbeddedAndExternalAttachment(t *testing.T) {
	load := func(t *testing.T, embedded string) *ubl.Invoice {
		t.Helper()
		data, err := testLoadXML("en16931/ubl-example2.xml")
		require.NoError(t, err)
		doc, err := ubl.Parse(data)
		require.NoError(t, err)
		inv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// Doc2 carries both an embedded binary and an external URI
		ref := &inv.AdditionalDocumentReference[1]
		require.Equal(t, "Doc2", ref.ID.Value)
		ref.Attachment.EmbeddedDocumentBinaryObject.Value = embedded
		ref.Attachment.ExternalReference = &ubl.ExternalReference{URI: "https://example.com/doc2.pdf"}
		return inv
	}

	t.Run("valid embedded is picked", func(t *testing.T) {
		inv := load(t, "VGVzdGluZyBCYXNlNjQgZW5jb2Rpbmc=")

		env, err := inv.Convert()
		require.NoError(t, err)
		gi, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		require.Len(t, gi.Attachments, 1)
		assert.Equal(t, cbc.Code("Doc1"), gi.Attachments[0].Code)

		bins := inv.ExtractBinaryAttachments()
		require.Len(t, bins, 1)
		assert.Equal(t, "Doc2", bins[0].ID)
		assert.Equal(t, []byte("Testing Base64 encoding"), bins[0].Data)
	})

	for name, embedded := range map[string]string{"empty": "", "corrupted": "!!not base64!!"} {
		t.Run(name+" embedded falls back to URI", func(t *testing.T) {
			inv := load(t, embedded)

			env, err := inv.Convert()
			require.NoError(t, err)
			gi, ok := env.Extract().(*bill.Invoice)
			require.True(t, ok)
			require.Len(t, gi.Attachments, 2)
			assert.Equal(t, cbc.Code("Doc2"), gi.Attachments[1].Code)
			assert.Equal(t, "https://example.com/doc2.pdf", gi.Attachments[1].URL)

			assert.Empty(t, inv.ExtractBinaryAttachments())
		})
	}
}
