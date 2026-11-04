package ubl

import (
	"regexp"
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/pay"
)

var (
	paymentMeansMap = map[string]cbc.Key{
		"10": pay.MeansKeyCash,
		"20": pay.MeansKeyCheque,
		"30": pay.MeansKeyCreditTransfer,
		// Payment to bank account has no typed key, and "other" keeps the
		// code from being rewritten by the EN 16931 key normalization.
		"42": pay.MeansKeyOther,
		"48": pay.MeansKeyCard,
		"49": pay.MeansKeyDirectDebit,
		"58": pay.MeansKeyCreditTransfer.With(pay.MeansKeySEPA),
		"59": pay.MeansKeyDirectDebit.With(pay.MeansKeySEPA),
	}

	// ibanRegex matches IBAN-like format: starts with 2+ letters followed by digits/alphanumeric
	// Allows optional whitespace throughout (IBANs are often formatted with spaces)
	ibanRegex = regexp.MustCompile(`^[A-Z]{2,}\s*[0-9A-Z\s]+$`)

	// bicRegex matches an ISO 9362 BIC: 6 letters followed by 2 or 5
	// alphanumeric characters
	bicRegex = regexp.MustCompile(`^[A-Z]{6}[A-Z0-9]{2}([A-Z0-9]{3})?$`)
)

func (ui *Invoice) goblAddPayment(out *bill.Invoice) error {
	payment := &bill.PaymentDetails{}

	if ui.PayeeParty != nil {
		payment.Payee = goblParty(ui.PayeeParty)
	}

	if ui.PaymentTerms != nil && ui.PaymentTerms.Note != "" {
		payment.Terms = &pay.Terms{
			Notes: cleanString(ui.PaymentTerms.Note),
		}
	}

	var dueDate string
	if ui.CreditNoteTypeCode == nil && ui.DueDate != "" {
		dueDate = ui.DueDate
	}
	if ui.CreditNoteTypeCode != nil && len(ui.PaymentMeans) > 0 && ui.PaymentMeans[0].PaymentDueDate != nil {
		dueDate = *ui.PaymentMeans[0].PaymentDueDate
	}

	if dueDate != "" {
		d, err := parseDate(dueDate)
		if err != nil {
			return err
		}
		if payment.Terms == nil {
			payment.Terms = &pay.Terms{}
		}
		payment.Terms.DueDates = append(payment.Terms.DueDates, &pay.DueDate{
			Date: &d,
		})
	}

	// If there's only one due date, set its percent to 100
	if payment.Terms != nil && len(payment.Terms.DueDates) == 1 {
		percent, err := num.PercentageFromString("100%")
		if err != nil {
			return err
		}
		payment.Terms.DueDates[0].Percent = &percent
	}

	if len(ui.PaymentMeans) > 0 {
		payment.Instructions = goblInvoiceInstructions(out, &ui.PaymentMeans[0])
	}

	// We do not currently map this as Peppol and EN16931 do not use it.
	/*
		if len(in.PrepaidPayment) > 0 {
			payment.Advances = make([]*pay.Record, 0, len(in.PrepaidPayment))
			for _, p := range in.PrepaidPayment {
				amount, err := num.AmountFromString(normalizeNumericString(p.PaidAmount.Value))
				if err != nil {
					return err
				}
				advance := &pay.Record{
					Amount: amount,
				}
				if p.ReceivedDate != nil {
					d, err := parseDate(*p.ReceivedDate)
					if err != nil {
						return err
					}
					advance.Date = &d
				}
				payment.Advances = append(payment.Advances, advance)
			}
			}
	*/

	if ui.LegalMonetaryTotal.PrepaidAmount != nil && ui.LegalMonetaryTotal.PrepaidAmount.Value != "" {
		totalPrepaid, err := num.AmountFromString(normalizeNumericString(ui.LegalMonetaryTotal.PrepaidAmount.Value))
		if err != nil {
			return err
		}

		advance := &pay.Record{
			Amount:      totalPrepaid,
			Description: "Prepaid Amount",
		}
		payment.Advances = append(payment.Advances, advance)

	}

	if payment.Payee != nil || payment.Payer != nil || payment.Terms != nil || payment.Instructions != nil || len(payment.Advances) > 0 {
		out.Payment = payment
	}
	return nil
}

