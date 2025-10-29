package ubl

import (
	"github.com/nbio/xml"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

var invoiceTypeMap = map[string]cbc.Key{
	"325": bill.InvoiceTypeProforma,
	"380": bill.InvoiceTypeStandard,
	"381": bill.InvoiceTypeCreditNote,
	"383": bill.InvoiceTypeDebitNote,
	"384": bill.InvoiceTypeCorrective,
	"389": bill.InvoiceTypeStandard,
	"326": bill.InvoiceTypeStandard,
	"261": bill.InvoiceTypeCreditNote,
}

// InvoiceTagMap maps UBL invoice type codes to GOBL tax tags.
var InvoiceTagMap = map[string][]cbc.Key{
	"389": {tax.TagSelfBilled},
	"326": {tax.TagPartial},
	"261": {tax.TagSelfBilled},
}

// parseInvoice takes the provided raw XML document and attempts to build
// a bill.Invoice. Contents may be either an Invoice or CreditNote.
func parseInvoice(data []byte) (*bill.Invoice, error) {
	in := new(Invoice)
	if err := xml.Unmarshal(data, in); err != nil {
		return nil, err
	}
	return goblInvoice(in)
}

func goblInvoice(in *Invoice) (*bill.Invoice, error) {
	out := &bill.Invoice{
		Code:     cbc.Code(in.ID),
		Currency: currency.Code(in.DocumentCurrencyCode),
		Tax: &bill.Tax{
			// Always default to currency rounding for incoming invoices
			// as this is the default for EN16931.
			Rounding: tax.RoundingRuleCurrency,
		},
		Supplier: goblParty(in.AccountingSupplierParty.Party),
		Customer: goblParty(in.AccountingCustomerParty.Party),
	}

	typeCode := in.InvoiceTypeCode
	if typeCode == "" {
		typeCode = in.CreditNoteTypeCode
	}
	out.Type = typeCodeParse(typeCode)
	tags := tagCodeParse(typeCode)

	if len(tags) != 0 {
		out.SetTags(tags...)
	}

	issueDate, err := parseDate(in.IssueDate)
	if err != nil {
		return nil, err
	}
	out.IssueDate = issueDate

	if err := goblAddLines(in, out); err != nil {
		return nil, err
	}
	if err := goblAddPayment(in, out); err != nil {
		return nil, err
	}
	if err = goblAddOrdering(in, out); err != nil {
		return nil, err
	}
	if err = goblAddDelivery(in, out); err != nil {
		return nil, err
	}

	if len(in.Note) > 0 {
		out.Notes = make([]*org.Note, 0, len(in.Note))
		for _, note := range in.Note {
			n := &org.Note{
				Text: note,
			}
			out.Notes = append(out.Notes, n)
		}
	}

	if len(in.BillingReference) > 0 {
		out.Preceding = make([]*org.DocumentRef, 0, len(in.BillingReference))
		for _, ref := range in.BillingReference {
			var docRef *org.DocumentRef
			var err error

			switch {
			case ref.InvoiceDocumentReference != nil:
				docRef, err = goblReference(ref.InvoiceDocumentReference)
			case ref.SelfBilledInvoiceDocumentReference != nil:
				docRef, err = goblReference(ref.SelfBilledInvoiceDocumentReference)
			case ref.CreditNoteDocumentReference != nil:
				docRef, err = goblReference(ref.CreditNoteDocumentReference)
			case ref.AdditionalDocumentReference != nil:
				docRef, err = goblReference(ref.AdditionalDocumentReference)
			}
			if err != nil {
				return nil, err
			}
			if docRef != nil {
				out.Preceding = append(out.Preceding, docRef)
			}
		}
	}

	if in.TaxRepresentativeParty != nil {
		// Move the original seller to the ordering.seller party
		if out.Ordering == nil {
			out.Ordering = &bill.Ordering{}
		}
		out.Ordering.Seller = out.Supplier

		// Overwrite the seller field with the tax representative
		out.Supplier = goblParty(in.TaxRepresentativeParty)
	}

	if len(in.AllowanceCharge) > 0 {
		if err := goblAddCharges(in, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// typeCodeParse maps UBL invoice type to GOBL equivalent.
// Source: https://unece.org/fileadmin/DAM/trade/untdid/d16b/tred/tred1001.htm
func typeCodeParse(typeCode string) cbc.Key {
	if val, ok := invoiceTypeMap[typeCode]; ok {
		return val
	}
	return bill.InvoiceTypeOther
}

// tagCodeParse maps UBL invoice type to GOBL equivalent tax tag.
func tagCodeParse(typeCode string) []cbc.Key {
	if val, ok := InvoiceTagMap[typeCode]; ok {
		return val
	}
	return nil
}
