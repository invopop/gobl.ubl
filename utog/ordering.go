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
			docRef, err := c.getReference(&despatchRef)
			if err != nil {
				return err
			}
			ordering.Despatch = append(ordering.Despatch, docRef)
		}
	}

	if doc.ReceiptDocumentReference != nil {
		ordering.Receiving = make([]*org.DocumentRef, 0)
		for _, receiptRef := range doc.ReceiptDocumentReference {
			docRef, err := c.getReference(&receiptRef)
			if err != nil {
				return err
			}
			ordering.Receiving = append(ordering.Receiving, docRef)
		}
	}

	if doc.ContractDocumentReference != nil {
		ordering.Contracts = make([]*org.DocumentRef, 0)
		for _, contractRef := range doc.ContractDocumentReference {
			docRef, err := c.getReference(&contractRef)
			if err != nil {
				return err
			}
			ordering.Contracts = append(ordering.Contracts, docRef)
		}
	}

	if doc.AdditionalDocumentReference != nil {
		for _, ref := range doc.AdditionalDocumentReference {
			switch *ref.DocumentTypeCode {
			case "50":
				if ordering.Tender == nil {
					ordering.Tender = make([]*org.DocumentRef, 0)
				}
				docRef, err := c.getReference(&ref)
				if err != nil {
					return err
				}
				ordering.Tender = append(ordering.Tender, docRef)
			case "130":
				if ordering.Identities == nil {
					ordering.Identities = make([]*org.Identity, 0)
				}
				identity := &org.Identity{
					Code: cbc.Code(ref.ID.Value),
				}
				if ref.ID.SchemeID != nil {
					identity.Label = *ref.ID.SchemeID
				}
				ordering.Identities = append(ordering.Identities, identity)
			}
			// Other document types not mapped to GOBL
		}
	}

	if ordering.Code != "" || ordering.Period != nil || ordering.Despatch != nil || ordering.Receiving != nil || ordering.Tender != nil || ordering.Identities != nil {
		c.inv.Ordering = ordering
	}
	return nil
}

func (c *Conversor) getReference(ref *DocumentReference) (*org.DocumentRef, error) {
	docRef := &org.DocumentRef{
		Code: cbc.Code(ref.ID.Value),
	}
	if ref.DocumentType != nil {
		docRef.Type = cbc.Key(*ref.DocumentType)
	}
	if ref.IssueDate != nil {
		refDate, err := ParseDate(*ref.IssueDate)
		if err != nil {
			return nil, err
		}
		docRef.IssueDate = &refDate
	}
	if ref.DocumentDescription != nil {
		docRef.Description = *ref.DocumentDescription
	}
	if ref.ValidityPeriod != nil {
		period := &cal.Period{}
		if ref.ValidityPeriod.StartDate != nil {
			start, err := ParseDate(*ref.ValidityPeriod.StartDate)
			if err != nil {
				return nil, err
			}
			period.Start = start
		}
		if ref.ValidityPeriod.EndDate != nil {
			end, err := ParseDate(*ref.ValidityPeriod.EndDate)
			if err != nil {
				return nil, err
			}
			period.End = end
		}
		docRef.Period = period
	}
	return docRef, nil
}
