package ubl

import "encoding/xml"

// CreditNote represents a UBL Credit Note document, with fields declared in the
// exact sequence of the UBL-CreditNote-2.1 XSD.
//
// The Invoice and CreditNote XSDs diverge in more than one place — cbc:TaxPointDate
// precedes the type code, there is no cbc:DueDate, cac:AllowanceCharge follows the
// exchange rates rather than preceding them, the document-reference block is
// ordered differently, and cac:ProjectReference / cac:PrepaidPayment /
// cac:WithholdingTaxTotal do not exist. Rather than juggle one struct for both
// layouts, credit notes are built as an Invoice and mapped to this type at the
// marshalling boundary (see toCreditNote); fields the CreditNote XSD omits are
// simply not carried over.
type CreditNote struct {
	XMLName xml.Name
	documentHeader

	TaxPointDate       string   `xml:"cbc:TaxPointDate,omitempty"`
	CreditNoteTypeCode *IDType  `xml:"cbc:CreditNoteTypeCode,omitempty"`
	Note               []string `xml:"cbc:Note,omitempty"`

	documentCurrency

	OrderReference              *OrderReference     `xml:"cac:OrderReference,omitempty"`
	BillingReference            []*BillingReference `xml:"cac:BillingReference,omitempty"`
	DespatchDocumentReference   []Reference         `xml:"cac:DespatchDocumentReference,omitempty"`
	ReceiptDocumentReference    []Reference         `xml:"cac:ReceiptDocumentReference,omitempty"`
	ContractDocumentReference   []Reference         `xml:"cac:ContractDocumentReference,omitempty"`
	AdditionalDocumentReference []Reference         `xml:"cac:AdditionalDocumentReference,omitempty"`
	StatementDocumentReference  []Reference         `xml:"cac:StatementDocumentReference,omitempty"`
	OriginatorDocumentReference []Reference         `xml:"cac:OriginatorDocumentReference,omitempty"`

	documentParties

	TaxExchangeRate                *ExchangeRate     `xml:"cac:TaxExchangeRate,omitempty"`
	PricingExchangeRate            *ExchangeRate     `xml:"cac:PricingExchangeRate,omitempty"`
	PaymentExchangeRate            *ExchangeRate     `xml:"cac:PaymentExchangeRate,omitempty"`
	PaymentAlternativeExchangeRate *ExchangeRate     `xml:"cac:PaymentAlternativeExchangeRate,omitempty"`
	AllowanceCharge                []AllowanceCharge `xml:"cac:AllowanceCharge,omitempty"`
	TaxTotal                       []TaxTotal        `xml:"cac:TaxTotal,omitempty"`
	LegalMonetaryTotal             MonetaryTotal     `xml:"cac:LegalMonetaryTotal"`
	CreditNoteLines                []InvoiceLine     `xml:"cac:CreditNoteLine,omitempty"`
}

// toCreditNote projects an Invoice built for a credit note onto the CreditNote
// layout. Only the elements the CreditNote XSD allows are carried across; the
// invoice-only fields (DueDate, ProjectReference, PrepaidPayment,
// WithholdingTaxTotal, InvoiceTypeCode) are dropped by construction. Lines are
// taken from whichever slice the builder populated.
func (ui *Invoice) toCreditNote() *CreditNote {
	lines := ui.CreditNoteLines
	if len(lines) == 0 {
		lines = ui.InvoiceLines
	}
	return &CreditNote{
		XMLName:            ui.XMLName,
		documentHeader:     ui.documentHeader,
		TaxPointDate:       ui.TaxPointDate,
		CreditNoteTypeCode: ui.CreditNoteTypeCode,
		Note:               ui.Note,
		documentCurrency:   ui.documentCurrency,

		OrderReference:              ui.OrderReference,
		BillingReference:            ui.BillingReference,
		DespatchDocumentReference:   ui.DespatchDocumentReference,
		ReceiptDocumentReference:    ui.ReceiptDocumentReference,
		ContractDocumentReference:   ui.ContractDocumentReference,
		AdditionalDocumentReference: ui.AdditionalDocumentReference,
		StatementDocumentReference:  ui.StatementDocumentReference,
		OriginatorDocumentReference: ui.OriginatorDocumentReference,

		documentParties: ui.documentParties,

		TaxExchangeRate:                ui.TaxExchangeRate,
		PricingExchangeRate:            ui.PricingExchangeRate,
		PaymentExchangeRate:            ui.PaymentExchangeRate,
		PaymentAlternativeExchangeRate: ui.PaymentAlternativeExchangeRate,
		AllowanceCharge:                ui.AllowanceCharge,
		TaxTotal:                       ui.TaxTotal,
		LegalMonetaryTotal:             ui.LegalMonetaryTotal,
		CreditNoteLines:                lines,
	}
}

// marshalDocument returns the value to hand to encoding/xml: a credit-note
// Invoice is remapped onto its CreditNote layout so the emitted element sequence
// matches the UBL-CreditNote XSD without any post-marshal byte surgery.
func marshalDocument(in any) any {
	var inv *Invoice
	switch v := in.(type) {
	case *Invoice:
		inv = v
	case Invoice:
		inv = &v
	default:
		return in
	}
	if inv.XMLName.Local == rootNameCreditNote {
		return inv.toCreditNote()
	}
	return in
}
