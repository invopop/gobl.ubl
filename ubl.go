// Package cii helps convert GOBL into Cross Industry Invoice documents and vice versa.
package ubl

import (
	"encoding/xml"
	"fmt"

	"github.com/invopop/gobl"
	gtou "github.com/invopop/gobl.ubl/internal/gtou"
	utog "github.com/invopop/gobl.ubl/internal/utog"
	"github.com/invopop/gobl.ubl/structs"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/org"
)

// UBL schema constants
const (
	CBC = "urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"
	CAC = "urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
	UBL = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
)

// Document is a pseudo-model for containing the XML document being created
type Document struct {
	XMLName              xml.Name `xml:"ubl:Invoice"`
	UBLNamespace         string   `xml:"xmlns:ubl,attr"`
	CBCNamespace         string   `xml:"xmlns:cbc,attr"`
	CACNamespace         string   `xml:"xmlns:cac,attr"`
	CustomizationID      string   `xml:"cbc:CustomizationID"`
	ProfileID            string   `xml:"cbc:ProfileID"`
	ID                   string   `xml:"cbc:ID"`
	IssueDate            string   `xml:"cbc:IssueDate"`
	InvoiceTypeCode      string   `xml:"cbc:InvoiceTypeCode"`
	DocumentCurrencyCode string   `xml:"cbc:DocumentCurrencyCode"`
	// AccountingSupplierParty *Party `xml:"cac:AccountingSupplierParty"`
	// AccountingCustomerParty *Party `xml:"cac:AccountingCustomerParty"`
	// InvoiceLines   []*InvoiceLine `xml:"cac:InvoiceLine"`
}

// NewDocument converts a GOBL envelope into a XRechnung and Factur-X document
func NewDocument(env *gobl.Envelope) (*Document, error) {
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, fmt.Errorf("invalid type %T", env.Document)
	}

	transaction, err := gtou.NewTransaction(inv)
	if err != nil {
		return nil, err
	}

	doc := Document{
		CACNamespace:    CAC,
		CBCNamespace:    CBC,
		UBLNamespace:    UBL,
		CustomizationID: "urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100",
		ProfileID:       "urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100",
		Transaction:     transaction,
	}
	return &doc, nil
}

// Bytes returns the XML representation of the document in bytes
func (d *Document) Bytes() ([]byte, error) {
	bytes, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), bytes...), nil
}

// NewDocument converts a XRechnung document into a GOBL envelope
func NewGOBLFromUBL(doc *structs.Invoice) (*gobl.Envelope, error) {

	inv := mapUBLToInvoice(doc)
	env, err := gobl.Envelop(inv)
	if err != nil {
		return nil, err
	}
	return env, nil
}

func mapUBLToInvoice(doc *structs.Invoice) *bill.Invoice {

	inv := &bill.Invoice{
		Code:      cbc.Code(doc.ID),
		Type:      cbc.Key(doc.InvoiceTypeCode),
		IssueDate: utog.ParseDate(doc.IssueDate),
		Currency:  currency.Code(doc.DocumentCurrencyCode),
		Supplier:  utog.ParseUtoGParty(&doc.AccountingSupplierParty.Party),
		Customer:  utog.ParseUtoGParty(&doc.AccountingCustomerParty.Party),
		Lines:     utog.ParseUtoGLines(doc),
	}

	// Payment comprised of terms, means and payee. Check there is relevant info in at least one of them to create a payment
	if doc.PaymentMeans != nil || len(doc.PaymentTerms) > 0 {
		inv.Payment = utog.ParseUtoGPayment(doc)
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

	ordering := utog.ParseutoGOrdering(inv, doc)
	if ordering != nil {
		inv.Ordering = ordering
	}

	delivery := utog.ParseutoGDelivery(inv, doc)
	if delivery != nil {
		inv.Delivery = delivery
	}

	if len(doc.BillingReference) > 0 {
		inv.Preceding = make([]*org.DocumentRef, 0, len(doc.BillingReference))
		for _, ref := range doc.BillingReference {
			docRef := &org.DocumentRef{
				Code: cbc.Code(ref.InvoiceDocumentReference.ID.Value),
			}
			if ref.InvoiceDocumentReference.IssueDate != "" {
				refDate := utog.ParseDate(ref.InvoiceDocumentReference.IssueDate)
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
		inv.Supplier = utog.ParseutoGParty(doc.TaxRepresentativeParty)
	}

	if len(doc.AllowanceCharge) > 0 {
		charges, discounts := utog.ParseutoGCharges(doc.AllowanceCharge)
		if len(charges) > 0 {
			inv.Charges = charges
		}
		if len(discounts) > 0 {
			inv.Discounts = discounts
		}
	}

	return inv
}
