package ubl

import (
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/dsig"
	"github.com/invopop/gobl/org"
)

// goblAddAttachments processes the attachment in the given reference.
// Binary attachments are now ignored - use ExtractBinaryAttachments instead.
func goblAddAttachments(ref Reference, o *options) (*org.Attachment, error) {
	if ref.Attachment == nil {
		return nil, nil
	}

	var att *org.Attachment

	switch {
	case ref.Attachment.ExternalReference != nil:
		extRef := ref.Attachment.ExternalReference
		att = &org.Attachment{
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
	case ref.Attachment.EmbeddedDocumentBinaryObject != nil:
		// Skip binary attachments - they should be extracted using ExtractBinaryAttachments
		return nil, nil
	}

	if att != nil {
		att.Code = cbc.Code(ref.ID.Value)
		att.Description = ref.DocumentDescription
		// Ensure name is set as GOBL validates this.
		// This will still be mapped to code if converted back into UBL
		if att.Name == "" {
			att.Name = att.Code.String()
			att.Code = ""
		}
	}

	return att, nil
}

// BinaryAttachment represents a binary attachment extracted from a UBL invoice.
type BinaryAttachment struct {
	ID          string
	Description string
	Binary      *BinaryObject
}

// ExtractBinaryAttachments extracts all binary attachments from the UBL Invoice.
// It returns a slice of BinaryAttachment containing the ID, description, and binary data.
// External reference attachments are not included in the result.
func (in *Invoice) ExtractBinaryAttachments() []BinaryAttachment {
	var result []BinaryAttachment

	for _, ref := range in.AdditionalDocumentReference {
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
