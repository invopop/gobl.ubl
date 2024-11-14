package document

// Period represents a time period with start and end dates
type Period struct {
	StartDate *string `xml:"cbc:StartDate"`
	EndDate   *string `xml:"cbc:EndDate"`
}

// OrderReference represents a reference to an order
type OrderReference struct {
	ID                string  `xml:"cbc:ID"`
	SalesOrderID      *string `xml:"cbc:SalesOrderID"`
	IssueDate         *string `xml:"cbc:IssueDate"`
	CustomerReference *string `xml:"cbc:CustomerReference"`
}

// BillingReference represents a reference to a billing document
type BillingReference struct {
	InvoiceDocumentReference           *Reference `xml:"cac:InvoiceDocumentReference"`
	SelfBilledInvoiceDocumentReference *Reference `xml:"cac:SelfBilledInvoiceDocumentReference"`
	CreditNoteDocumentReference        *Reference `xml:"cac:CreditNoteDocumentReference"`
	AdditionalDocumentReference        *Reference `xml:"cac:AdditionalDocumentReference"`
}

// Reference represents a reference to a document
type Reference struct {
	ID                  IDType      `xml:"cbc:ID"`
	IssueDate           *string     `xml:"cbc:IssueDate"`
	DocumentTypeCode    *string     `xml:"cbc:DocumentTypeCode"`
	DocumentType        *string     `xml:"cbc:DocumentType"`
	Attachment          *Attachment `xml:"cac:Attachment"`
	DocumentDescription *string     `xml:"cbc:DocumentDescription"`
	ValidityPeriod      *Period     `xml:"cac:ValidityPeriod"`
}

// Attachment represents an attached document
type Attachment struct {
	EmbeddedDocumentBinaryObject BinaryObject `xml:"cbc:EmbeddedDocumentBinaryObject"`
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

// ProjectReference represents a reference to a project
type ProjectReference struct {
	ID *string `xml:"cbc:ID"`
}
