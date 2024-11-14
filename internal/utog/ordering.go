package utog

import (
	"github.com/invopop/gobl.ubl/document"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

func (c *Converter) getOrdering(doc *document.Invoice) error {
	ordering := &bill.Ordering{}

	if doc.OrderReference != nil && doc.OrderReference.ID != "" {
		ordering.Code = cbc.Code(doc.OrderReference.ID)
	}

	// GOBL does not currently support multiple periods, so only the first one is taken
	if len(doc.InvoicePeriod) > 0 {
		ordering.Period = c.setPeriodDates(&doc.InvoicePeriod[0])
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
			if ref.DocumentTypeCode != nil {
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
			}
			// Other document types not mapped to GOBL
		}
	}

	if ordering.Code != "" || ordering.Period != nil || ordering.Despatch != nil || ordering.Receiving != nil || ordering.Tender != nil || ordering.Identities != nil {
		c.inv.Ordering = ordering
	}
	return nil
}

func (c *Converter) getReference(ref *document.Reference) (*org.DocumentRef, error) {
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
		docRef.Period = c.setPeriodDates(ref.ValidityPeriod)
	}
	return docRef, nil
}

func (c *Converter) setPeriodDates(invoicePeriod *document.Period) *cal.Period {
	period := &cal.Period{}
	if invoicePeriod.StartDate != nil {
		start, err := ParseDate(*invoicePeriod.StartDate)
		if err != nil {
			return nil
		}
		period.Start = start
	}
	if invoicePeriod.EndDate != nil {
		end, err := ParseDate(*invoicePeriod.EndDate)
		if err != nil {
			return nil
		}
		period.End = end
	}
	return period
}
