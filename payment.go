package ubl

import (
	"errors"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/validation"
)

// PaymentMeans represents the means of payment
type PaymentMeans struct {
	PaymentMeansCode      IDType            `xml:"cbc:PaymentMeansCode"`
	InstructionID         *string           `xml:"cbc:InstructionID"`
	InstructionNote       []string          `xml:"cbc:InstructionNote"`
	PaymentID             *string           `xml:"cbc:PaymentID"`
	CardAccount           *CardAccount      `xml:"cac:CardAccount"`
	PayerFinancialAccount *FinancialAccount `xml:"cac:PayerFinancialAccount"`
	PayeeFinancialAccount *FinancialAccount `xml:"cac:PayeeFinancialAccount"`
	PaymentMandate        *PaymentMandate   `xml:"cac:PaymentMandate"`
}

// PaymentMandate represents a payment mandate
type PaymentMandate struct {
	ID                    IDType            `xml:"cbc:ID"`
	PayerFinancialAccount *FinancialAccount `xml:"cac:PayerFinancialAccount"`
}

// CardAccount represents a card account
type CardAccount struct {
	PrimaryAccountNumberID *string `xml:"cbc:PrimaryAccountNumberID"`
	NetworkID              *string `xml:"cbc:NetworkID"`
	HolderName             *string `xml:"cbc:HolderName"`
}

// FinancialAccount represents a financial account
type FinancialAccount struct {
	ID                         *string `xml:"cbc:ID"`
	Name                       *string `xml:"cbc:Name"`
	FinancialInstitutionBranch *Branch `xml:"cac:FinancialInstitutionBranch"`
	AccountTypeCode            *string `xml:"cbc:AccountTypeCode"`
}

// Branch represents a branch of a financial institution
type Branch struct {
	ID   *string `xml:"cbc:ID"`
	Name *string `xml:"cbc:Name"`
}

// PaymentTerms represents the terms of payment
type PaymentTerms struct {
	Note           []string `xml:"cbc:Note"`
	Amount         *Amount  `xml:"cbc:Amount"`
	PaymentPercent *string  `xml:"cbc:PaymentPercent"`
	PaymentDueDate *string  `xml:"cbc:PaymentDueDate"`
}

// PrepaidPayment represents a prepaid payment
type PrepaidPayment struct {
	ID            string  `xml:"cbc:ID"`
	PaidAmount    *Amount `xml:"cbc:PaidAmount"`
	ReceivedDate  *string `xml:"cbc:ReceivedDate"`
	InstructionID *string `xml:"cbc:InstructionID"`
}

func (out *Invoice) addPayment(pymt *bill.PaymentDetails) error {
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
		out.PaymentMeans = []PaymentMeans{
			{
				PaymentMeansCode: IDType{Value: pymt.Instructions.Ext.Get(untdid.ExtKeyPaymentMeans).String()},
				PaymentID:        &ref,
			},
		}

		if pymt.Instructions.CreditTransfer != nil {
			out.PaymentMeans[0].PayeeFinancialAccount = &FinancialAccount{
				ID: &pymt.Instructions.CreditTransfer[0].IBAN,
			}
			if pymt.Instructions.CreditTransfer[0].Name != "" {
				out.PaymentMeans[0].PayeeFinancialAccount.Name = &pymt.Instructions.CreditTransfer[0].Name
			}
			if pymt.Instructions.CreditTransfer[0].BIC != "" {
				out.PaymentMeans[0].PayeeFinancialAccount.FinancialInstitutionBranch = &Branch{
					ID: &pymt.Instructions.CreditTransfer[0].BIC,
				}
			}
		}
		if pymt.Instructions.DirectDebit != nil {
			out.PaymentMeans[0].PaymentMandate = &PaymentMandate{
				ID: IDType{Value: pymt.Instructions.DirectDebit.Ref},
			}
			if pymt.Instructions.DirectDebit.Account != "" {
				out.PaymentMeans[0].PayerFinancialAccount = &FinancialAccount{
					ID: &pymt.Instructions.DirectDebit.Account,
				}
			}
		}
		if pymt.Instructions.Card != nil {
			out.PaymentMeans[0].CardAccount = &CardAccount{
				PrimaryAccountNumberID: &pymt.Instructions.Card.Last4,
			}
			if pymt.Instructions.Card.Holder != "" {
				out.PaymentMeans[0].CardAccount.HolderName = &pymt.Instructions.Card.Holder
			}
		}
	}

	if pymt.Terms != nil {
		out.PaymentTerms = make([]PaymentTerms, 0)
		if len(pymt.Terms.DueDates) > 1 {
			for _, dueDate := range pymt.Terms.DueDates {
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
				out.PaymentTerms = append(out.PaymentTerms, term)
			}
		} else if len(pymt.Terms.DueDates) == 1 {
			out.DueDate = formatDate(*pymt.Terms.DueDates[0].Date)
		} else {
			out.PaymentTerms = append(out.PaymentTerms, PaymentTerms{
				Note: []string{pymt.Terms.Detail},
			})
		}
	}

	if pymt.Payee != nil {
		out.PayeeParty = newParty(pymt.Payee)
	}

	return nil
}