func goblInvoiceInstructions(out *bill.Invoice, paymentMeans *PaymentMeans) *pay.Instructions {
	instructions := &pay.Instructions{
		Key: goblPaymentMeansCode(paymentMeans.PaymentMeansCode.Value),
	}
	instructions.Ext = instructions.Ext.Set(untdid.ExtKeyPaymentMeans, cbc.Code(paymentMeans.PaymentMeansCode.Value))

	if paymentMeans.PaymentMeansCode.Name != nil {
		instructions.Detail = cleanString(*paymentMeans.PaymentMeansCode.Name)
	}

	if paymentMeans.PaymentID != nil {
		instructions.Ref = cbc.Code(*paymentMeans.PaymentID)
	}

	if paymentMeans.PayeeFinancialAccount != nil {
		instructions.CreditTransfer = goblCreditTransfer(paymentMeans)
	}
	// A mandate holding only the extended-profile PayerParty is not a
	// direct-debit arrangement.
	if pm := paymentMeans.PaymentMandate; pm != nil && (pm.ID != nil || pm.PayerFinancialAccount != nil) {
		instructions.DirectDebit = goblInvoiceDirectDebit(out, paymentMeans)
	}
	if paymentMeans.CardAccount != nil {
		instructions.Card = goblCard(paymentMeans)
	}

	return instructions
}

func goblCreditTransfer(paymentMeans *PaymentMeans) []*pay.CreditTransfer {
	creditTransfer := &pay.CreditTransfer{}
	account := paymentMeans.PayeeFinancialAccount

	if account.ID != nil {
		id := *account.ID
		if isIBAN(id) {
			creditTransfer.IBAN = cbc.Code(id)
		} else {
			creditTransfer.Number = cbc.Code(id)
		}
	}
	if account.Name != nil {
		creditTransfer.Name = cleanString(*account.Name)
	}
	if branch := account.FinancialInstitutionBranch; branch != nil && branch.ID != nil {
		// BT-86 holds a BIC, or on a domestic transfer a national clearing
		// code such as a Danish reg. no. Some senders still use a BIC with
		// a domestic transfer code, so the shape decides there.
		id := *branch.ID
		if isDomesticTransfer(paymentMeans.PaymentMeansCode.Value) && !isBIC(id) {
			creditTransfer.Clearing = cbc.Code(id)
		} else {
			creditTransfer.BIC = cbc.Code(id)
		}
	}

	if creditTransfer.IBAN == "" && creditTransfer.Number == "" && creditTransfer.Name == "" && creditTransfer.BIC == "" {
		return nil
	}
	return []*pay.CreditTransfer{creditTransfer}
}

// isIBAN checks if a string looks like an IBAN
// Returns true if the string starts with 2+ letters followed by alphanumeric characters
// This covers standard IBANs (e.g., NO9386011117947 or NO93 8601 1117 947) and allows
// some flexibility for various IBAN-like formats that may appear in UBL documents
func isIBAN(s string) bool {
	s = strings.ToUpper(strings.TrimSpace(s))
	return ibanRegex.MatchString(s)
}

// isBIC checks if a string looks like a BIC (ISO 9362): a 4-letter bank code,
// a 2-letter country code, a 2-character location code and an optional
// 3-character branch code.
func isBIC(s string) bool {
	s = strings.ToUpper(strings.TrimSpace(s))
	return bicRegex.MatchString(s)
}

func goblInvoiceDirectDebit(out *bill.Invoice, paymentMeans *PaymentMeans) *pay.DirectDebit {
	directDebit := &pay.DirectDebit{}

	if paymentMeans.PaymentMandate.ID != nil {
		directDebit.Ref = cbc.Code(paymentMeans.PaymentMandate.ID.Value)
	}
	if paymentMeans.PaymentMandate.PayerFinancialAccount != nil && paymentMeans.PaymentMandate.PayerFinancialAccount.ID != nil {
		directDebit.Account = *paymentMeans.PaymentMandate.PayerFinancialAccount.ID
	}
	seller := out.Supplier
	if seller != nil {
		for _, id := range seller.Identities {
			if id.Label == "SEPA" {
				directDebit.Creditor = id.Code.String()
				break
			}
		}
	}
	payment := out.Payment
	if payment != nil && payment.Payee != nil {
		payee := payment.Payee
		for _, id := range payee.Identities {
			if id.Label == "SEPA" {
				directDebit.Creditor = id.Code.String()
				break
			}
		}
	}
	return directDebit
}

func goblCard(paymentMeans *PaymentMeans) *pay.Card {
	card := &pay.Card{}
	if paymentMeans.CardAccount.PrimaryAccountNumberID != nil {
		pan := *paymentMeans.CardAccount.PrimaryAccountNumberID
		if len(pan) >= 4 {
			pan = pan[len(pan)-4:]
		}
		card.Last4 = pan
	}
	if paymentMeans.CardAccount.HolderName != nil {
		card.Holder = *paymentMeans.CardAccount.HolderName
	}
	if card.Last4 == "" && card.Holder == "" {
		return nil
	}
	return card
}

// goblPaymentMeansCode maps UBL payment means to GOBL equivalent.
func goblPaymentMeansCode(code string) cbc.Key {
	if val, ok := paymentMeansMap[code]; ok {
		return val
	}
	return pay.MeansKeyAny
}
