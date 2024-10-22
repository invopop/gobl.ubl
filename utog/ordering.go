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
		ordering.Despatch = make([]*org.DocumentRef, 0)
		for _, despatchRef := range doc.DespatchDocumentReference {
			docRef := &org.DocumentRef{
				Code: cbc.Code(despatchRef.ID),
			}
			if despatchRef.IssueDate != nil {
				refDate, err := ParseDate(*despatchRef.IssueDate)
				if err != nil {
					return err
				}
				docRef.IssueDate = &refDate
			}
			ordering.Despatch = append(ordering.Despatch, docRef)
		}
	}

	if doc.ReceiptDocumentReference != nil {
		ordering.Receiving = make([]*org.DocumentRef, 0)
		for _, receiptRef := range doc.ReceiptDocumentReference {
			docRef := &org.DocumentRef{
				Code: cbc.Code(receiptRef.ID),
			}
			if receiptRef.IssueDate != nil {
				refDate, err := ParseDate(*receiptRef.IssueDate)
				if err != nil {
					return err
				}
				docRef.IssueDate = &refDate
			}
			ordering.Receiving = append(ordering.Receiving, docRef)
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
				if ref.IssueDate != nil {
					refDate, err := ParseDate(*ref.IssueDate)
					if err != nil {
						return err
					}
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
		c.inv.Ordering = ordering
	}
	return nil
}
