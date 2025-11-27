package ubl

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/org"
)

// Period represents a time period with start and end dates
type Period struct {
	StartDate string `xml:"cbc:StartDate,omitempty"`
	EndDate   string `xml:"cbc:EndDate,omitempty"`
}

// OrderReference represents a reference to an order
type OrderReference struct {
	ID                string `xml:"cbc:ID"`
	SalesOrderID      string `xml:"cbc:SalesOrderID,omitempty"`
	IssueDate         string `xml:"cbc:IssueDate,omitempty"`
	CustomerReference string `xml:"cbc:CustomerReference,omitempty"`
}

// BillingReference represents a reference to a billing document
type BillingReference struct {
	InvoiceDocumentReference           *Reference `xml:"cac:InvoiceDocumentReference,omitempty"`
	SelfBilledInvoiceDocumentReference *Reference `xml:"cac:SelfBilledInvoiceDocumentReference,omitempty"`
	CreditNoteDocumentReference        *Reference `xml:"cac:CreditNoteDocumentReference,omitempty"`
	AdditionalDocumentReference        *Reference `xml:"cac:AdditionalDocumentReference,omitempty"`
}

// Reference represents a reference to a document
type Reference struct {
	ID                  IDType      `xml:"cbc:ID"`
	IssueDate           string      `xml:"cbc:IssueDate,omitempty"`
	DocumentTypeCode    string      `xml:"cbc:DocumentTypeCode,omitempty"`
	DocumentType        string      `xml:"cbc:DocumentType,omitempty"`
	DocumentDescription string      `xml:"cbc:DocumentDescription,omitempty"`
	Attachment          *Attachment `xml:"cac:Attachment,omitempty"`
	ValidityPeriod      *Period     `xml:"cac:ValidityPeriod,omitempty"`
}

// ProjectReference represents a reference to a project
type ProjectReference struct {
	ID string `xml:"cbc:ID,omitempty"`
}

func (ui *Invoice) addPreceding(refs []*org.DocumentRef) {
	if len(refs) == 0 {
		return
	}
	ui.BillingReference = make([]*BillingReference, len(refs))
	for i, ref := range refs {
		r := &Reference{
			ID: IDType{Value: ref.Series.Join(ref.Code).String()},
		}
		if ref.IssueDate != nil {
			r.IssueDate = ref.IssueDate.String()
		}
		if dt := ref.Ext.Get(untdid.ExtKeyDocumentType); dt != "" {
			r.DocumentTypeCode = dt.String()
		}
		ui.BillingReference[i] = &BillingReference{
			InvoiceDocumentReference: r,
		}
	}
}

func (ui *Invoice) addOrdering(o *bill.Ordering) {
	if o == nil {
		return
	}

	if o.Code != "" {
		ui.BuyerReference = o.Code.String()
	}

	// If both ordering.seller and seller are present, the original seller is used
	// as the tax representative.
	if o.Seller != nil {
		p := ui.AccountingSupplierParty.Party
		ui.TaxRepresentativeParty = p
		ui.AccountingSupplierParty = SupplierParty{
			Party: newParty(o.Seller),
		}
	}

	if o.Period != nil {
		ui.InvoicePeriod = []Period{
			{
				StartDate: formatDate(o.Period.Start),
				EndDate:   formatDate(o.Period.End),
			},
		}
	}

	if len(o.Purchases) > 0 {
		purchase := o.Purchases[0]
		ui.OrderReference = &OrderReference{
			ID: purchase.Code.String(),
		}
	}

	for _, despatch := range o.Despatch {
		ui.DespatchDocumentReference = append(ui.DespatchDocumentReference, Reference{
			ID: IDType{Value: string(despatch.Code)},
		})
	}

	for _, receiving := range o.Receiving {
		ui.ReceiptDocumentReference = append(ui.ReceiptDocumentReference, Reference{
			ID: IDType{Value: string(receiving.Code)},
		})
	}

	for _, contract := range o.Contracts {
		ui.ContractDocumentReference = append(ui.ContractDocumentReference, Reference{
			ID: IDType{Value: string(contract.Code)},
		})
	}

	for _, tender := range o.Tender {
		ui.OriginatorDocumentReference = append(ui.OriginatorDocumentReference, Reference{
			ID: IDType{Value: string(tender.Code)},
		})
	}

	if len(o.Identities) > 0 {
		ioi := o.Identities[0]

		for _, id := range o.Identities {
			if id.Ext.Has(untdid.ExtKeyReference) {
				ioi = id
				break
			}
		}

		id := IDType{Value: string(ioi.Code)}
		if ref := ioi.Ext.Get(untdid.ExtKeyReference); ref != "" {
			schemeID := ref.String()
			id.SchemeID = &schemeID
		}
		ui.AdditionalDocumentReference = append(ui.AdditionalDocumentReference, Reference{
			ID:               id,
			DocumentTypeCode: "130",
		})
	}
}
