package utog

import (
	"github.com/invopop/gobl.ubl/document"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
)

func (c *Converter) getInstructions(paymentMeans *document.PaymentMeans) *pay.Instructions {
	instructions := &pay.Instructions{
		Key: paymentMeansCode(paymentMeans.PaymentMeansCode.Value),
		Ext: tax.Extensions{
			untdid.ExtKeyPaymentMeans: tax.ExtValue(paymentMeans.PaymentMeansCode.Value),
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
		instructions.CreditTransfer = c.getCreditTransfer(paymentMeans)
	case pay.MeansKeyDirectDebit, pay.MeansKeyDirectDebit.With(pay.MeansKeySEPA):
		instructions.DirectDebit = c.getDirectDebit(paymentMeans)
	case pay.MeansKeyCard:
		instructions.Card = c.getCard(paymentMeans)
	}

	return instructions
}

func (c *Converter) getCreditTransfer(paymentMeans *document.PaymentMeans) []*pay.CreditTransfer {
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

func (c *Converter) getDirectDebit(paymentMeans *document.PaymentMeans) *pay.DirectDebit {
	directDebit := &pay.DirectDebit{}

	if paymentMeans.PaymentMandate != nil {
		directDebit.Ref = paymentMeans.PaymentMandate.ID.Value
		if paymentMeans.PaymentMandate.PayerFinancialAccount != nil && paymentMeans.PaymentMandate.PayerFinancialAccount.ID != nil {
			directDebit.Account = *paymentMeans.PaymentMandate.PayerFinancialAccount.ID
		}
	}
	seller := c.GetInvoice().Supplier
	if seller != nil {
		for _, id := range seller.Identities {
			if id.Label == "SEPA" {
				directDebit.Creditor = id.Code.String()
				break
			}
		}
	}
	payment := c.GetInvoice().Payment
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

func (c *Converter) getCard(paymentMeans *document.PaymentMeans) *pay.Card {
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
