package gtou

import (
	"github.com/invopop/gobl.ubl/document"
	"github.com/invopop/gobl/bill"
)

func (c *Converter) newOrdering(o *bill.Ordering) error {
	if o == nil {
		return nil
	}

	if o.Code != "" {
		c.doc.BuyerReference = o.Code.String()
	}

	// If both ordering.seller and seller are present, the original seller is used
	// as the tax representative.
	if o.Seller != nil {
		p := c.doc.AccountingSupplierParty.Party
		c.doc.TaxRepresentativeParty = &p
		c.doc.AccountingSupplierParty = document.SupplierParty{
			Party: c.newParty(o.Seller),
		}
	}

	if o.Period != nil {
		start := formatDate(o.Period.Start)
		end := formatDate(o.Period.End)
		c.doc.InvoicePeriod = []document.Period{
			{
				StartDate: &start,
				EndDate:   &end,
			},
		}
	}

	if len(o.Despatch) > 0 {
		c.doc.DespatchDocumentReference = make([]document.Reference, 0, len(o.Despatch))
		for _, despatch := range o.Despatch {
			c.doc.DespatchDocumentReference = append(c.doc.DespatchDocumentReference, document.Reference{
				ID: document.IDType{Value: string(despatch.Code)},
			})
		}
	}

	if len(o.Receiving) > 0 {
		c.doc.ReceiptDocumentReference = make([]document.Reference, 0, len(o.Receiving))
		for _, receiving := range o.Receiving {
			c.doc.ReceiptDocumentReference = append(c.doc.ReceiptDocumentReference, document.Reference{
				ID: document.IDType{Value: string(receiving.Code)},
			})
		}
	}

	if len(o.Contracts) > 0 {
		c.doc.ContractDocumentReference = make([]document.Reference, 0, len(o.Contracts))
		for _, contract := range o.Contracts {
			c.doc.ContractDocumentReference = append(c.doc.ContractDocumentReference, document.Reference{
				ID: document.IDType{Value: string(contract.Code)},
			})
		}
	}

	if len(o.Tender) > 0 {
		c.doc.AdditionalDocumentReference = make([]document.Reference, 0, len(o.Tender))
		for _, tender := range o.Tender {
			c.doc.AdditionalDocumentReference = append(c.doc.AdditionalDocumentReference, document.Reference{
				ID: document.IDType{Value: string(tender.Code)},
			})
		}
	}

	if len(o.Purchases) > 0 {
		purchase := o.Purchases[0]
		c.doc.OrderReference = &document.OrderReference{
			ID: purchase.Code.String(),
		}
	}

	return nil
}
