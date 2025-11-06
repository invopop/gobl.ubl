package ubl

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

func goblAddOrdering(in *Invoice, out *bill.Invoice) error {
	ordering := new(bill.Ordering)
	// set ensures that ordering is only added when needed
	set := false
	if in.BuyerReference != "" {
		ordering.Code = cbc.Code(in.BuyerReference)
		set = true
	}

	// GOBL does not currently support multiple periods, so only the first one is taken
	if len(in.InvoicePeriod) > 0 {
		ordering.Period = goblPeriodDates(&in.InvoicePeriod[0])
		set = true
	}

	if in.DespatchDocumentReference != nil {
		ordering.Despatch = make([]*org.DocumentRef, 0)
		for _, despatchRef := range in.DespatchDocumentReference {
			docRef, err := goblReference(&despatchRef)
			if err != nil {
				return err
			}
			ordering.Despatch = append(ordering.Despatch, docRef)
		}
		set = true
	}

	if in.ReceiptDocumentReference != nil {
		ordering.Receiving = make([]*org.DocumentRef, 0)
		for _, receiptRef := range in.ReceiptDocumentReference {
			docRef, err := goblReference(&receiptRef)
			if err != nil {
				return err
			}
			ordering.Receiving = append(ordering.Receiving, docRef)
		}
		set = true
	}

	if in.OrderReference != nil && in.OrderReference.ID != "" {
		ordering.Purchases = []*org.DocumentRef{
			{
				Code: cbc.Code(in.OrderReference.ID),
			},
		}
		set = true
	}

	if in.ContractDocumentReference != nil {
		ordering.Contracts = make([]*org.DocumentRef, 0)
		for _, contractRef := range in.ContractDocumentReference {
			docRef, err := goblReference(&contractRef)
			if err != nil {
				return err
			}
			ordering.Contracts = append(ordering.Contracts, docRef)
		}
		set = true
	}

	if in.AdditionalDocumentReference != nil {
		for _, ref := range in.AdditionalDocumentReference {
			if ref.DocumentTypeCode != nil {
				switch *ref.DocumentTypeCode {
				case "50":
					if ordering.Tender == nil {
						ordering.Tender = make([]*org.DocumentRef, 0)
					}
					docRef, err := goblReference(&ref)
					if err != nil {
						return err
					}
					ordering.Tender = append(ordering.Tender, docRef)
					set = true
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
					set = true
				}
			}
			// Other document types not mapped to GOBL
		}
	}

	if set {
		out.Ordering = ordering
	}

	return nil
}

func goblReference(ref *Reference) (*org.DocumentRef, error) {
	docRef := &org.DocumentRef{
		Code: cbc.Code(ref.ID.Value),
	}
	if ref.DocumentType != nil {
		docRef.Type = cbc.Key(*ref.DocumentType)
	}
	if ref.IssueDate != nil {
		refDate, err := parseDate(*ref.IssueDate)
		if err != nil {
			return nil, err
		}
		docRef.IssueDate = &refDate
	}
	if ref.DocumentDescription != nil {
		docRef.Description = *ref.DocumentDescription
	}
	if ref.ValidityPeriod != nil {
		docRef.Period = goblPeriodDates(ref.ValidityPeriod)
	}
	return docRef, nil
}

func goblPeriodDates(invoicePeriod *Period) *cal.Period {
	period := &cal.Period{}
	if invoicePeriod.StartDate != nil {
		start, err := parseDate(*invoicePeriod.StartDate)
		if err != nil {
			return nil
		}
		period.Start = start
	}
	if invoicePeriod.EndDate != nil {
		end, err := parseDate(*invoicePeriod.EndDate)
		if err != nil {
			return nil
		}
		period.End = end
	}
	return period
}
