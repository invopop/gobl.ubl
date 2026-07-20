package ubl

import (
	"regexp"
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
)

var (
	paymentMeansMap = map[string]cbc.Key{
		"10": pay.MeansKeyCash,
		"20": pay.MeansKeyCheque,
		"30": pay.MeansKeyCreditTransfer,
		"42": pay.MeansKeyDebitTransfer,
		"48": pay.MeansKeyCard,
		"49": pay.MeansKeyDirectDebit,
		"58": pay.MeansKeyCreditTransfer.With(pay.MeansKeySEPA),
		"59": pay.MeansKeyDirectDebit.With(pay.MeansKeySEPA),
	}

	// ibanRegex matches IBAN-like format: starts with 2+ letters followed by digits/alphanumeric
	// Allows optional whitespace throughout (IBANs are often formatted with spaces)
	ibanRegex = regexp.MustCompile(`^[A-Z]{2,}\s*[0-9A-Z\s]+$`)
)

func (ui *Invoice) goblAddPayment(out *bill.Invoice, o *options) error {
	payment := &bill.PaymentDetails{}

	if ui.PayeeParty != nil {
		payment.Payee = goblParty(ui.PayeeParty, o)
	}

	// The payer (EXT-FR-FE-BG-02) is only defined in the French extended
	// profile, which maps it to the PaymentMandate's PayerParty.
	if o.context.Is(ContextPeppolFranceExtended) && len(ui.PaymentMeans) > 0 {
		if pm := ui.PaymentMeans[0].PaymentMandate; pm != nil && pm.PayerParty != nil {
			payment.Payer = goblParty(pm.PayerParty, o)
		}
	}

	if ui.PaymentTerms != nil {
		payment.Terms = &pay.Terms{
			Notes: CleanString(ui.PaymentTerms.Note),
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
		d, err := ParseDate(dueDate)
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
		payment.Instructions = GoblInvoiceInstructions(out, &ui.PaymentMeans[0])
	}

	// We do not currently map this as Peppol and EN16931 do not use it.
	/*
		if len(in.PrepaidPayment) > 0 {
			payment.Advances = make([]*pay.Record, 0, len(in.PrepaidPayment))
			for _, p := range in.PrepaidPayment {
				amount, err := num.AmountFromString(NormalizeNumericString(p.PaidAmount.Value))
				if err != nil {
					return err
				}
				advance := &pay.Record{
					Amount: amount,
				}
				if p.ReceivedDate != nil {
					d, err := ParseDate(*p.ReceivedDate)
					if err != nil {
						return err
					}
					advance.Date = &d
				}
				payment.Advances = append(payment.Advances, advance)
			}
			}
	*/

	if ui.LegalMonetaryTotal.PrepaidAmount != nil {
		totalPrepaid, err := num.AmountFromString(NormalizeNumericString(ui.LegalMonetaryTotal.PrepaidAmount.Value))
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

// GoblPaymentAdvances reconstructs bill.PaymentDetails.Advances from either
// each cac:PrepaidPayment entry, or — when none are present — a single
// advance recovered from the document's LegalMonetaryTotal/PrepaidAmount.
func GoblPaymentAdvances(payment *bill.PaymentDetails, prepaidPayments []PrepaidPayment, prepaidAmount *Amount) error {
	switch {
	case len(prepaidPayments) > 0:
		payment.Advances = make([]*pay.Record, 0, len(prepaidPayments))
		for _, p := range prepaidPayments {
			if p.PaidAmount == nil {
				continue
			}
			amount, err := num.AmountFromString(NormalizeNumericString(p.PaidAmount.Value))
			if err != nil {
				return err
			}
			advance := &pay.Record{Amount: amount}
			if p.ReceivedDate != nil {
				d, err := ParseDate(*p.ReceivedDate)
				if err != nil {
					return err
				}
				advance.Date = &d
			}
			if p.InstructionID != nil {
				advance.Ref = *p.InstructionID
			}
			payment.Advances = append(payment.Advances, advance)
		}
	case prepaidAmount != nil:
		totalPrepaid, err := num.AmountFromString(NormalizeNumericString(prepaidAmount.Value))
		if err != nil {
			return err
		}
		payment.Advances = append(payment.Advances, &pay.Record{
			Amount:      totalPrepaid,
			Description: "Prepaid Amount",
		})
	}
	return nil
}

// GoblInvoiceInstructions builds a GOBL pay.Instructions from a UBL PaymentMeans.
func GoblInvoiceInstructions(out *bill.Invoice, paymentMeans *PaymentMeans) *pay.Instructions {
	instructions := &pay.Instructions{
		Key: GoblPaymentMeansCode(paymentMeans.PaymentMeansCode.Value),
		Ext: tax.ExtensionsOf(cbc.CodeMap{
			untdid.ExtKeyPaymentMeans: cbc.Code(paymentMeans.PaymentMeansCode.Value),
		}),
	}

	if paymentMeans.PaymentMeansCode.Name != nil {
		instructions.Detail = CleanString(*paymentMeans.PaymentMeansCode.Name)
	}

	if paymentMeans.PaymentID != nil {
		instructions.Ref = cbc.Code(*paymentMeans.PaymentID)
	}

	if paymentMeans.PayeeFinancialAccount != nil {
		instructions.CreditTransfer = GoblCreditTransfer(paymentMeans)
	}
	// A mandate holding only the extended-profile PayerParty is not a
	// direct-debit arrangement.
	if pm := paymentMeans.PaymentMandate; pm != nil && (pm.ID != nil || pm.PayerFinancialAccount != nil) {
		instructions.DirectDebit = GoblInvoiceDirectDebit(out, paymentMeans)
	}
	if paymentMeans.CardAccount != nil {
		instructions.Card = GoblCard(paymentMeans)
	}

	return instructions
}

// GoblCreditTransfer builds a GOBL CreditTransfer from a UBL PaymentMeans' PayeeFinancialAccount.
func GoblCreditTransfer(paymentMeans *PaymentMeans) []*pay.CreditTransfer {
	creditTransfer := &pay.CreditTransfer{}
	account := paymentMeans.PayeeFinancialAccount

	if account.ID != nil {
		id := CleanString(*account.ID)
		if IsIBAN(id) {
			creditTransfer.IBAN = id
		} else {
			creditTransfer.Number = id
		}
	}
	if account.Name != nil {
		creditTransfer.Name = CleanString(*account.Name)
	}
	if account.FinancialInstitutionBranch != nil && account.FinancialInstitutionBranch.ID != nil {
		creditTransfer.BIC = CleanString(*account.FinancialInstitutionBranch.ID)
	}

	return []*pay.CreditTransfer{creditTransfer}
}

// IsIBAN checks if a string looks like an IBAN
// Returns true if the string starts with 2+ letters followed by alphanumeric characters
// This covers standard IBANs (e.g., NO9386011117947 or NO93 8601 1117 947) and allows
// some flexibility for various IBAN-like formats that may appear in UBL documents
func IsIBAN(s string) bool {
	s = strings.ToUpper(strings.TrimSpace(s))
	return ibanRegex.MatchString(s)
}

// GoblInvoiceDirectDebit builds a GOBL DirectDebit from a UBL PaymentMeans' PaymentMandate.
func GoblInvoiceDirectDebit(out *bill.Invoice, paymentMeans *PaymentMeans) *pay.DirectDebit {
	directDebit := &pay.DirectDebit{}

	if paymentMeans.PaymentMandate.ID != nil {
		directDebit.Ref = paymentMeans.PaymentMandate.ID.Value
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

// GoblCard builds a GOBL Card from a UBL PaymentMeans' CardAccount.
func GoblCard(paymentMeans *PaymentMeans) *pay.Card {
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
	return card
}

// GoblPaymentMeansCode maps UBL payment means to GOBL equivalent.
func GoblPaymentMeansCode(code string) cbc.Key {
	if val, ok := paymentMeansMap[code]; ok {
		return val
	}
	return pay.MeansKeyAny
}
