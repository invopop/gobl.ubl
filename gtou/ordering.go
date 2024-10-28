package gtou

import (
	"github.com/invopop/gobl/bill"
)

func (c *Conversor) newOrdering(ordering *bill.Ordering) error {
	if ordering == nil {
		return nil
	}

	if ordering.Code != "" {
		c.doc.OrderReference = &OrderReference{ID: string(ordering.Code)}
	}

	// If both ordering.seller and seller are present, the original seller is used
	// as the tax representative.
	if ordering.Seller != nil {
		c.doc.TaxRepresentativeParty = &c.doc.AccountingSupplierParty.Party
		c.doc.AccountingSupplierParty = SupplierParty{
			Party: c.newParty(ordering.Seller),
		}
	}

	if ordering.Period != nil {
		c.doc.InvoicePeriod = []Period{makePeriod(ordering.Period)}
	}

	if len(ordering.Despatch) > 0 {
		c.doc.DespatchDocumentReference = make([]DocumentReference, 0, len(ordering.Despatch))
		for _, despatch := range ordering.Despatch {
			c.doc.DespatchDocumentReference = append(c.doc.DespatchDocumentReference, DocumentReference{
				ID: IDType{Value: string(despatch.Code)},
			})
		}
	}

	if len(ordering.Receiving) > 0 {
		c.doc.ReceiptDocumentReference = make([]DocumentReference, 0, len(ordering.Receiving))
		for _, receiving := range ordering.Receiving {
			c.doc.ReceiptDocumentReference = append(c.doc.ReceiptDocumentReference, DocumentReference{
				ID: IDType{Value: string(receiving.Code)},
			})
		}
	}

	if len(ordering.Contracts) > 0 {
		c.doc.ContractDocumentReference = make([]DocumentReference, 0, len(ordering.Contracts))
		for _, contract := range ordering.Contracts {
			c.doc.ContractDocumentReference = append(c.doc.ContractDocumentReference, DocumentReference{
				ID: IDType{Value: string(contract.Code)},
			})
		}
	}

	if len(ordering.Tender) > 0 {
		c.doc.AdditionalDocumentReference = make([]DocumentReference, 0, len(ordering.Tender))
		for _, tender := range ordering.Tender {
			c.doc.AdditionalDocumentReference = append(c.doc.AdditionalDocumentReference, DocumentReference{
				ID: IDType{Value: string(tender.Code)},
			})
		}
	}

	return nil
}
