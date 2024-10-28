package gtou

import "github.com/invopop/gobl/bill"

func (c *Conversor) newPayment(payment *bill.Payment) error {
	if payment == nil {
		return nil
	}
	if payment.Instructions != nil {
		c.doc.PaymentMeans = []PaymentMeans{
			{
				PaymentMeansCode: IDType{Value: findPaymentKey(payment.Instructions.Key)},
				PaymentID:        &payment.Instructions.Ref,
			},
		}

		if payment.Instructions.CreditTransfer != nil {
			c.doc.PaymentMeans[0].PayeeFinancialAccount = &FinancialAccount{
				ID: &payment.Instructions.CreditTransfer[0].IBAN,
			}
			if payment.Instructions.CreditTransfer[0].Name != "" {
				c.doc.PaymentMeans[0].PayeeFinancialAccount.Name = &payment.Instructions.CreditTransfer[0].Name
			}
			if payment.Instructions.CreditTransfer[0].BIC != "" {
				c.doc.PaymentMeans[0].PayeeFinancialAccount.FinancialInstitutionBranch = &Branch{
					ID: &payment.Instructions.CreditTransfer[0].BIC,
				}
			}
		}
		if payment.Instructions.DirectDebit != nil {
			c.doc.PaymentMeans[0].PaymentMandate = &PaymentMandate{
				ID: IDType{Value: payment.Instructions.DirectDebit.Ref},
			}
			if payment.Instructions.DirectDebit.Account != "" {
				c.doc.PaymentMeans[0].PayerFinancialAccount = &FinancialAccount{
					ID: &payment.Instructions.DirectDebit.Account,
				}
			}
		}
		if payment.Instructions.Card != nil {
			c.doc.PaymentMeans[0].CardAccount = &CardAccount{
				PrimaryAccountNumberID: &payment.Instructions.Card.Last4,
			}
			if payment.Instructions.Card.Holder != "" {
				c.doc.PaymentMeans[0].CardAccount.HolderName = &payment.Instructions.Card.Holder
			}
		}
	}

	if payment.Terms != nil {
		c.doc.PaymentTerms = make([]PaymentTerms, 0)
		if len(payment.Terms.DueDates) > 1 {
			for _, dueDate := range payment.Terms.DueDates {
				currency := dueDate.Currency.String()
				term := PaymentTerms{
					Amount: &Amount{Value: dueDate.Amount.String(), CurrencyID: &currency},
				}
				if dueDate.Date != nil {
					d := formatDate(*dueDate.Date)
					term.PaymentDueDate = &d
				}
				if dueDate.Percent != nil {
					p := dueDate.Percent.String()
					term.PaymentPercent = &p
				}
				if dueDate.Notes != "" {
					term.Note = []string{dueDate.Notes}
				}
				c.doc.PaymentTerms = append(c.doc.PaymentTerms, term)
			}
		} else if len(payment.Terms.DueDates) == 1 {
			c.doc.DueDate = formatDate(*payment.Terms.DueDates[0].Date)
		} else {
			c.doc.PaymentTerms = append(c.doc.PaymentTerms, PaymentTerms{
				Note: []string{payment.Terms.Detail},
			})
		}
	}

	if payment.Payee != nil {
		payee := c.newParty(payment.Payee)
		c.doc.PayeeParty = &payee
	}
	return nil
}
