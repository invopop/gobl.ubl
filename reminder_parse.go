package ubl

import (
	"github.com/invopop/gobl"
	oioubl "github.com/invopop/gobl.dk.oioubl/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/gobl/uuid"
)

// Convert turns a parsed UBL Reminder into a GOBL envelope wrapping a
// bill.Payment of type "request".
func (rem *Reminder) Convert() (*gobl.Envelope, error) {
	o := new(options)
	profileID := ""
	if rem.ProfileID != nil {
		profileID = rem.ProfileID.Value
	}
	ctx := FindContext(rem.CustomizationID, profileID)
	if ctx == nil {
		ctx = FindContext(rem.CustomizationID, "")
	}
	if ctx != nil {
		o.context = *ctx
	}

	pmt, err := rem.goblPayment(o)
	if err != nil {
		return nil, err
	}

	env := gobl.NewEnvelope()
	if err := env.Insert(pmt); err != nil {
		return nil, err
	}
	return env, nil
}

func (rem *Reminder) goblPayment(o *options) (*bill.Payment, error) {
	out := &bill.Payment{
		Addons:   tax.Addons{List: o.context.Addons},
		Type:     bill.PaymentTypeRequest,
		Code:     cbc.Code(rem.ID),
		Currency: currency.Code(rem.DocumentCurrencyCode),
		Supplier: goblParty(rem.AccountingSupplierParty.Party, o),
		Customer: goblParty(rem.AccountingCustomerParty.Party, o),
	}
	if rem.PayeeParty != nil {
		out.Payee = goblParty(rem.PayeeParty, o)
	}

	issueDate, err := parseDate(rem.IssueDate)
	if err != nil {
		return nil, err
	}
	out.IssueDate = issueDate

	for _, n := range rem.Note {
		out.Notes = append(out.Notes, &org.Note{Text: n})
	}

	for _, rl := range rem.ReminderLine {
		line, err := rem.goblPaymentLine(rl)
		if err != nil {
			return nil, err
		}
		out.Lines = append(out.Lines, line)
	}

	for i := range rem.PaymentMeans {
		if m := goblReminderMethod(&rem.PaymentMeans[i], o.context); m != nil {
			out.Methods = append(out.Methods, m)
		}
	}

	if o.context.Is(ContextOIOUBL21) {
		applyOIOUBL21ReminderParse(out, rem)
	}

	return out, nil
}

func (rem *Reminder) goblPaymentLine(rl ReminderLine) (*bill.PaymentLine, error) {
	amount, err := num.AmountFromString(normalizeNumericString(rl.DebitLineAmount.Value))
	if err != nil {
		return nil, err
	}
	line := &bill.PaymentLine{Amount: amount}

	if br := rl.BillingReference; br != nil && br.InvoiceDocumentReference != nil {
		ref := br.InvoiceDocumentReference
		doc := &org.DocumentRef{Code: cbc.Code(ref.ID.Value)}
		if ref.UUID != "" {
			doc.UUID = uuid.UUID(ref.UUID)
		}
		if ref.IssueDate != "" {
			d, err := parseDate(ref.IssueDate)
			if err != nil {
				return nil, err
			}
			doc.IssueDate = &d
		}
		line.Document = doc
	}

	return line, nil
}

// goblReminderMethod reconstructs a payment Record from a cac:PaymentMeans, the
// mirror of addReminderPaymentMeans. The amount is left for Calculate to fill
// from the document total.
func goblReminderMethod(pm *PaymentMeans, ctx Context) *pay.Record {
	channel := ""
	if pm.PaymentChannelCode != nil {
		channel = pm.PaymentChannelCode.Value
	}
	if ctx.Is(ContextOIOUBL21) && (channel == oioubl21PaymentChannelGiro || channel == oioubl21PaymentChannelFIK) {
		return goblReminderGiroFIKMethod(pm, channel)
	}

	code := pm.PaymentMeansCode.Value
	if ctx.Is(ContextOIOUBL21) && code == "31" {
		code = "30"
	}
	key := goblPaymentMeansCode(code)
	if !key.HasPrefix(pay.MeansKeyCreditTransfer) {
		return nil
	}
	rec := &pay.Record{Key: key}
	if pm.PayeeFinancialAccount != nil {
		rec.CreditTransfer = goblCreditTransfer(pm)[0]
	}
	return rec
}

// goblReminderGiroFIKMethod reverses an OIOUBL Giro/FIK payment means. The OIOUBL
// means code and channel are preserved on a generic "other" record, the kortart
// and payment number are recovered from cbc:PaymentID / cbc:InstructionID, and
// the FIK creditor account from cac:CreditAccount.
func goblReminderGiroFIKMethod(pm *PaymentMeans, channel string) *pay.Record {
	rec := &pay.Record{
		Key: pay.MeansKeyOther,
		Ext: tax.ExtensionsOf(cbc.CodeMap{
			untdid.ExtKeyPaymentMeans:   cbc.Code(pm.PaymentMeansCode.Value),
			oioubl.ExtKeyPaymentChannel: cbc.Code(channel),
		}),
	}
	if pm.PaymentID != nil {
		rec.Ext = rec.Ext.Set(oioubl.ExtKeyPaymentID, cbc.Code(*pm.PaymentID))
	}
	if pm.InstructionID != nil {
		rec.Ref = cleanString(*pm.InstructionID)
	}
	switch {
	case pm.CreditAccount != nil && pm.CreditAccount.AccountID != "":
		rec.CreditTransfer = &pay.CreditTransfer{Number: pm.CreditAccount.AccountID}
	case pm.PayeeFinancialAccount != nil:
		rec.CreditTransfer = goblCreditTransfer(pm)[0]
	}
	return rec
}

// applyOIOUBL21ReminderParse restores the reminder type and sequence onto the
// payment extensions the emit side reads, so they round-trip.
func applyOIOUBL21ReminderParse(pmt *bill.Payment, rem *Reminder) {
	if rem.ReminderTypeCode != nil && rem.ReminderTypeCode.Value != "" {
		pmt.Ext = pmt.Ext.Set(oioubl.ExtKeyReminderType, cbc.Code(rem.ReminderTypeCode.Value))
	}
	if rem.ReminderSequenceNumeric != "" {
		pmt.Ext = pmt.Ext.Set(oioubl.ExtKeyReminderSequence, cbc.Code(rem.ReminderSequenceNumeric))
	}
}
