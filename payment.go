package ubl

import (
	"errors"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/validation"
)

// PaymentMeans represents the means of payment
type PaymentMeans struct {
	PaymentMeansCode      IDType            `xml:"cbc:PaymentMeansCode"`
	PaymentDueDate        *string           `xml:"cbc:PaymentDueDate"`
	PaymentChannelCode    *IDType           `xml:"cbc:PaymentChannelCode,omitempty"`
	InstructionID         *string           `xml:"cbc:InstructionID"`
	InstructionNote       []string          `xml:"cbc:InstructionNote,omitempty"`
	PaymentID             *string           `xml:"cbc:PaymentID"`
	CardAccount           *CardAccount      `xml:"cac:CardAccount"`
	PayerFinancialAccount *FinancialAccount `xml:"cac:PayerFinancialAccount"`
	PayeeFinancialAccount *FinancialAccount `xml:"cac:PayeeFinancialAccount"`
	CreditAccount         *CreditAccount    `xml:"cac:CreditAccount"`
	PaymentMandate        *PaymentMandate   `xml:"cac:PaymentMandate"`
}

// PaymentMandate represents a payment mandate
type PaymentMandate struct {
	ID                    *IDType           `xml:"cbc:ID"`
	PayerParty            *Party            `xml:"cac:PayerParty"`
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

// FinancialInstitution identifies a financial institution by its BIC in cbc:ID.
type FinancialInstitution struct {
	ID *string `xml:"cbc:ID"`
}

// CreditAccount represents a credit account (e.g. the Danish Giro/FIK account)
type CreditAccount struct {
	AccountID string `xml:"cbc:AccountID"`
}

// PaymentTerms represents the terms of payment
type PaymentTerms struct {
	Note   string  `xml:"cbc:Note,omitempty"`
	Amount *Amount `xml:"cbc:Amount,omitempty"`
}

// PrepaidPayment represents a prepaid payment
type PrepaidPayment struct {
	ID            string  `xml:"cbc:ID"`
	PaidAmount    *Amount `xml:"cbc:PaidAmount"`
	ReceivedDate  *string `xml:"cbc:ReceivedDate"`
	InstructionID *string `xml:"cbc:InstructionID"`
}

const sepaSchemeID = "SEPA"

// AddPayment populates PaymentMeans, PayeeParty, PaymentTerms, and PrepaidPayment from the GOBL invoice's payment details.
func (ui *Invoice) AddPayment(inv *bill.Invoice, ctx Context) error {
	if inv == nil || inv.Payment == nil {
		return nil
	}
	pymt := inv.Payment

	if pymt.Instructions != nil {
		if err := ui.addPaymentInstructions(inv, ctx); err != nil {
			return err
		}
	}

	if pymt.Terms != nil {
		ui.AddPaymentTerms(pymt)
	}

	if pymt.Payee != nil {
		ui.PayeeParty = NewPayeeParty(pymt.Payee)
	}

	// The payer (EXT-FR-FE-BG-02) is only defined in the French extended
	// profile, which maps it to the PaymentMandate's PayerParty.
	if pymt.Payer != nil && ctx.Is(ContextPeppolFranceExtended) {
		if len(ui.PaymentMeans) == 0 {
			// PaymentMeans requires a PaymentMeansCode, so when the invoice
			// carries no payment instructions, fall back to UNTDID 4461 code
			// "1" (instrument not defined).
			ui.PaymentMeans = []PaymentMeans{
				{PaymentMeansCode: IDType{Value: "1"}},
			}
		}
		if ui.PaymentMeans[0].PaymentMandate == nil {
			ui.PaymentMeans[0].PaymentMandate = new(PaymentMandate)
		}
		ui.PaymentMeans[0].PaymentMandate.PayerParty = NewParty(pymt.Payer, ctx)
	}

	// BT-90: Bank assigned creditor identifier
	// In UBL this lives as a SEPA PartyIdentification on the payee (or seller)
	if pymt.Instructions != nil && pymt.Instructions.DirectDebit != nil && pymt.Instructions.DirectDebit.Creditor != "" {
		sepaID := sepaSchemeID
		id := Identification{
			ID: &IDType{
				Value:    pymt.Instructions.DirectDebit.Creditor,
				SchemeID: &sepaID,
			},
		}
		if ui.PayeeParty != nil {
			ui.PayeeParty.PartyIdentification = append(ui.PayeeParty.PartyIdentification, id)
		} else {
			ui.AccountingSupplierParty.Party.PartyIdentification = append(ui.AccountingSupplierParty.Party.PartyIdentification, id)
		}
	}

	return nil
}

func (ui *Invoice) addPaymentInstructions(inv *bill.Invoice, ctx Context) error {
	instr := inv.Payment.Instructions
	if instr.Ext.IsZero() || instr.Ext.Get(untdid.ExtKeyPaymentMeans).String() == "" {
		return validation.Errors{
			"instructions": validation.Errors{
				extFieldKey: validation.Errors{
					untdid.ExtKeyPaymentMeans.String(): errors.New("required"),
				},
			},
		}
	}
	ui.PaymentMeans = []PaymentMeans{
		{
			PaymentMeansCode: IDType{Value: instr.Ext.Get(untdid.ExtKeyPaymentMeans).String()},
		},
	}
	if ref := instr.Ref.String(); ref != "" {
		ui.PaymentMeans[0].PaymentID = &ref
	}
	if instr.Detail != "" {
		ui.PaymentMeans[0].PaymentMeansCode.Name = &instr.Detail
	}
	if len(instr.CreditTransfer) > 0 {
		ui.PaymentMeans[0].PayeeFinancialAccount = newCreditTransferAccount(instr.CreditTransfer[0])
	}
	if instr.DirectDebit != nil {
		// Skip the mandate without a reference; an empty <cbc:ID/> is rejected by Peppol.
		if instr.DirectDebit.Ref != "" {
			ui.PaymentMeans[0].PaymentMandate = &PaymentMandate{
				ID: &IDType{Value: instr.DirectDebit.Ref},
			}
		}
		if instr.DirectDebit.Account != "" {
			ui.PaymentMeans[0].PayerFinancialAccount = &FinancialAccount{
				ID: &instr.DirectDebit.Account,
			}
		}
	}
	if instr.Card != nil {
		ui.PaymentMeans[0].CardAccount = &CardAccount{
			PrimaryAccountNumberID: &instr.Card.Last4,
		}
		if instr.Card.Holder != "" {
			ui.PaymentMeans[0].CardAccount.HolderName = &instr.Card.Holder
		}
	}
	if ui.CreditNoteTypeCode != nil && inv.Payment.Terms != nil && len(inv.Payment.Terms.DueDates) > 0 {
		formattedDate := FormatDate(*inv.Payment.Terms.DueDates[0].Date)
		ui.PaymentMeans[0].PaymentDueDate = &formattedDate
	}
	// BR-KSA-17: Debit and credit note must contain the
	// reason for this invoice type issuing.
	if inv.Preceding != nil && ctx.Is(ContextZATCA) {
		for _, ref := range inv.Preceding {
			ui.PaymentMeans[0].InstructionNote = append(ui.PaymentMeans[0].InstructionNote, ref.Reason)
		}
	}
	return nil
}

func newCreditTransferAccount(ct *pay.CreditTransfer) *FinancialAccount {
	pfa := new(FinancialAccount)
	if ct.IBAN != "" {
		pfa.ID = &ct.IBAN
	} else if ct.Number != "" {
		pfa.ID = &ct.Number
	}
	if ct.Name != "" {
		pfa.Name = &ct.Name
	}
	if ct.BIC != "" {
		pfa.FinancialInstitutionBranch = &Branch{ID: &ct.BIC}
	}
	return pfa
}

// AddPaymentTerms sets PaymentTerms from the GOBL payment details' notes and due amount.
func (ui *Invoice) AddPaymentTerms(pymt *bill.PaymentDetails) {
	if pymt.Terms.Notes != "" {
		ui.PaymentTerms = &PaymentTerms{
			Note: pymt.Terms.Notes,
		}
	}

	// Only one due date allowed under EN 16931
	if ui.CreditNoteTypeCode == nil && len(pymt.Terms.DueDates) > 0 {
		ui.DueDate = FormatDate(*pymt.Terms.DueDates[0].Date)
	}
}
