package ubl

import (
	"errors"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/validation"
)

// PaymentMeans represents the means of payment
type PaymentMeans struct {
	PaymentMeansCode      IDType            `xml:"cbc:PaymentMeansCode"`
	PaymentDueDate        *string           `xml:"cbc:PaymentDueDate,omitempty"`
	PaymentChannelCode    *IDType           `xml:"cbc:PaymentChannelCode,omitempty"`
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
	ID                   *string               `xml:"cbc:ID"`
	Name                 *string               `xml:"cbc:Name"`
	FinancialInstitution *FinancialInstitution `xml:"cac:FinancialInstitution"`
}

// FinancialInstitution represents a financial institution.
type FinancialInstitution struct {
	ID *string `xml:"cbc:ID"`
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

func (ui *Invoice) addPayment(inv *bill.Invoice, o *options) error {
	if inv == nil || inv.Payment == nil {
		return nil
	}
	pymt := inv.Payment

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
		paymentMeansCode := pymt.Instructions.Ext.Get(untdid.ExtKeyPaymentMeans).String()
		if o != nil && o.context.IsOIOUBL() && paymentMeansCode == "30" {
			// OIOUBL restricts allowed payment means and expects code 31 for IBAN transfers.
			paymentMeansCode = "31"
		}

		ui.PaymentMeans = []PaymentMeans{
			{
				PaymentMeansCode: IDType{Value: paymentMeansCode},
			},
		}

		if pymt.Instructions.Meta != nil {
			if channel, ok := pymt.Instructions.Meta[cbc.Key("payment-channel")]; ok && channel != "" {
				ui.PaymentMeans[0].PaymentChannelCode = &IDType{Value: channel}
			}
		}

		if ref != "" {
			ui.PaymentMeans[0].PaymentID = &ref
		}

		if pymt.Instructions.CreditTransfer != nil {
			pfa := new(FinancialAccount)

			if pymt.Instructions.CreditTransfer[0].IBAN != "" {
				pfa.ID = &pymt.Instructions.CreditTransfer[0].IBAN
			} else if pymt.Instructions.CreditTransfer[0].Number != "" {
				pfa.ID = &pymt.Instructions.CreditTransfer[0].Number
			}
			if pymt.Instructions.CreditTransfer[0].Name != "" {
				pfa.Name = &pymt.Instructions.CreditTransfer[0].Name
			}
			if pymt.Instructions.CreditTransfer[0].BIC != "" {
				branch := &Branch{ID: &pymt.Instructions.CreditTransfer[0].BIC}
				if o != nil && o.context.IsOIOUBL() && paymentMeansCode == "31" {
					branch.FinancialInstitution = &FinancialInstitution{
						ID: &pymt.Instructions.CreditTransfer[0].BIC,
					}
				}
				pfa.FinancialInstitutionBranch = branch
			}
			if o != nil && o.context.IsOIOUBL() && paymentMeansCode == "31" && ui.PaymentMeans[0].PaymentChannelCode == nil {
				ui.PaymentMeans[0].PaymentChannelCode = &IDType{Value: "IBAN"}
			}

			ui.PaymentMeans[0].PayeeFinancialAccount = pfa
		}
		if pymt.Instructions.DirectDebit != nil {
			ui.PaymentMeans[0].PaymentMandate = &PaymentMandate{
				ID: IDType{Value: pymt.Instructions.DirectDebit.Ref},
			}
			if pymt.Instructions.DirectDebit.Account != "" {
				ui.PaymentMeans[0].PayerFinancialAccount = &FinancialAccount{
					ID: &pymt.Instructions.DirectDebit.Account,
				}
			}
		}
		if pymt.Instructions.Card != nil {
			ui.PaymentMeans[0].CardAccount = &CardAccount{
				PrimaryAccountNumberID: &pymt.Instructions.Card.Last4,
			}
			if pymt.Instructions.Card.Holder != "" {
				ui.PaymentMeans[0].CardAccount.HolderName = &pymt.Instructions.Card.Holder
			}
		}
	}

	if pymt.Terms != nil {
		ui.PaymentTerms = make([]PaymentTerms, 0)
		if (len(pymt.Terms.DueDates) > 1) || (ui.CreditNoteTypeCode != "" && len(pymt.Terms.DueDates) > 0) {
			for _, dueDate := range pymt.Terms.DueDates {
				currency := dueDate.Currency.String()
				if currency == "" {
					currency = inv.Currency.String()
				}
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
				ui.PaymentTerms = append(ui.PaymentTerms, term)
			}
			// credit notes should not have due dates by schema
		} else if len(pymt.Terms.DueDates) == 1 && ui.CreditNoteTypeCode == "" {
			if pymt.Terms.DueDates[0].Date != nil {
				ui.DueDate = formatDate(*pymt.Terms.DueDates[0].Date)
			}
		} else {
			ui.PaymentTerms = append(ui.PaymentTerms, PaymentTerms{
				Note: []string{pymt.Terms.Notes},
			})
		}
	}

	if pymt.Payee != nil {
		ui.PayeeParty = newPayeeParty(pymt.Payee)
	}

	return nil
}
