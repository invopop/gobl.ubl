package ubl

import (
	"github.com/invopop/gobl"
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

// Convert converts the UBL Invoice to a GOBL envelope.
// It automatically detects the context based on CustomizationID and ProfileID.
// Binary attachments are ignored during conversion - use ExtractBinaryAttachments
// to retrieve them separately.
func (in *Invoice) Convert() (*gobl.Envelope, error) {
	o := new(options)

	// Detect context from the invoice
	ctx := FindContext(in.CustomizationID, in.ProfileID)
	if ctx != nil {
		o.context = *ctx
	}

	inv, err := goblInvoice(in, o)
	if err != nil {
		return nil, err
	}

	env := gobl.NewEnvelope()
	if err := env.Insert(inv); err != nil {
		return nil, err
	}

	return env, nil
}

func goblInvoice(in *Invoice, o *options) (*bill.Invoice, error) {
	out := &bill.Invoice{
		Addons: tax.Addons{
			List: o.context.Addons,
		},
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

	if err := in.goblAddLines(out); err != nil {
		return nil, err
	}
	if err := in.goblAddPayment(out); err != nil {
		return nil, err
	}
	if err = in.goblAddOrdering(out); err != nil {
		return nil, err
	}
	if err = in.goblAddDelivery(out); err != nil {
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
		if err := in.goblAddCharges(out); err != nil {
			return nil, err
		}
	}

	for _, ref := range in.AdditionalDocumentReference {
		att, err := goblAddAttachments(ref)
		if err != nil {
			return nil, err
		}

		if att != nil {
			out.Attachments = append(out.Attachments, att)
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
