package ubl

import (
	"github.com/invopop/gobl/org"
	"github.com/nbio/xml"
)

// ParseAttachments extracts attachments from a UBL document and converts them
// to GOBL attachments using the provided handler function.
// The handler is called for each attachment that has either a URL or base64 data.
// If the handler returns an error, parsing stops immediately.
func ParseAttachments(data []byte, handler func(*Reference) (*org.Attachment, error)) ([]*org.Attachment, error) {
	in := new(Invoice)
	if err := xml.Unmarshal(data, in); err != nil {
		return nil, err
	}

	var attachments []*org.Attachment

	// Process AdditionalDocumentReference
	for i := range in.AdditionalDocumentReference {
		ref := &in.AdditionalDocumentReference[i]
		if att, err := processReference(ref, handler); err != nil {
			return nil, err
		} else if att != nil {
			attachments = append(attachments, att)
		}
	}

	return attachments, nil
}

// processReference checks if a reference has a valid attachment and processes it
func processReference(ref *Reference, handler func(*Reference) (*org.Attachment, error)) (*org.Attachment, error) {
	if ref == nil || ref.Attachment == nil {
		return nil, nil
	}

	if !hasValidAttachment(ref.Attachment) {
		return nil, nil
	}

	return handler(ref)
}

// hasValidAttachment checks if an attachment has either a URL or base64 data
func hasValidAttachment(att *Attachment) bool {
	hasURL := att.ExternalReference.URI != ""
	hasBase64 := att.EmbeddedDocumentBinaryObject.Value != ""
	return hasURL || hasBase64
}
