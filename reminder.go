package ubl

import (
	"encoding/xml"
	"strconv"

	oioubl "github.com/invopop/gobl.dk.oioubl/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
)

// NamespaceUBLReminder is the UBL 2.1 Reminder root namespace.
const NamespaceUBLReminder = "urn:oasis:names:specification:ubl:schema:xsd:Reminder-2"

// Reminder is a UBL 2.1 Reminder, the OIOUBL dunning document (Rykker) mapped
// from a bill.Payment of type "request".
type Reminder struct {
	XMLName      xml.Name
	CBCNamespace string `xml:"xmlns:cbc,attr"`
	CACNamespace string `xml:"xmlns:cac,attr"`
	UBLNamespace string `xml:"xmlns,attr"`

	UBLVersionID    string  `xml:"cbc:UBLVersionID,omitempty"`
	CustomizationID string  `xml:"cbc:CustomizationID,omitempty"`
	ProfileID       *IDType `xml:"cbc:ProfileID,omitempty"`
	ID              string  `xml:"cbc:ID"`
	CopyIndicator   string  `xml:"cbc:CopyIndicator,omitempty"`
	UUID            string  `xml:"cbc:UUID,omitempty"`
	IssueDate       string  `xml:"cbc:IssueDate"`
	IssueTime       string  `xml:"cbc:IssueTime,omitempty"`

	ReminderTypeCode        *IDType `xml:"cbc:ReminderTypeCode,omitempty"`
	ReminderSequenceNumeric string  `xml:"cbc:ReminderSequenceNumeric,omitempty"`
	DocumentCurrencyCode    string  `xml:"cbc:DocumentCurrencyCode,omitempty"`

	Note []string `xml:"cbc:Note,omitempty"`

	AccountingSupplierParty SupplierParty  `xml:"cac:AccountingSupplierParty"`
	AccountingCustomerParty CustomerParty  `xml:"cac:AccountingCustomerParty"`
	PayeeParty              *Party         `xml:"cac:PayeeParty,omitempty"`
	TaxTotal                []TaxTotal     `xml:"cac:TaxTotal,omitempty"`
	LegalMonetaryTotal      MonetaryTotal  `xml:"cac:LegalMonetaryTotal"`
	ReminderLine            []ReminderLine `xml:"cac:ReminderLine"`
}

// ReminderLine restates one outstanding amount and references the document it concerns.
type ReminderLine struct {
	ID               string            `xml:"cbc:ID"`
	DebitLineAmount  Amount            `xml:"cbc:DebitLineAmount"`
	BillingReference *BillingReference `xml:"cac:BillingReference,omitempty"`
}

func ublReminder(pmt *bill.Payment, o *options) *Reminder {
	currency := pmt.Currency.String()

	// OIOUBL declares UBLVersionID 2.0 on the wire; other UBL contexts keep 2.1.
	ublVersion := Version
	if o.context.Is(ContextOIOUBL21) {
		ublVersion = oioublUBLVersion
	}

	out := &Reminder{
		XMLName:                 xml.Name{Local: "Reminder"},
		CBCNamespace:            NamespaceCBC,
		CACNamespace:            NamespaceCAC,
		UBLNamespace:            NamespaceUBLReminder,
		UBLVersionID:            ublVersion,
		CustomizationID:         o.context.CustomizationID,
		ID:                      invoiceNumber(pmt.Series, pmt.Code),
		IssueDate:               formatDate(pmt.IssueDate),
		DocumentCurrencyCode:    currency,
		AccountingSupplierParty: SupplierParty{Party: newParty(pmt.Supplier, o.context)},
		AccountingCustomerParty: CustomerParty{Party: newParty(pmt.Customer, o.context)},
	}
	if o.context.ProfileID != "" {
		out.ProfileID = &IDType{Value: o.context.ProfileID}
	}
	if !pmt.UUID.IsZero() {
		out.UUID = pmt.UUID.String()
	}
	if pmt.IssueTime != nil {
		out.IssueTime = pmt.IssueTime.String()
	}
	for _, n := range pmt.Notes {
		if n != nil && n.Text != "" {
			out.Note = append(out.Note, n.Text)
		}
	}
	if pmt.Payee != nil {
		out.PayeeParty = newParty(pmt.Payee, o.context)
	}

	out.addReminderLines(pmt, currency)
	out.addReminderTotals(pmt, currency)

	if o.context.Is(ContextOIOUBL21) {
		applyOIOUBL21Reminder(out, pmt)
	}

	return out
}

