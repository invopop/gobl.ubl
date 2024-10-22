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
		Code:      cbc.Code(doc.ID),
		Type:      cbc.Key(doc.InvoiceTypeCode),
		IssueDate: ParseDate(doc.IssueDate),
		Currency:  currency.Code(doc.DocumentCurrencyCode),
		Supplier:  ParseUtoGParty(&doc.AccountingSupplierParty.Party),
		Customer:  ParseUtoGParty(&doc.AccountingCustomerParty.Party),
		Lines:     ParseUtoGLines(doc),
	}

	// Payment comprised of terms, means and payee. Check there is relevant info in at least one of them to create a payment
	if doc.PaymentMeans != nil || len(doc.PaymentTerms) > 0 {
		inv.Payment = ParseUtoGPayment(doc)
	}

	if len(doc.Note) > 0 {
		inv.Notes = make([]*cbc.Note, 0, len(doc.Note))
		for _, note := range doc.Note {
			n := &cbc.Note{
				Text: note.Value,
			}
			inv.Notes = append(inv.Notes, n)
		}
	}

	ordering := ParseUtoGOrdering(inv, doc)
	if ordering != nil {
		inv.Ordering = ordering
	}

	delivery := ParseUtoGDelivery(inv, doc)
	if delivery != nil {
		inv.Delivery = delivery
	}

	if len(doc.BillingReference) != nil {
		inv.Preceding = make([]*org.DocumentRef, 0, len(doc.BillingReference))
		for _, ref := range doc.BillingReference {
			docRef := &org.DocumentRef{
				Code: cbc.Code(ref.InvoiceDocumentReference.ID.Value),
			}
			if ref.InvoiceDocumentReference.IssueDate != "" {
				refDate := ParseDate(ref.InvoiceDocumentReference.IssueDate)
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
		inv.Supplier = ParseutoGParty(doc.TaxRepresentativeParty)
	}

	if len(doc.AllowanceCharge) > 0 {
		charges, discounts := ParseutoGCharges(doc.AllowanceCharge)
		if len(charges) > 0 {
			inv.Charges = charges
		}
		if len(discounts) > 0 {
			inv.Discounts = discounts
		}
	}

	return inv
}
