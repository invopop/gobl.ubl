package ubl

import (
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/dsig"
	"github.com/invopop/gobl/org"
)

// BinaryAttachment represents a binary attachment extracted from a UBL invoice.
type BinaryAttachment struct {
	ID          string
	Description string
	Binary      *BinaryObject
}

// goblAddAttachments processes all attachments from the UBL Invoice and returns
// external reference attachments only.
// Binary attachments are skipped - use ExtractBinaryAttachments to retrieve them.
func (ui *Invoice) goblAddAttachments() []*org.Attachment {
	var attachments []*org.Attachment

	for _, ref := range ui.AdditionalDocumentReference {
		if ref.Attachment == nil {
			continue
		}

		// Only process external reference attachments
		if ref.Attachment.ExternalReference != nil {
			att := ui.processExternalAttachment(&ref)
			if att != nil {
				attachments = append(attachments, att)
			}
		}
		// Binary attachments are skipped - handled by ExtractBinaryAttachments
	}

	return attachments
}

// processExternalAttachment converts a UBL external reference attachment to GOBL format.
func (ui *Invoice) processExternalAttachment(ref *Reference) *org.Attachment {
	extRef := ref.Attachment.ExternalReference
	if extRef == nil {
		return nil
	}

	att := &org.Attachment{
		URL:  extRef.URI,
		MIME: extRef.MimeCode,
		Name: extRef.FileName,
	}

	if extRef.DocumentHash != "" && extRef.HashAlgorithmMethod != "" {
		att.Digest = &dsig.Digest{
			Value:     extRef.DocumentHash,
			Algorithm: dsig.DigestAlgorithm(extRef.HashAlgorithmMethod),
		}
	}

	att.Code = cbc.Code(ref.ID.Value)
	att.Description = ref.DocumentDescription

	return att
}

// ExtractBinaryAttachments extracts all binary attachments from the UBL Invoice.
// It returns a slice of BinaryAttachment containing the ID, description, and binary data.
// External reference attachments are not included in the result.
func (ui *Invoice) ExtractBinaryAttachments() []BinaryAttachment {
	var result []BinaryAttachment

	for _, ref := range ui.AdditionalDocumentReference {
		if ref.Attachment != nil && ref.Attachment.EmbeddedDocumentBinaryObject != nil {
			result = append(result, BinaryAttachment{
				ID:          ref.ID.Value,
				Description: ref.DocumentDescription,
				Binary:      ref.Attachment.EmbeddedDocumentBinaryObject,
			})
		}
	}

	return result
}