// addReminderLines builds one ReminderLine per payment line.
func (rem *Reminder) addReminderLines(pmt *bill.Payment, currency string) {
	for _, l := range pmt.Lines {
		if l == nil {
			continue
		}
		line := ReminderLine{
			ID:              strconv.Itoa(l.Index),
			DebitLineAmount: Amount{Value: l.Amount.String(), CurrencyID: &currency},
		}
		if l.Document != nil {
			line.BillingReference = &BillingReference{
				InvoiceDocumentReference: reminderDocumentReference(l.Document),
			}
		}
		rem.ReminderLine = append(rem.ReminderLine, line)
	}
}

// addReminderTotals builds the LegalMonetaryTotal. A reminder restates
// already-taxed amounts, so it levies no tax of its own: TaxExclusiveAmount
// (OIOUBL reads this as the reminder's own tax, F-REM079) is zero and every
// other total equals the debit-line sum.
func (rem *Reminder) addReminderTotals(pmt *bill.Payment, currency string) {
	exp := pmt.Total.Exp()
	sum := num.MakeAmount(0, exp)
	for _, l := range pmt.Lines {
		if l != nil {
			sum = sum.Add(l.Amount)
		}
	}
	zero := num.MakeAmount(0, exp)
	rem.LegalMonetaryTotal = MonetaryTotal{
		LineExtensionAmount: Amount{Value: sum.String(), CurrencyID: &currency},
		TaxExclusiveAmount:  Amount{Value: zero.String(), CurrencyID: &currency},
		TaxInclusiveAmount:  Amount{Value: sum.String(), CurrencyID: &currency},
		PayableAmount:       &Amount{Value: sum.String(), CurrencyID: &currency},
	}
}

// reminderDocumentReference maps a paid document to a UBL Reference.
func reminderDocumentReference(doc *org.DocumentRef) *Reference {
	ref := &Reference{
		ID: IDType{Value: invoiceNumber(doc.Series, doc.Code)},
	}
	if !doc.UUID.IsZero() {
		ref.UUID = doc.UUID.String()
	}
	if doc.IssueDate != nil {
		ref.IssueDate = formatDate(*doc.IssueDate)
	}
	return ref
}

// OIOUBL 2.1 Reminder specifics follow.

// Reminders ride the same profile5:ver2.0 / profileid-1.2 profile as invoices.
const (
	reminderTypeCodeListID = "urn:oioubl:codelist:remindertypecode-1.1"
	oioublProfileSchemeV12 = "urn:oioubl:id:profileid-1.2"
)

// applyOIOUBL21Reminder stamps the OIOUBL specifics: party formatting, the
// profileid scheme attributes, and the reminder type (F-REM006/061) and
// sequence (F-REM007) from the payment extensions.
func applyOIOUBL21Reminder(out *Reminder, pmt *bill.Payment) {
	applyOIOUBL21Party(out.AccountingSupplierParty.Party)
	applyOIOUBL21Party(out.AccountingCustomerParty.Party)
	if out.PayeeParty != nil {
		applyOIOUBL21Party(out.PayeeParty)
	}

	if out.ProfileID != nil {
		schemeID := oioublProfileSchemeV12
		agencyID := oioublCodeListAgencyID
		out.ProfileID.SchemeID = &schemeID
		out.ProfileID.SchemeAgencyID = &agencyID
	}

	if code := pmt.Ext.Get(oioubl.ExtKeyReminderType); code != "" {
		agencyID := oioublCodeListAgencyID
		listID := reminderTypeCodeListID
		out.ReminderTypeCode = &IDType{
			ListAgencyID: &agencyID,
			ListID:       &listID,
			Value:        code.String(),
		}
	}
	out.ReminderSequenceNumeric = pmt.Ext.Get(oioubl.ExtKeyReminderSequence).String()
}
