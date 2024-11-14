// Package utog provides a converter from UBL to GOBL.
package utog

import (
	"github.com/nbio/xml"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl.ubl/document"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/org"
)

// Converter is a struct that contains the necessary elements to convert between GOBL and UBL
type Converter struct {
	inv *bill.Invoice
	doc *document.Invoice
}

// Convert converts a UBL document into a GOBL envelope
func Convert(xmlData []byte) (*gobl.Envelope, error) {
	c := new(Converter)
	c.inv = new(bill.Invoice)
	c.doc = new(document.Invoice)
	if err := xml.Unmarshal(xmlData, &c.doc); err != nil {
		return nil, err
	}

	err := c.NewInvoice(c.doc)
	if err != nil {
		return nil, err
	}
	env, err := gobl.Envelop(c.inv)
	if err != nil {
		return nil, err
	}
	return env, nil
}

// NewInvoice creates a new invoice from a UBL document
func (c *Converter) NewInvoice(doc *document.Invoice) error {

	c.inv = &bill.Invoice{
		Code:     cbc.Code(doc.ID),
		Type:     TypeCodeParse(doc.InvoiceTypeCode),
		Currency: currency.Code(doc.DocumentCurrencyCode),
		Supplier: c.getParty(&doc.AccountingSupplierParty.Party),
		Customer: c.getParty(&doc.AccountingCustomerParty.Party),
	}

	issueDate, err := ParseDate(doc.IssueDate)
	if err != nil {
		return err
	}
	c.inv.IssueDate = issueDate

	err = c.getLines(doc)
	if err != nil {
		return err
	}

	err = c.getPayment(doc)
	if err != nil {
		return err
	}

	err = c.getOrdering(doc)
	if err != nil {
		return err
	}

	err = c.getDelivery(doc)
	if err != nil {
		return err
	}

	if len(doc.Note) > 0 {
		c.inv.Notes = make([]*cbc.Note, 0, len(doc.Note))
		for _, note := range doc.Note {
			n := &cbc.Note{
				Text: note,
			}
			c.inv.Notes = append(c.inv.Notes, n)
		}
	}

	if len(doc.BillingReference) > 0 {
		c.inv.Preceding = make([]*org.DocumentRef, 0, len(doc.BillingReference))
		for _, ref := range doc.BillingReference {
			var docRef *org.DocumentRef
			var err error

			switch {
			case ref.InvoiceDocumentReference != nil:
				docRef, err = c.getReference(ref.InvoiceDocumentReference)
			case ref.SelfBilledInvoiceDocumentReference != nil:
				docRef, err = c.getReference(ref.SelfBilledInvoiceDocumentReference)
			case ref.CreditNoteDocumentReference != nil:
				docRef, err = c.getReference(ref.CreditNoteDocumentReference)
			case ref.AdditionalDocumentReference != nil:
				docRef, err = c.getReference(ref.AdditionalDocumentReference)
			}
			if err != nil {
				return err
			}
			if docRef != nil {
				c.inv.Preceding = append(c.inv.Preceding, docRef)
			}
		}
	}

	if doc.TaxRepresentativeParty != nil {
		// Move the original seller to the ordering.seller party
		if c.inv.Ordering == nil {
			c.inv.Ordering = &bill.Ordering{}
		}
		c.inv.Ordering.Seller = c.inv.Supplier

		// Overwrite the seller field with the tax representative
		c.inv.Supplier = c.getParty(doc.TaxRepresentativeParty)
	}

	if len(doc.AllowanceCharge) > 0 {
		err := c.getCharges(doc)
		if err != nil {
			return err
		}
	}
	return nil
}
