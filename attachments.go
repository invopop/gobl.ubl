package ubl

import "github.com/invopop/gobl/org"

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
		idValue := a.Code.String()
		if idValue == "" {
			idValue = a.Name
		}

		ref := Reference{
			ID: IDType{
				Value: idValue,
			},
		}

		ref.DocumentDescription = a.Description

		if a.URL != "" || a.Digest != nil || a.MIME != "" || a.Name != "" {
			extRef := &ExternalReference{
				Description: a.Description,
				URI:         a.URL,
				MimeCode:    a.MIME,
				FileName:    a.Name,
			}

			if a.Digest != nil {
				extRef.DocumentHash = a.Digest.Value
				extRef.HashAlgorithmMethod = string(a.Digest.Algorithm)
			}

			ref.Attachment = &Attachment{
				ExternalReference: extRef,
			}
		}

		ui.AdditionalDocumentReference = append(ui.AdditionalDocumentReference, ref)
	}
}
