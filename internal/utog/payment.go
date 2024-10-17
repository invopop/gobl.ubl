package ubl

import (
	"strings"

	"github.com/invopop/gobl.ubl/structs"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/pay"
)

// ParseUtoGPayment parses the UBL XML information for a Payment object
func ParseUtoGPayment(doc *structs.Invoice) *bill.Payment {
	payment := &bill.Payment{}

	if doc.PayeeParty != nil {
		payment.Payee = ParseUtoGParty(doc.PayeeParty)
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
		payment.Instructions = parsePaymentMeans(doc.PaymentMeans[0])
	}

	if doc.PrepaidPayment != nil {
		advance := &pay.Advance{
			Amount: num.MakeAmount(doc.PrepaidPayment.PaidAmount, 2),
		}
		if doc.PrepaidPayment.PaidDate != "" {
			advancePaymentDate := ParseDate(doc.PrepaidPayment.PaidDate)
			advance.Date = &advancePaymentDate
		}
		payment.Advances = []*pay.Advance{advance}
	}

	return payment
}

func parsePaymentMeans(paymentMeans *structs.PaymentMeans) *pay.Instructions {
	instructions := &pay.Instructions{
		Key: PaymentMeansTypeCodeParse(paymentMeans.PaymentMeansCode),
	}

	if paymentMeans.PaymentID != "" {
		instructions.Detail = paymentMeans.PaymentID
	}

	if paymentMeans.PayeeFinancialAccount != nil {
		account := paymentMeans.PayeeFinancialAccount
		if account.ID != "" {
			instructions.CreditTransfer = []*pay.CreditTransfer{
				{
					IBAN: account.ID,
				},
			}
		}
		if account.Name != "" {
			if len(instructions.CreditTransfer) > 0 {
				instructions.CreditTransfer[0].Name = account.Name
			}
		}
		if paymentMeans.PayeeFinancialAccount.FinancialInstitutionBranch != nil &&
			paymentMeans.PayeeFinancialAccount.FinancialInstitutionBranch.ID != "" {
			if len(instructions.CreditTransfer) > 0 {
				instructions.CreditTransfer[0].BIC = paymentMeans.PayeeFinancialAccount.FinancialInstitutionBranch.ID
			}
		}
	}

	return instructions
}
