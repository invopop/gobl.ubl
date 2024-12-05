package gtou

import (
	"errors"

	"github.com/invopop/gobl.ubl/document"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/validation"
)

func (c *Converter) newPayment(pymt *bill.Payment) error {
	if pymt == nil {
		return nil
	}
	if pymt.Instructions != nil {
		ref := pymt.Instructions.Ref.String()
		if pymt.Instructions.Ext == nil || pymt.Instructions.Ext.Get(untdid.ExtKeyPaymentMeans).String() == "" {
			return validation.Errors{
				"instructions": validation.Errors{
					"ext": validation.Errors{
						untdid.ExtKeyPaymentMeans.String(): errors.New("required"),
					},
				},
			}
		}
		c.doc.PaymentMeans = []document.PaymentMeans{
			{
				PaymentMeansCode: document.IDType{Value: pymt.Instructions.Ext.Get(untdid.ExtKeyPaymentMeans).String()},
				PaymentID:        &ref,
			},
		}

		if pymt.Instructions.CreditTransfer != nil {
			c.doc.PaymentMeans[0].PayeeFinancialAccount = &document.FinancialAccount{
				ID: &pymt.Instructions.CreditTransfer[0].IBAN,
			}
			if pymt.Instructions.CreditTransfer[0].Name != "" {
				c.doc.PaymentMeans[0].PayeeFinancialAccount.Name = &pymt.Instructions.CreditTransfer[0].Name
			}
			if pymt.Instructions.CreditTransfer[0].BIC != "" {
				c.doc.PaymentMeans[0].PayeeFinancialAccount.FinancialInstitutionBranch = &document.Branch{
					ID: &pymt.Instructions.CreditTransfer[0].BIC,
				}
			}
		}
		if pymt.Instructions.DirectDebit != nil {
			c.doc.PaymentMeans[0].PaymentMandate = &document.PaymentMandate{
				ID: document.IDType{Value: pymt.Instructions.DirectDebit.Ref},
			}
			if pymt.Instructions.DirectDebit.Account != "" {
				c.doc.PaymentMeans[0].PayerFinancialAccount = &document.FinancialAccount{
					ID: &pymt.Instructions.DirectDebit.Account,
				}
			}
		}
		if pymt.Instructions.Card != nil {
			c.doc.PaymentMeans[0].CardAccount = &document.CardAccount{
				PrimaryAccountNumberID: &pymt.Instructions.Card.Last4,
			}
			if pymt.Instructions.Card.Holder != "" {
				c.doc.PaymentMeans[0].CardAccount.HolderName = &pymt.Instructions.Card.Holder
			}
		}
	}

	if pymt.Terms != nil {
		c.doc.PaymentTerms = make([]document.PaymentTerms, 0)
		if len(pymt.Terms.DueDates) > 1 {
			for _, dueDate := range pymt.Terms.DueDates {
				currency := dueDate.Currency.String()
				term := document.PaymentTerms{
					Amount: &document.Amount{Value: dueDate.Amount.String(), CurrencyID: &currency},
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
		} else if len(pymt.Terms.DueDates) == 1 {
			c.doc.DueDate = formatDate(*pymt.Terms.DueDates[0].Date)
		} else {
			c.doc.PaymentTerms = append(c.doc.PaymentTerms, document.PaymentTerms{
				Note: []string{pymt.Terms.Detail},
			})
		}
	}

	if pymt.Payee != nil {
		p := c.newParty(pymt.Payee)
		c.doc.PayeeParty = &p
	}
	return nil
}
