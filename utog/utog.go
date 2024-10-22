package utog

import (
	"encoding/xml"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/org"
)

type Conversor struct {
	doc *Document
	inv *bill.Invoice
}

func NewConversor() *Conversor {
	c := new(Conversor)
	c.doc = new(Document)
	c.inv = new(bill.Invoice)
	return c
}

func (c *Conversor) GetInvoice() *bill.Invoice {
	return c.inv
}

func (c *Conversor) ConvertToGOBL(xmlData []byte) (*gobl.Envelope, error) {
	if err := xml.Unmarshal(xmlData, &c.doc); err != nil {
		return nil, err
	}

	inv, err := c.NewInvoice(c.doc)

	if err != nil {
		return nil, err
	}
	env, err := gobl.Envelop(inv)
	if err != nil {
		return nil, err
	}
	return env, nil
}

func (c *Conversor) NewInvoice(doc *Document) (*bill.Invoice, error) {

	inv := &bill.Invoice{
		Code:     cbc.Code(doc.ID),
		Type:     cbc.Key(*doc.InvoiceTypeCode),
		Currency: currency.Code(*doc.DocumentCurrencyCode),
		Supplier: c.getParty(&doc.AccountingSupplierParty.Party),
		Customer: c.getParty(&doc.AccountingCustomerParty.Party),
	}

	issueDate, err := ParseDate(*doc.IssueDate)
	if err != nil {
		return nil, err
	}
	inv.IssueDate = issueDate

	err = c.getLines(doc)
	if err != nil {
		return nil, err
	}

	err = c.getPayment(doc)
	if err != nil {
		return nil, err
	}

	err = c.getOrdering(doc)
	if err != nil {
		return nil, err
	}

	err = c.getDelivery(doc)
	if err != nil {
		return nil, err
	}

	if len(doc.Note) > 0 {
		inv.Notes = make([]*cbc.Note, 0, len(doc.Note))
		for _, note := range doc.Note {
			n := &cbc.Note{
				Text: note,
			}
			inv.Notes = append(inv.Notes, n)
		}
	}

	if len(doc.BillingReference) > 0 {
		inv.Preceding = make([]*org.DocumentRef, 0, len(doc.BillingReference))
		for _, ref := range doc.BillingReference {
			docRef := &org.DocumentRef{
				Code: cbc.Code(ref.InvoiceDocumentReference.ID),
			}
			if ref.InvoiceDocumentReference.IssueDate != nil {
				refDate, err := ParseDate(*ref.InvoiceDocumentReference.IssueDate)
				if err != nil {
					return nil, err
				}
				docRef.IssueDate = &refDate
			}
			inv.Preceding = append(inv.Preceding, docRef)
		}
	}

	if doc.TaxRepresentativeParty != nil {
		// Move the original seller to the ordering.seller party
		if inv.Ordering == nil {
			inv.Ordering = &bill.Ordering{}
		}
		inv.Ordering.Seller = inv.Supplier

		// Overwrite the seller field with the tax representative
		inv.Supplier = c.getParty(doc.TaxRepresentativeParty)
	}

	if len(doc.AllowanceCharge) > 0 {
		err := c.getCharges(doc)
		if err != nil {
			return nil, err
		}
	}

	return inv, nil
}
