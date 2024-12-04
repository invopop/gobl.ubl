// Package gtou provides a converter from GOBL to UBL.
package gtou

import (
	"fmt"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl.ubl/document"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
)

// Converter is a struct that contains the necessary elements to convert between GOBL and UBL
type Converter struct {
	doc *document.Invoice
}

// Convert converts a GOBL envelope into a UBL document
func Convert(env *gobl.Envelope) (*document.Invoice, error) {
	c := new(Converter)
	c.doc = new(document.Invoice)
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

	tc, err := getTypeCode(inv)
	if err != nil {
		return err
	}

	// Create the UBL document
	c.doc = &document.Invoice{
		CACNamespace:            document.CAC,
		CBCNamespace:            document.CBC,
		QDTNamespace:            document.QDT,
		UDTNamespace:            document.UDT,
		UBLNamespace:            document.UBL,
		CCTSNamespace:           document.CCTS,
		XSINamespace:            document.XSI,
		SchemaLocation:          document.SchemaLocation,
		CustomizationID:         document.CustomizationID,
		ID:                      invoiceNumber(inv.Series, inv.Code),
		IssueDate:               formatDate(inv.IssueDate),
		InvoiceTypeCode:         tc,
		DocumentCurrencyCode:    string(inv.Currency),
		AccountingSupplierParty: document.SupplierParty{Party: c.newParty(inv.Supplier)},
		AccountingCustomerParty: document.CustomerParty{Party: c.newParty(inv.Customer)},
	}

	if len(inv.Notes) > 0 {
		c.doc.Note = make([]string, len(inv.Notes))
		for i, note := range inv.Notes {
			c.doc.Note[i] = note.Text
		}
	}

	err = c.newOrdering(inv.Ordering)
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

func getTypeCode(inv *bill.Invoice) (string, error) {
	if inv.Tax == nil || inv.Tax.Ext == nil || inv.Tax.Ext[untdid.ExtKeyDocumentType].String() == "" {
		return "", fmt.Errorf("validation: invoice must contain document type extension, added automatically with the EN16931 addon")
	}
	return inv.Tax.Ext[untdid.ExtKeyDocumentType].String(), nil
}

func invoiceNumber(series cbc.Code, code cbc.Code) string {
	if series == "" {
		return code.String()
	}
	return fmt.Sprintf("%s-%s", series, code)
}

func formatDate(date cal.Date) string {
	if date.IsZero() {
		return ""
	}
	t := date.Time()
	return t.Format("2006-01-02")
}
