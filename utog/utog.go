package utog

import (
	"encoding/xml"
	"fmt"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/org"
)

// Conversor is a struct that contains the necessary elements to convert between GOBL and UBL
type Conversor struct {
	inv *bill.Invoice
	doc *Document
}

// NewConversor Builder function
func NewConversor() *Conversor {
	c := new(Conversor)
	c.inv = new(bill.Invoice)
	c.doc = new(Document)
	return c
}

// GetInvoice returns the invoice from the conversor
func (c *Conversor) GetInvoice() *bill.Invoice {
	return c.inv
}

// ConvertToGOBL converts a UBL document into a GOBL envelope
func (c *Conversor) ConvertToGOBL(xmlData []byte) (*gobl.Envelope, error) {
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

func (c *Conversor) NewInvoice(doc *Document) error {

	inv := &bill.Invoice{
		Code:     cbc.Code(doc.ID),
		Type:     cbc.Key(*doc.InvoiceTypeCode),
		Currency: currency.Code(*doc.DocumentCurrencyCode),
		Supplier: c.getParty(&doc.AccountingSupplierParty.Party),
		Customer: c.getParty(&doc.AccountingCustomerParty.Party),
	}

	issueDate, err := ParseDate(*doc.IssueDate)
	if err != nil {
		return err
	}
	inv.IssueDate = issueDate

	err = c.getLines(doc)
	if err != nil {
		return err
	}
	fmt.Println(inv.Lines)

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
					return err
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
			return err
		}
	}

	return nil
}
