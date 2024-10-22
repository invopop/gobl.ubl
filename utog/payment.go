package utog

import (
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/pay"
)

func (c *Conversor) getPayment(doc *Document) error {
	payment := &bill.Payment{}

	if doc.PayeeParty != nil {
		payment.Payee = c.getParty(doc.PayeeParty)
	}

	if len(doc.PaymentTerms) > 0 {
		payment.Terms = &pay.Terms{}
		var notes []string
		for _, term := range doc.PaymentTerms {
			if term.Note != "" {
				notes = append(notes, term.Note)
			}
		}
		if len(notes) > 0 {
			payment.Terms.Notes = strings.Join(notes, " ")
		}
	}

	if len(doc.PaymentMeans) > 0 {
		payment.Instructions = parsePaymentMeans(&doc.PaymentMeans[0])
	}

	if len(doc.PrepaidPayment) > 0 {
		payment.Advances = make([]*pay.Advance, 0, len(doc.PrepaidPayment))
		for _, p := range doc.PrepaidPayment {
			amount, err := num.AmountFromString(p.PaidAmount.Value)
			if err != nil {
				return err
			}
			advance := &pay.Advance{
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
	c.inv.Payment = payment
	return nil
}

func parsePaymentMeans(paymentMeans *PaymentMeans) *pay.Instructions {
	instructions := &pay.Instructions{
		Key: PaymentMeansTypeCodeParse(paymentMeans.PaymentMeansCode),
	}

	if paymentMeans.PaymentID != nil {
		instructions.Detail = *paymentMeans.PaymentID
	}

	if paymentMeans.PayeeFinancialAccount != nil {
		account := paymentMeans.PayeeFinancialAccount
		if account.ID != nil {
			instructions.CreditTransfer = []*pay.CreditTransfer{
				{
					IBAN: *account.ID,
				},
			}
		}
		if account.Name != nil {
			if len(instructions.CreditTransfer) > 0 {
				instructions.CreditTransfer[0].Name = *account.Name
			}
		}
		if paymentMeans.PayeeFinancialAccount != nil {
			if paymentMeans.PayeeFinancialAccount.FinancialInstitutionBranch.ID != nil {
				if len(instructions.CreditTransfer) > 0 {
					instructions.CreditTransfer[0].BIC = *paymentMeans.PayeeFinancialAccount.FinancialInstitutionBranch.ID
				}
			}
		}
	}

	return instructions
}
