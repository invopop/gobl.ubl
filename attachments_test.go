package ubl_test

import (
	"encoding/base64"
	"testing"

	"github.com/invopop/gobl.ubl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttachments(t *testing.T) {
	t.Run("invoice-attachments.json", func(t *testing.T) {
		doc, err := testInvoiceFrom("invoice-attachments.json")
		require.NoError(t, err)

		require.Len(t, doc.AdditionalDocumentReference, 1)

		ref := doc.AdditionalDocumentReference[0]
		assert.Equal(t, "doc1", ref.ID.Value)
		assert.Equal(t, "test file", ref.DocumentDescription)

		require.NotNil(t, ref.Attachment)
		require.NotNil(t, ref.Attachment.ExternalReference)

		extRef := ref.Attachment.ExternalReference
		assert.Equal(t, "testfile.com/test.html", extRef.URI)
		assert.Equal(t, "application/pdf", extRef.MimeCode)
		assert.Equal(t, "test", extRef.FileName)
	})

	t.Run("add binary attachment", func(t *testing.T) {
		doc, err := testInvoiceFrom("invoice-minimal.json")
		require.NoError(t, err)

		// Sample binary data (e.g., a simple text file for testing)
		sampleData := []byte("This is a test document content")

		// Add a binary attachment to the generated UBL invoice
		doc.AddBinaryAttachment(ubl.BinaryAttachment{
			ID:          "attachment1",
			Description: "Sample PDF document",
			Data:        sampleData,
			MimeCode:    "application/pdf",
			Filename:    "sample.pdf",
		})

		// Find the attachment we just added
		var found bool
		for _, ref := range doc.AdditionalDocumentReference {
			if ref.ID.Value == "attachment1" {
				found = true
				assert.Equal(t, "Sample PDF document", ref.DocumentDescription)

				require.NotNil(t, ref.Attachment)
				require.NotNil(t, ref.Attachment.EmbeddedDocumentBinaryObject)

				binObj := ref.Attachment.EmbeddedDocumentBinaryObject
				// Verify the data is base64-encoded
				assert.NotEmpty(t, binObj.Value)
				assert.Equal(t, "application/pdf", *binObj.MimeCode)
				assert.Equal(t, "sample.pdf", *binObj.Filename)
				assert.Equal(t, "Base64", *binObj.EncodingCode)

				// Verify we can decode it back
				decoded, err := base64.StdEncoding.DecodeString(binObj.Value)
				require.NoError(t, err)
				assert.Equal(t, sampleData, decoded)
			}
		}

		require.True(t, found, "Binary attachment should be added to AdditionalDocumentReference")
	})

	t.Run("extract binary attachment roundtrip", func(t *testing.T) {
		doc, err := testInvoiceFrom("invoice-minimal.json")
		require.NoError(t, err)

		// Create and add a binary attachment
		originalData := []byte("Test PDF content with special chars: üöä")
		doc.AddBinaryAttachment(ubl.BinaryAttachment{
			ID:          "test-pdf",
			Description: "Test document",
			Data:        originalData,
			MimeCode:    "application/pdf",
			Filename:    "test.pdf",
			URI:         "http://example.com/test.pdf",
		})

		// Extract binary attachments
		attachments := doc.ExtractBinaryAttachments()

		// Find our attachment
		var found bool
		for _, att := range attachments {
			if att.ID == "test-pdf" {
				found = true
				assert.Equal(t, "Test document", att.Description)
				assert.Equal(t, originalData, att.Data)
				assert.Equal(t, "application/pdf", att.MimeCode)
				assert.Equal(t, "test.pdf", att.Filename)
				assert.Equal(t, "http://example.com/test.pdf", att.URI)
			}
		}

		require.True(t, found, "Should be able to extract the binary attachment we added")
	})
}
