package ubl

import "github.com/invopop/gobl/bill"

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

func (out *Invoice) addOrdering(o *bill.Ordering) {
	if o == nil {
		return
	}

	if o.Code != "" {
		out.BuyerReference = o.Code.String()
	}

	// If both ordering.seller and seller are present, the original seller is used
	// as the tax representative.
	if o.Seller != nil {
		p := out.AccountingSupplierParty.Party
		out.TaxRepresentativeParty = p
		out.AccountingSupplierParty = SupplierParty{
			Party: newParty(o.Seller),
		}
	}

	if o.Period != nil {
		start := formatDate(o.Period.Start)
		end := formatDate(o.Period.End)
		out.InvoicePeriod = []Period{
			{
				StartDate: &start,
				EndDate:   &end,
			},
		}
	}

	if len(o.Despatch) > 0 {
		out.DespatchDocumentReference = make([]Reference, 0, len(o.Despatch))
		for _, despatch := range o.Despatch {
			out.DespatchDocumentReference = append(out.DespatchDocumentReference, Reference{
				ID: IDType{Value: string(despatch.Code)},
			})
		}
	}

	if len(o.Receiving) > 0 {
		out.ReceiptDocumentReference = make([]Reference, 0, len(o.Receiving))
		for _, receiving := range o.Receiving {
			out.ReceiptDocumentReference = append(out.ReceiptDocumentReference, Reference{
				ID: IDType{Value: string(receiving.Code)},
			})
		}
	}

	if len(o.Contracts) > 0 {
		out.ContractDocumentReference = make([]Reference, 0, len(o.Contracts))
		for _, contract := range o.Contracts {
			out.ContractDocumentReference = append(out.ContractDocumentReference, Reference{
				ID: IDType{Value: string(contract.Code)},
			})
		}
	}

	if len(o.Tender) > 0 {
		out.AdditionalDocumentReference = make([]Reference, 0, len(o.Tender))
		for _, tender := range o.Tender {
			out.AdditionalDocumentReference = append(out.AdditionalDocumentReference, Reference{
				ID: IDType{Value: string(tender.Code)},
			})
		}
	}

	if len(o.Purchases) > 0 {
		purchase := o.Purchases[0]
		out.OrderReference = &OrderReference{
			ID: purchase.Code.String(),
		}
	}

	// done
}
