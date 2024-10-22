package utog

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

func (c *Conversor) getOrdering(doc *Document) error {
	ordering := &bill.Ordering{}

	if doc.OrderReference != nil && doc.OrderReference.ID != "" {
		ordering.Code = cbc.Code(doc.OrderReference.ID)
	}

	// GOBL does not currently support multiple periods, so only the first one is taken
	if doc.InvoicePeriod != nil {
		period := &cal.Period{}

		if doc.InvoicePeriod[0].StartDate != nil {
			start, err := ParseDate(*doc.InvoicePeriod[0].StartDate)
			if err != nil {
				return err
			}
			period.Start = start
		}

		if doc.InvoicePeriod[0].EndDate != nil {
			end, err := ParseDate(*doc.InvoicePeriod[0].EndDate)
			if err != nil {
				return err
			}
			period.End = end
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
