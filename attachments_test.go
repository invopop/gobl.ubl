package ubl_test

import (
	"testing"

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
}
