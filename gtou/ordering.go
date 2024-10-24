package gtou

import (
	"github.com/invopop/gobl/bill"
)

func (c *Conversor) getOrdering(ordering *bill.Ordering) error {
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
		err := c.newSupplier(ordering.Seller)
		if err != nil {
			return err
		}
	}

	if ordering.Period != nil {
		c.doc.InvoicePeriod = []Period{makePeriod(ordering.Period)}
	}

	if len(ordering.Despatch) > 0 {
		c.doc.DespatchDocumentReference = make([]DocumentReference, len(ordering.Despatch))
		for i, despatch := range ordering.Despatch {
			c.doc.DespatchDocumentReference[i] = DocumentReference{
				ID:           IDType{Value: string(despatch.Code)},
				DocumentType: string(despatch.Type),
			}
		}
	}

	if len(ordering.Receiving) > 0 {
		c.doc.ReceiptDocumentReference = make([]DocumentReference, len(ordering.Receiving))
		for i, receiving := range ordering.Receiving {
			c.doc.ReceiptDocumentReference[i] = DocumentReference{
				ID: IDType{Value: string(receiving.Code)},
			}
		}
	}

	if len(ordering.Contracts) > 0 {
		c.doc.ContractDocumentReference = make([]DocumentReference, len(ordering.Contracts))
		for i, contract := range ordering.Contracts {
			c.doc.ContractDocumentReference[i] = DocumentReference{
				ID: IDType{Value: string(contract.Code)},
			}
		}
	}

	if len(ordering.Tender) > 0 {
		c.doc.AdditionalDocumentReference = make([]DocumentReference, len(ordering.Tender))
		for i, tender := range ordering.Tender {
			c.doc.AdditionalDocumentReference[i] = DocumentReference{
				ID: IDType{Value: string(tender.Code)},
			}
		}
	}

	return nil
}
