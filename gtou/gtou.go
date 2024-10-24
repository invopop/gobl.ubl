// Package gtou provides a conversor from GOBL to UBL.
package gtou

import (
	"fmt"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/tax"
)

// Conversor is a struct that contains the necessary elements to convert between GOBL and UBL
type Conversor struct {
	doc *Document
}

// NewConversor creates a new Conversor instance
func NewConversor() *Conversor {
	c := new(Conversor)
	c.doc = new(Document)
	return c
}

// GetDocument returns the document from the conversor
func (c *Conversor) GetDocument() *Document {
	return c.doc
}

// ConvertToUBL converts a GOBL envelope into a UBL document
func (c *Conversor) ConvertToUBL(env *gobl.Envelope) (*Document, error) {
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, fmt.Errorf("invalid type %T", env.Document)
	}

	// Create the UBL document
	doc := &Document{
		CACNamespace:         CAC,
		CBCNamespace:         CBC,
		QDTNamespace:         QDT,
		UDTNamespace:         UDT,
		CCTSNamespace:        CCTS,
		CustomizationID:      "urn:cen.eu:en16931:2017",
		ProfileID:            "Invoicing on purchase order",
		ID:                   invoiceNumber(inv.Series, inv.Code),
		IssueDate:            formatDate(inv.IssueDate),
		InvoiceTypeCode:      invoiceTypeCode(inv),
		DocumentCurrencyCode: string(inv.Currency),
		AccountingSupplierParty: newSupplier(inv.Supplier),
		AccountingCustomerParty: newCustomer(inv.Customer),
		// Delivery:                createDelivery(inv.Delivery),
		LegalMonetaryTotal: createMonetaryTotal(inv.MonetaryTotal),
		InvoiceLine:        createInvoiceLines(inv.Lines),
	}
	OrderReference:       createOrderReference(inv),
	ContractDocumentReference: []DocumentReference{
		{ID: "Contract321"},
	},
	AdditionalDocumentReference: []DocumentReference{
		{ID: "Doc1", DocumentDescription: "Timesheet"},
		{ID: "Doc2", DocumentDescription: "EHF specification"},
	},
	AccountingCost:       "Project cost code 123",
	Note:                 []string{"Ordered in our booth at the convention"},
	DueDate:              formatDate(inv.DueDate),
	PaymentMeans:       createPaymentMeans(inv.PaymentMeans),
	PaymentTerms:       createPaymentTerms(inv.PaymentTerms),
	AllowanceCharge:    createAllowanceCharges(inv.AllowanceCharges),
	TaxTotal:           createTaxTotals(inv.TaxTotals),

	if len(inv.Payment.Terms.DueDates) > 0 {
		doc.DueDate = formatDate(inv.Payment.Terms.DueDates[0])
	}

	if inv.Payment != nil && inv.Payment.Payee != nil {
		doc.PayeeParty = createPayeeParty(inv.Payment.Payee)
	}

	// If both ordering.seller and seller are present, the original seller is used
	// as the tax representative.
	if inv.Ordering != nil && inv.Ordering.Seller != nil {
		doc.TaxRepresentativeParty = doc.Seller.Party
		doc.Seller = newSeller(inv.Ordering.Seller)
	}

	if inv.Ordering != nil && inv.Ordering.Period != nil {
		doc.InvoicePeriod = []Period{makePeriod(inv.Ordering.Period)}
	}

	err := c.createCustomerParty(inv.Customer)
	if err != nil {
		return nil, err
	}

	err = c.createDelivery(inv.Delivery)
	if err != nil {
		return nil, err
	}

	return doc, nil
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