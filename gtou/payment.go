package gtou

import "github.com/invopop/gobl/bill"

func (c *Conversor) newPayment(payment *bill.Payment) error {
	if payment.Instructions != nil {
		c.doc.PaymentMeans = []PaymentMeans{
			{
				PaymentMeansCode: IDType{Value: string(payment.Instructions.Key)},
				PaymentID:        payment.Instructions.Ref,
			},
		}

		if payment.Instructions.CreditTransfer != nil {
			c.doc.PaymentMeans[0].PayeeFinancialAccount = &FinancialAccount{
				ID:   payment.Instructions.CreditTransfer[0].IBAN,
				Name: payment.Instructions.CreditTransfer[0].Name,
				FinancialInstitutionBranch: &Branch{
					ID: payment.Instructions.CreditTransfer[0].BIC,
				},
			}
		}
		if payment.Instructions.DirectDebit != nil {
			c.doc.PaymentMeans[0].PaymentMandate = &PaymentMandate{
				ID: IDType{Value: payment.Instructions.DirectDebit.Ref},
				PayerFinancialAccount: &FinancialAccount{
					ID: payment.Instructions.DirectDebit.Account,
				},
			}
		}
		if payment.Instructions.Card != nil {
			c.doc.PaymentMeans[0].CardAccount = &CardAccount{
				PrimaryAccountNumberID: payment.Instructions.Card.Last4,
				HolderName:             payment.Instructions.Card.Holder,
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
					term.PaymentDueDate = formatDate(*dueDate.Date)
				}
				if dueDate.Percent != nil {
					term.PaymentPercent = dueDate.Percent.String()
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
