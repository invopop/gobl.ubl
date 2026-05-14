package ubl

import (
	"strings"

	"cloud.google.com/go/civil"
	"github.com/invopop/gobl"
	"github.com/invopop/gobl/addons/fr/ctc"
	"github.com/invopop/gobl/addons/sa/zatca"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
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
	"388": bill.InvoiceTypeStandard,
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
func (ui *Invoice) Convert() (*gobl.Envelope, error) {
	o := new(options)

	// Detect context from the invoice
	ctx := FindContext(ui.CustomizationID, ui.ProfileID)
	if ctx != nil {
		o.context = *ctx
	}

	inv, err := ui.goblInvoice(o)
	if err != nil {
		return nil, err
	}

	env := gobl.NewEnvelope()
	if err := env.Insert(inv); err != nil {
		return nil, err
	}

	return env, nil
}

func (ui *Invoice) goblInvoice(o *options) (*bill.Invoice, error) {
	out := &bill.Invoice{
		Addons: tax.Addons{
			List: o.context.Addons,
		},
		Code:     cbc.Code(ui.ID),
		Currency: currency.Code(ui.DocumentCurrencyCode),
		Tax: &bill.Tax{
			// Always default to currency rounding for incoming invoices
			// as this is the default for EN16931.
			Rounding: tax.RoundingRuleCurrency,
		},
		Supplier: goblParty(ui.AccountingSupplierParty.Party, o),
		Customer: goblParty(ui.AccountingCustomerParty.Party, o),
	}

	if o.context.Is(ContextPeppolFranceCIUS) || o.context.Is(ContextPeppolFranceExtended) {
		out.Tax.Ext = out.Tax.Ext.Set(ctc.ExtKeyBillingMode, cbc.Code(ui.ProfileID))
	}

	if o.context.Is(ContextZATCA) && ui.InvoiceTypeCode != nil && ui.InvoiceTypeCode.Name != nil {
		out.Tax.Ext = out.Tax.Ext.Set(zatca.ExtKeyInvoiceTypeTransactions, cbc.Code(*ui.InvoiceTypeCode.Name))
	}

	var typeCode *IDType
	if ui.InvoiceTypeCode != nil {
		typeCode = ui.InvoiceTypeCode
	} else {
		typeCode = ui.CreditNoteTypeCode
	}
	out.Type = typeCodeParse(typeCode, o.context)
	tags := tagCodeParse(typeCode, o.context)

	if len(tags) != 0 {
		out.SetTags(tags...)
	}

	issueDate, err := parseDate(ui.IssueDate)
	if err != nil {
		return nil, err
	}
	out.IssueDate = issueDate

	if ui.IssueTime != "" {
		ct, err := civil.ParseTime(ui.IssueTime)
		if err != nil {
			return nil, err
		}
		t := cal.Time{Time: ct}
		out.IssueTime = &t
	}

	// BT-7: VAT point date
	if ui.TaxPointDate != "" {
		vd, err := parseDate(ui.TaxPointDate)
		if err != nil {
			return nil, err
		}
		out.ValueDate = &vd
	}

	if ui.TaxCurrencyCode != "" && ui.DocumentCurrencyCode != ui.TaxCurrencyCode {
		out.ExchangeRates = goblExchangeRates(
			currency.Code(ui.DocumentCurrencyCode),
			currency.Code(ui.TaxCurrencyCode),
			ui.TaxTotal,
		)
	}

	if err := ui.goblAddLines(out); err != nil {
		return nil, err
	}
	if err := ui.goblAddPayment(out, o); err != nil {
		return nil, err
	}
	if err = ui.goblAddOrdering(out); err != nil {
		return nil, err
	}
	if err = ui.goblAddDelivery(out); err != nil {
		return nil, err
	}

	if len(ui.Note) > 0 {
		out.Notes = make([]*org.Note, 0, len(ui.Note))
		for _, note := range ui.Note {
			out.Notes = append(out.Notes, parseNote(note))
		}
	}

	if len(ui.BillingReference) > 0 {
		out.Preceding = make([]*org.DocumentRef, 0, len(ui.BillingReference))
		for _, ref := range ui.BillingReference {
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

	// BR-KSA-17: In ZATCA, preceding document reasons are stored
	// in PaymentMeans InstructionNote.
	if o.context.Is(ContextZATCA) && len(out.Preceding) > 0 && len(ui.PaymentMeans) > 0 {
		notes := ui.PaymentMeans[0].InstructionNote
		for i, note := range notes {
			if i < len(out.Preceding) {
				out.Preceding[i].Reason = note
			}
		}
	}

	if ui.TaxRepresentativeParty != nil {
		// Move the original seller to the ordering.seller party
		if out.Ordering == nil {
			out.Ordering = &bill.Ordering{}
		}
		out.Ordering.Seller = out.Supplier

		// Overwrite the seller field with the tax representative
		out.Supplier = goblParty(ui.TaxRepresentativeParty, o)
	}

	if len(ui.AllowanceCharge) > 0 {
		if err := ui.goblAddCharges(out); err != nil {
			return nil, err
		}
	}

	out.Attachments = ui.goblAddAttachments()

	ui.goblAddTaxNotes(out)

	return out, nil
}

// typeCodeParse maps UBL invoice type to GOBL equivalent.
// Source: https://unece.org/fileadmin/DAM/trade/untdid/d16b/tred/tred1001.htm
func typeCodeParse(typeCode *IDType, ctx Context) cbc.Key {
	if typeCode == nil {
		return bill.InvoiceTypeOther
	}
	if ctx.Is(ContextZATCA) && typeCode.Name != nil {
		code := *typeCode.Name
		if len(code) < 7 || code[0] != '0' || code[2] != '0' || code[3] != '0' || code[6] != '0' {
			return bill.InvoiceTypeOther
		}
	}

	if val, ok := invoiceTypeMap[typeCode.Value]; ok {
		return val
	}
	return bill.InvoiceTypeOther
}

// tagCodeParse maps UBL invoice type to GOBL equivalent tax tag.
func tagCodeParse(typeCode *IDType, ctx Context) []cbc.Key {
	var tags []cbc.Key
	if typeCode == nil {
		return tags
	}

	if ctx.Is(ContextZATCA) && typeCode.Name != nil {
		transactionTypeCode := *typeCode.Name
		if len(transactionTypeCode) < 7 || transactionTypeCode[0] != '0' {
			return tags
		}

		if strings.HasPrefix(transactionTypeCode, "02") {
			tags = append(tags, tax.TagSimplified)
		}

		if transactionTypeCode[4] == '1' {
			tags = append(tags, tax.TagExport)
		}

		if transactionTypeCode[5] == '1' {
			tags = append(tags, zatca.TagSummary)
		}

	} else {
		tags = InvoiceTagMap[typeCode.Value]
	}
	return tags
}
