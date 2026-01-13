package ubl

import (
	"encoding/base64"

	"github.com/invopop/gobl/org"
)

// Attachment represents an attached document
type Attachment struct {
	ExternalReference            *ExternalReference `xml:"cac:ExternalReference,omitempty"`
	EmbeddedDocumentBinaryObject *BinaryObject      `xml:"cbc:EmbeddedDocumentBinaryObject,omitempty"`
}

// BinaryObject represents binary data with associated metadata
type BinaryObject struct {
	MimeCode         *string `xml:"mimeCode,attr"`
	Filename         *string `xml:"filename,attr"`
	EncodingCode     *string `xml:"encodingCode,attr"`
	CharacterSetCode *string `xml:"characterSetCode,attr"`
	URI              *string `xml:"uri,attr"`
	Value            string  `xml:",chardata"`
}

// ExternalReference represents a reference to an external resource
type ExternalReference struct {
	URI                 string `xml:"cbc:URI,omitempty"`
	DocumentHash        string `xml:"cbc:DocumentHash,omitempty"`
	HashAlgorithmMethod string `xml:"cbc:HashAlgorithmMethod,omitempty"`
	ExpiryDate          string `xml:"cbc:ExpiryDate,omitempty"`
	ExpiryTime          string `xml:"cbc:ExpiryTime,omitempty"`
	MimeCode            string `xml:"cbc:MimeCode,omitempty"`
	FormatCode          string `xml:"cbc:FormatCode,omitempty"`
	EncodingCode        string `xml:"cbc:EncodingCode,omitempty"`
	CharacterSetCode    string `xml:"cbc:CharacterSetCode,omitempty"`
	FileName            string `xml:"cbc:FileName,omitempty"`
	Description         string `xml:"cbc:Description,omitempty"`
}

func (ui *Invoice) addAttachments(attachments []*org.Attachment) {
	for _, a := range attachments {
		ref := Reference{
			ID: IDType{
				Value: a.Code.String(),
			},
		}

		if a.Description != "" {
			ref.DocumentDescription = a.Description
		}

		extRef := &ExternalReference{
			URI: a.URL,
		}

		if a.MIME != "" {
			extRef.MimeCode = a.MIME
		}

		if a.Name != "" {
			extRef.FileName = a.Name
		}

		if a.Digest != nil {
			extRef.DocumentHash = a.Digest.Value
			extRef.HashAlgorithmMethod = string(a.Digest.Algorithm)
		}

		ref.Attachment = &Attachment{
			ExternalReference: extRef,
		}

		ui.AdditionalDocumentReference = append(ui.AdditionalDocumentReference, ref)
	}
}

// AddBinaryAttachment adds an embedded binary attachment to the UBL Invoice.
// This is useful for including documents like PDFs directly within the UBL XML.
// The binary data will be automatically base64-encoded.
func (ui *Invoice) AddBinaryAttachment(attachment BinaryAttachment) {
	ref := Reference{
		ID: IDType{
			Value: attachment.ID,
		},
	}

	if attachment.Description != "" {
		ref.DocumentDescription = attachment.Description
	}

	// Base64-encode the binary data
	encodedData := base64.StdEncoding.EncodeToString(attachment.Data)

	binaryObj := &BinaryObject{
		Value: encodedData,
	}

	if attachment.MimeCode != "" {
		binaryObj.MimeCode = &attachment.MimeCode
	}

	if attachment.Filename != "" {
		binaryObj.Filename = &attachment.Filename
	}

	if attachment.CharacterSetCode != "" {
		binaryObj.CharacterSetCode = &attachment.CharacterSetCode
	}

	if attachment.URI != "" {
		binaryObj.URI = &attachment.URI
	}

	ref.Attachment = &Attachment{
		EmbeddedDocumentBinaryObject: binaryObj,
	}

	ui.AdditionalDocumentReference = append(ui.AdditionalDocumentReference, ref)
}
