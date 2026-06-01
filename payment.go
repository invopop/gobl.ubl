package ubl

import (
	"errors"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/validation"
)

// PaymentMeans represents the means of payment
type PaymentMeans struct {
	PaymentMeansCode      IDType            `xml:"cbc:PaymentMeansCode"`
	PaymentDueDate        *string           `xml:"cbc:PaymentDueDate,omitempty"`
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

// CreditAccount carries the OIOUBL FIK creditor account (cbc:AccountID).
type CreditAccount struct {
	AccountID string `xml:"cbc:AccountID"`
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

func (ui *Invoice) addPayment(inv *bill.Invoice, ctx Context) error {
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
		ui.addPaymentTerms(pymt)
	}

	if pymt.Payee != nil {
		ui.PayeeParty = newPayeeParty(pymt.Payee)
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
	paymentMeansCode := instr.Ext.Get(untdid.ExtKeyPaymentMeans).String()
	if ctx.Is(ContextOIOUBL21) && paymentMeansCode == "30" {
		paymentMeansCode = "31"
	}
	ui.PaymentMeans = []PaymentMeans{
		{
			PaymentMeansCode: IDType{Value: paymentMeansCode},
		},
	}
	if instr.Meta != nil {
		if channel, ok := instr.Meta[cbc.Key("payment-channel")]; ok && channel != "" {
			ui.PaymentMeans[0].PaymentChannelCode = &IDType{Value: channel}
		}
	}
	if ref := instr.Ref.String(); ref != "" {
		ui.PaymentMeans[0].PaymentID = &ref
	}
	// OIOUBL Giro (50) / FIK (93): cbc:PaymentID is the dk-oioubl-payment-id
	// "kortart" (overriding instr.Ref, which is the Peppol mapping), and the
	// PaymentChannelCode is DK:GIRO / DK:FIK (F-LIB155/F-LIB277). The FIK
	// creditor account flows through the credit-transfer Number below (F-LIB305).
	if ctx.Is(ContextOIOUBL21) {
		applyOIOUBL21PaymentID(&ui.PaymentMeans[0], instr, paymentMeansCode)
	}
	if instr.Detail != "" {
		ui.PaymentMeans[0].PaymentMeansCode.Name = &instr.Detail
	}
	ui.addCreditTransferAccount(instr, ctx, paymentMeansCode)
	if instr.DirectDebit != nil {
		ui.PaymentMeans[0].PaymentMandate = &PaymentMandate{
			ID: IDType{Value: instr.DirectDebit.Ref},
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
		formattedDate := formatDate(*inv.Payment.Terms.DueDates[0].Date)
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

// applyOIOUBL21PaymentID sets the OIOUBL Giro (50) / FIK (93) cbc:PaymentID from
// the dk-oioubl-payment-id "kortart" (overriding instr.Ref, which is the Peppol
// mapping) and the PaymentChannelCode DK:GIRO / DK:FIK (F-LIB155/F-LIB277). The
// FIK creditor account flows through the credit-transfer Number (F-LIB305).
func applyOIOUBL21PaymentID(pm *PaymentMeans, instr *pay.Instructions, paymentMeansCode string) {
	if paymentMeansCode != "50" && paymentMeansCode != "93" {
		return
	}
	if kortart := instr.Ext.Get(cbc.Key("dk-oioubl-payment-id")).String(); kortart != "" {
		pm.PaymentID = &kortart
	}
	channel := "DK:GIRO"
	if paymentMeansCode == "93" {
		channel = "DK:FIK"
	}
	pm.PaymentChannelCode = &IDType{Value: channel}
}

// addCreditTransferAccount wires the credit-transfer account onto the payment
// means. For OIOUBL FIK (93) the creditor account lives in
// cac:CreditAccount/cbc:AccountID (8 chars, F-LIB305) rather than
// PayeeFinancialAccount.
func (ui *Invoice) addCreditTransferAccount(instr *pay.Instructions, ctx Context, paymentMeansCode string) {
	if len(instr.CreditTransfer) == 0 {
		return
	}
	pm := &ui.PaymentMeans[0]
	if ctx.Is(ContextOIOUBL21) && paymentMeansCode == "93" {
		pm.CreditAccount = &CreditAccount{AccountID: instr.CreditTransfer[0].Number}
		return
	}
	pm.PayeeFinancialAccount = newCreditTransferAccount(instr.CreditTransfer[0], ctx, paymentMeansCode)
	if ctx.Is(ContextOIOUBL21) && paymentMeansCode == "31" && pm.PaymentChannelCode == nil {
		pm.PaymentChannelCode = &IDType{Value: oioubl21PaymentChannelIBAN}
	}
}

func newCreditTransferAccount(ct *pay.CreditTransfer, ctx Context, paymentMeansCode string) *FinancialAccount {
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
		branch := &Branch{ID: &ct.BIC}
		if ctx.Is(ContextOIOUBL21) && paymentMeansCode == "31" {
			branch.FinancialInstitution = &FinancialInstitution{
				ID: &ct.BIC,
			}
		}
		pfa.FinancialInstitutionBranch = branch
	}
	return pfa
}

func (ui *Invoice) addPaymentTerms(pymt *bill.PaymentDetails) {
	if pymt.Terms.Notes != "" {
		ui.PaymentTerms = &PaymentTerms{
			Note: pymt.Terms.Notes,
		}
	}

	// Only one due date allowed under EN 16931
	if ui.CreditNoteTypeCode == nil && len(pymt.Terms.DueDates) > 0 && pymt.Terms.DueDates[0].Date != nil {
		ui.DueDate = formatDate(*pymt.Terms.DueDates[0].Date)
	}
}
