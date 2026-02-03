package ubl

import (
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
)

var paymentMeansMap = map[string]cbc.Key{
	"10": pay.MeansKeyCash,
	"20": pay.MeansKeyCheque,
	"30": pay.MeansKeyCreditTransfer,
	"42": pay.MeansKeyDebitTransfer,
	"48": pay.MeansKeyCard,
	"49": pay.MeansKeyDirectDebit,
	"58": pay.MeansKeyCreditTransfer.With(pay.MeansKeySEPA),
	"59": pay.MeansKeyDirectDebit.With(pay.MeansKeySEPA),
}

func (ui *Invoice) goblAddPayment(out *bill.Invoice) error {
	payment := &bill.PaymentDetails{}

	if ui.PayeeParty != nil {
		payment.Payee = goblParty(ui.PayeeParty)
	}

	if len(ui.PaymentTerms) > 0 {
		payment.Terms = &pay.Terms{}
		note := make([]string, 0)
		for _, term := range ui.PaymentTerms {
			note = append(note, term.Note...)
			if term.Amount != nil {
				amount, err := num.AmountFromString(normalizeNumericString(term.Amount.Value))
				if err != nil {
					return err
				}
				payment.Terms.DueDates = append(payment.Terms.DueDates, &pay.DueDate{
					Amount: amount,
				})
			}
		}
		if len(note) > 0 {
			payment.Terms.Notes = strings.Join(note, " ")
		}
	}

	if ui.DueDate != "" {
		d, err := parseDate(ui.DueDate)
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
			payment.Advances = make([]*pay.Advance, 0, len(in.PrepaidPayment))
			for _, p := range in.PrepaidPayment {
				amount, err := num.AmountFromString(normalizeNumericString(p.PaidAmount.Value))
				if err != nil {
					return err
				}
				advance := &pay.Advance{
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

	if ui.LegalMonetaryTotal.PrepaidAmount != nil {
		totalPrepaid, err := num.AmountFromString(normalizeNumericString(ui.LegalMonetaryTotal.PrepaidAmount.Value))
		if err != nil {
			return err
		}

		advance := &pay.Advance{
			Amount:      totalPrepaid,
			Description: "Prepaid Amount",
		}
		payment.Advances = append(payment.Advances, advance)

	}

	if payment.Payee != nil || payment.Terms != nil || payment.Instructions != nil || len(payment.Advances) > 0 {
		out.Payment = payment
	}
	return nil
}

func goblInvoiceInstructions(out *bill.Invoice, paymentMeans *PaymentMeans) *pay.Instructions {
	instructions := &pay.Instructions{
		Key: goblPaymentMeansCode(paymentMeans.PaymentMeansCode.Value),
		Ext: tax.Extensions{
			untdid.ExtKeyPaymentMeans: cbc.Code(paymentMeans.PaymentMeansCode.Value),
		},
	}

	if paymentMeans.PaymentMeansCode.Name != nil {
		instructions.Detail = *paymentMeans.PaymentMeansCode.Name
	}

	if paymentMeans.PaymentID != nil {
		instructions.Ref = cbc.Code(*paymentMeans.PaymentID)
	}

	switch instructions.Key {
	case pay.MeansKeyCreditTransfer, pay.MeansKeyCreditTransfer.With(pay.MeansKeySEPA):
		instructions.CreditTransfer = goblCreditTransfer(paymentMeans)
	case pay.MeansKeyDirectDebit, pay.MeansKeyDirectDebit.With(pay.MeansKeySEPA):
		instructions.DirectDebit = goblInvoiceDirectDebit(out, paymentMeans)
	case pay.MeansKeyCard:
		instructions.Card = goblCard(paymentMeans)
	}

	return instructions
}

func goblCreditTransfer(paymentMeans *PaymentMeans) []*pay.CreditTransfer {
	creditTransfer := &pay.CreditTransfer{}

	if paymentMeans.PayeeFinancialAccount != nil {
		account := paymentMeans.PayeeFinancialAccount
		if account.ID != nil {
			creditTransfer.IBAN = *account.ID
		}
		if account.Name != nil {
			creditTransfer.Name = *account.Name
		}
		if account.FinancialInstitutionBranch != nil && account.FinancialInstitutionBranch.ID != nil {
			creditTransfer.BIC = *account.FinancialInstitutionBranch.ID
		}
	}

	return []*pay.CreditTransfer{creditTransfer}
}

func goblInvoiceDirectDebit(out *bill.Invoice, paymentMeans *PaymentMeans) *pay.DirectDebit {
	directDebit := &pay.DirectDebit{}

	if paymentMeans.PaymentMandate != nil {
		directDebit.Ref = paymentMeans.PaymentMandate.ID.Value
		if paymentMeans.PaymentMandate.PayerFinancialAccount != nil && paymentMeans.PaymentMandate.PayerFinancialAccount.ID != nil {
			directDebit.Account = *paymentMeans.PaymentMandate.PayerFinancialAccount.ID
		}
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
	if paymentMeans.CardAccount != nil {
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
