// Package gtou provides a conversor from GOBL to UBL.
package gtou

import (
	"encoding/xml"
	"fmt"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
)

// Converter is a struct that contains the necessary elements to convert between GOBL and UBL
type Converter struct {
	doc *Document
}

// NewConverter creates a new Converter instance
func NewConverter() *Converter {
	c := new(Converter)
	c.doc = new(Document)
	return c
}

// GetDocument returns the document from the conversor
func (c *Converter) GetDocument() *Document {
	return c.doc
}

// ConvertToUBL converts a GOBL envelope into a UBL document
func (c *Converter) ConvertToUBL(env *gobl.Envelope) (*Document, error) {
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, fmt.Errorf("invalid type %T", env.Document)
	}

	err := c.newDocument(inv)
	if err != nil {
		return nil, err
	}

	return c.doc, nil
}

func (c *Converter) newDocument(inv *bill.Invoice) error {

	// Create the UBL document
	c.doc = &Document{
		CACNamespace:            CAC,
		CBCNamespace:            CBC,
		QDTNamespace:            QDT,
		UDTNamespace:            UDT,
		UBLNamespace:            UBL,
		CCTSNamespace:           CCTS,
		XSINamespace:            XSI,
		SchemaLocation:          SchemaLocation,
		CustomizationID:         CustomizationID,
		ID:                      invoiceNumber(inv.Series, inv.Code),
		IssueDate:               formatDate(inv.IssueDate),
		InvoiceTypeCode:         invoiceTypeCode(inv),
		DocumentCurrencyCode:    string(inv.Currency),
		AccountingSupplierParty: SupplierParty{Party: c.newParty(inv.Supplier)},
		AccountingCustomerParty: CustomerParty{Party: c.newParty(inv.Customer)},
	}

	if len(inv.Notes) > 0 {
		c.doc.Note = make([]string, len(inv.Notes))
		for i, note := range inv.Notes {
			c.doc.Note[i] = note.Text
		}
	}

	err := c.newOrdering(inv.Ordering)
	if err != nil {
		return err
	}

	err = c.newPayment(inv.Payment)
	if err != nil {
		return err
	}

	err = c.newDelivery(inv.Delivery)
	if err != nil {
		return err
	}

	err = c.newCharges(inv)
	if err != nil {
		return err
	}

	err = c.newTotals(inv.Totals, string(inv.Currency))
	if err != nil {
		return err
	}

	err = c.newLines(inv)
	if err != nil {
		return err
	}

	return nil
}

func invoiceNumber(series cbc.Code, code cbc.Code) string {
	if series == "" {
		return code.String()
	}
	return fmt.Sprintf("%s-%s", series, code)
}

// TODO: Use tags from EN 16931 Add-on to expand the valid list of invoice types
func invoiceTypeCode(inv *bill.Invoice) string {
	if inv.Type == bill.InvoiceTypeStandard && inv.HasTags(tax.TagSelfBilled) {
		return "389"
	}
	hash := map[cbc.Key]string{
		bill.InvoiceTypeStandard:   "380",
		bill.InvoiceTypeCorrective: "384",
		bill.InvoiceTypeCreditNote: "381",
	}
	return hash[inv.Type]
}

// Bytes returns the XML representation of the document in bytes
func (d *Document) Bytes() ([]byte, error) {
	bytes, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), bytes...), nil
}
