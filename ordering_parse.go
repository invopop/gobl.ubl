package ubl

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

func hasOrderingData(o *bill.Ordering) bool {
	return o.Code != "" ||
		o.Cost != "" ||
		o.Period != nil ||
		len(o.Despatch) > 0 ||
		len(o.Receiving) > 0 ||
		len(o.Purchases) > 0 ||
		len(o.Sales) > 0 ||
		len(o.Projects) > 0 ||
		len(o.Contracts) > 0 ||
		len(o.Tender) > 0 ||
		len(o.Identities) > 0 ||
		o.Issuer != nil
}

func (ui *Invoice) goblAddOrdering(out *bill.Invoice, o *options) error {
	ordering := new(bill.Ordering)

	if ui.BuyerReference != "" {
		ordering.Code = cbc.Code(cleanString(ui.BuyerReference))
	}

	// BT-19: Buyer accounting reference
	if ui.AccountingCost != "" {
		ordering.Cost = cbc.Code(cleanString(ui.AccountingCost))
	}

	ordering.Issuer = ui.goblOrderingIssuer(o)

	// GOBL does not currently support multiple periods, so only the first one is taken
	if len(ui.InvoicePeriod) > 0 {
		p, err := goblPeriodDates(&ui.InvoicePeriod[0])
		if err != nil {
			return err
		}
		ordering.Period = p

		// BT-8: VAT point date code
		if code := ui.InvoicePeriod[0].DescriptionCode; code != "" {
			if key, ok := taxPointKeyMap[code]; ok {
				if out.Tax == nil {
					out.Tax = new(bill.Tax)
				}
				out.Tax.Point = key
			}
		}
	}

	if ui.DespatchDocumentReference != nil {
		ordering.Despatch = make([]*org.DocumentRef, 0)
		for _, despatchRef := range ui.DespatchDocumentReference {
			docRef, err := goblReference(&despatchRef)
			if err != nil {
				return err
			}
			ordering.Despatch = append(ordering.Despatch, docRef)
		}
	}

	if ui.ReceiptDocumentReference != nil {
		ordering.Receiving = make([]*org.DocumentRef, 0)
		for _, receiptRef := range ui.ReceiptDocumentReference {
			docRef, err := goblReference(&receiptRef)
			if err != nil {
				return err
			}
			ordering.Receiving = append(ordering.Receiving, docRef)
		}
	}

	if ui.OrderReference != nil && ui.OrderReference.ID != "" {
		ordering.Purchases = []*org.DocumentRef{
			{
				Code: cbc.Code(ui.OrderReference.ID),
			},
		}
		// BT-14: Sales order reference
		if ui.OrderReference.SalesOrderID != "" {
			ordering.Sales = []*org.DocumentRef{
				{Code: cbc.Code(ui.OrderReference.SalesOrderID)},
			}
		}
	}

	// BT-11: Project reference
	for _, proj := range ui.ProjectReference {
		if proj.ID != "" {
			ordering.Projects = append(ordering.Projects, &org.DocumentRef{
				Code: cbc.Code(proj.ID),
			})
		}
	}

	if ui.ContractDocumentReference != nil {
		ordering.Contracts = make([]*org.DocumentRef, 0)
		for _, contractRef := range ui.ContractDocumentReference {
			docRef, err := goblReference(&contractRef)
			if err != nil {
				return err
			}
			ordering.Contracts = append(ordering.Contracts, docRef)
		}
	}

	if ui.OriginatorDocumentReference != nil {
		ordering.Tender = make([]*org.DocumentRef, 0)
		for _, tenderRef := range ui.OriginatorDocumentReference {
			docRef, err := goblReference(&tenderRef)
			if err != nil {
				return err
			}
			ordering.Tender = append(ordering.Tender, docRef)
		}
	}

	ordering.Identities = goblAdditionalDocumentIdentities(ui.AdditionalDocumentReference)

	if hasOrderingData(ordering) {
		out.Ordering = ordering
	}

	return nil
}

// goblAdditionalDocumentIdentities maps AdditionalDocumentReference entries with
// DocumentTypeCode "130" (product invoice data sheet) to GOBL identities.
// Other document types are not mapped to GOBL.
func goblAdditionalDocumentIdentities(refs []Reference) []*org.Identity {
	if refs == nil {
		return nil
	}

	var identities []*org.Identity
	for _, ref := range refs {
		if ref.DocumentTypeCode != "130" {
			continue
		}
		if identities == nil {
			identities = make([]*org.Identity, 0)
		}
		identity := &org.Identity{
			Code: cbc.Code(ref.ID.Value),
		}
		if ref.ID.SchemeID != nil {
			// This is very EN specific, but we currently do not provide a way to identify by context how we should handle each case
			identity.Ext = identity.Ext.Set(untdid.ExtKeyReference, cbc.Code(*ref.ID.SchemeID))
		}
		identities = append(identities, identity)
	}
	return identities
}

func (ui *Invoice) goblOrderingIssuer(o *options) *org.Party {
	sp := ui.AccountingSupplierParty.Party
	if sp == nil || sp.ServiceProviderParty == nil {
		return nil
	}
	return goblParty(sp.ServiceProviderParty.Party, o)
}

func goblReference(ref *Reference) (*org.DocumentRef, error) {
	docRef := &org.DocumentRef{
		Code: cbc.Code(ref.ID.Value),
	}
	if ref.DocumentType != "" {
		docRef.Type = cbc.Key(ref.DocumentType)
	}
	if ref.IssueDate != "" {
		refDate, err := parseDate(ref.IssueDate)
		if err != nil {
			return nil, err
		}
		docRef.IssueDate = &refDate
	}
	if ref.DocumentTypeCode != "" {
		docRef.Ext = docRef.Ext.Set(untdid.ExtKeyDocumentType, cbc.Code(ref.DocumentTypeCode))
	}
	if ref.DocumentDescription != "" {
		docRef.Description = cleanString(ref.DocumentDescription)
	}
	if ref.ValidityPeriod != nil {
		p, err := goblPeriodDates(ref.ValidityPeriod)
		if err != nil {
			return nil, err
		}
		docRef.Period = p
	}
	return docRef, nil
}

func goblPeriodDates(invoicePeriod *Period) (*cal.Period, error) {
	if invoicePeriod.StartDate == "" && invoicePeriod.EndDate == "" {
		return nil, nil
	}
	period := &cal.Period{}
	if invoicePeriod.StartDate != "" {
		start, err := parseDate(invoicePeriod.StartDate)
		if err != nil {
			return nil, err
		}
		period.Start = start
	}
	if invoicePeriod.EndDate != "" {
		end, err := parseDate(invoicePeriod.EndDate)
		if err != nil {
			return nil, err
		}
		period.End = end
	}
	return period, nil
}
