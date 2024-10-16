package ubl

import (
	"github.com/invopop/gobl.ubl/structs"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

func ParseCtoGOrdering(inv *bill.Invoice, doc *structs.XMLDoc) *bill.Ordering {
	ordering := &bill.Ordering{}

	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.BuyerReference != nil {
		if *doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.BuyerReference != "N/A" {
			ordering.Code = cbc.Code(*doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.BuyerReference)
		}
	}

	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod != nil {
		period := &cal.Period{}

		if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod.StartDateTime != nil {
			period.Start = ParseDate(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod.StartDateTime.DateTimeString)
		}

		if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod.EndDateTime != nil {
			period.End = ParseDate(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod.EndDateTime.DateTimeString)
		}
		if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod.Description != nil {
			period.Label = *doc.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.BillingSpecifiedPeriod.Description
		}
		ordering.Period = period
	}

	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.DespatchAdviceReferencedDocument != nil {
		ordering.Despatch = []*org.DocumentRef{
			{
				Code: cbc.Code(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.DespatchAdviceReferencedDocument.IssuerAssignedID),
			},
		}
		if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.DespatchAdviceReferencedDocument.FormattedIssueDateTime != nil {
			refDate := ParseDate(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.DespatchAdviceReferencedDocument.FormattedIssueDateTime.DateTimeString)
			ordering.Despatch[0].IssueDate = &refDate
		}
	}

	if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.ReceivingAdviceReferencedDocument != nil {
		ordering.Receiving = []*org.DocumentRef{
			{
				Code: cbc.Code(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.ReceivingAdviceReferencedDocument.IssuerAssignedID),
			},
		}
		if doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.ReceivingAdviceReferencedDocument.FormattedIssueDateTime != nil {
			refDate := ParseDate(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeDelivery.ReceivingAdviceReferencedDocument.FormattedIssueDateTime.DateTimeString)
			ordering.Receiving[0].IssueDate = &refDate
		}
	}

	if len(doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.AdditionalReferencedDocument) > 0 {
		for _, ref := range doc.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.AdditionalReferencedDocument {
			switch ref.TypeCode {
			case "50":
				if ordering.Tender == nil {
					ordering.Tender = make([]*org.DocumentRef, 0)
				}
				docRef := &org.DocumentRef{
					Code: cbc.Code(ref.IssuerAssignedID),
				}
				if ref.FormattedIssueDateTime != nil {
					refDate := ParseDate(ref.FormattedIssueDateTime.DateTimeString)
					docRef.IssueDate = &refDate
				}
				ordering.Tender = append(ordering.Tender, docRef)
			case "130":
				if ordering.Identities == nil {
					ordering.Identities = make([]*org.Identity, 0)
				}
				ordering.Identities = append(ordering.Identities, &org.Identity{
					Code: cbc.Code(ref.IssuerAssignedID),
				})
			}
			// Case 916: Additional Document Reference not mapped to GOBL
		}
	}

	if ordering.Code != "" || ordering.Period != nil || ordering.Despatch != nil || ordering.Receiving != nil || ordering.Tender != nil || ordering.Identities != nil {
		return ordering
	}
	return nil
}
