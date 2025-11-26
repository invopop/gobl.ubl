package ubl

import (
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/dsig"
	"github.com/invopop/gobl/org"
)

// goblAddAttachments processes the attachment in the given reference.
// Binary attachments are delegated to the provided options handler if available.
func goblAddAttachments(ref Reference, o *options) (att *org.Attachment, err error) {
	if ref.Attachment == nil {
		return
	}

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
		if o == nil || o.binaryHandler == nil {
			return
		}

		att, err = o.binaryHandler(ref.Attachment.EmbeddedDocumentBinaryObject)
		if err != nil {
			return
		}
	default:
		return
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

	return
}
