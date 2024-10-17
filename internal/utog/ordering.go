package ubl

import (
	"github.com/invopop/gobl.ubl/structs"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

func ParseUtoGOrdering(inv *bill.Invoice, doc *structs.Invoice) *bill.Ordering {
	ordering := &bill.Ordering{}

	if doc.OrderReference != nil && doc.OrderReference.ID != "" {
		ordering.Code = cbc.Code(doc.OrderReference.ID)
	}

	if doc.InvoicePeriod != nil {
		period := &cal.Period{}

		if doc.InvoicePeriod.StartDate != "" {
			period.Start = ParseDate(doc.InvoicePeriod.StartDate)
		}

		if doc.InvoicePeriod.EndDate != "" {
			period.End = ParseDate(doc.InvoicePeriod.EndDate)
		}

		ordering.Period = period
	}

	if doc.DespatchDocumentReference != nil {
		ordering.Despatch = []*org.DocumentRef{
			{
				Code: cbc.Code(doc.DespatchDocumentReference.ID),
			},
		}
		if doc.DespatchDocumentReference.IssueDate != "" {
			refDate := ParseDate(doc.DespatchDocumentReference.IssueDate)
			ordering.Despatch[0].IssueDate = &refDate
		}
	}

	if doc.ReceiptDocumentReference != nil {
		ordering.Receiving = []*org.DocumentRef{
			{
				Code: cbc.Code(doc.ReceiptDocumentReference.ID),
			},
		}
		if doc.ReceiptDocumentReference.IssueDate != "" {
			refDate := ParseDate(doc.ReceiptDocumentReference.IssueDate)
			ordering.Receiving[0].IssueDate = &refDate
		}
	}

	if doc.AdditionalDocumentReference != nil {
		for _, ref := range doc.AdditionalDocumentReference {
			switch ref.DocumentType {
			case "50":
				if ordering.Tender == nil {
					ordering.Tender = make([]*org.DocumentRef, 0)
				}
				docRef := &org.DocumentRef{
					Code: cbc.Code(ref.ID),
				}
				if ref.IssueDate != "" {
					refDate := ParseDate(ref.IssueDate)
					docRef.IssueDate = &refDate
				}
				ordering.Tender = append(ordering.Tender, docRef)
			case "130":
				if ordering.Identities == nil {
					ordering.Identities = make([]*org.Identity, 0)
				}
				ordering.Identities = append(ordering.Identities, &org.Identity{
					Code: cbc.Code(ref.ID),
				})
			}
			// Other document types not mapped to GOBL
		}
	}

	if ordering.Code != "" || ordering.Period != nil || ordering.Despatch != nil || ordering.Receiving != nil || ordering.Tender != nil || ordering.Identities != nil {
		return ordering
	}
	return nil
}
