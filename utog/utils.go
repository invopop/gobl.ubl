package utog

import (
	"time"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
)

const (
	keyPaymentMeansSEPACreditTransfer cbc.Key = "sepa-credit-transfer"
	keyPaymentMeansSEPADirectDebit    cbc.Key = "sepa-direct-debit"
)

const (
	paymentMeansCash               = "10"
	paymentMeansCheque             = "20"
	paymentMeansCreditTransfer     = "30"
	paymentMeansBankAccount        = "42"
	paymentMeansCard               = "48"
	paymentMeansDirectDebit        = "49"
	paymentMeansStandingOrder      = "57"
	paymentMeansSEPACreditTransfer = "58"
	paymentMeansSEPADirectDebit    = "59"
	paymentMeansReport             = "97"
)

const (
	standardSalesTax  = "S"
	zeroRatedGoodsTax = "Z"
	taxExempt         = "E"
)

const (
	keyInvoiceTypeSelfBilled               cbc.Key = "self-billed"
	keyInvoiceTypePartial                  cbc.Key = "partial"
	keyInvoiceTypePartialConstruction      cbc.Key = "partial-construction"
	keyInvoiceTypePartialFinalConstruction cbc.Key = "partial-final-construction"
	keyInvoiceTypeFinalConstruction        cbc.Key = "final-construction"
)

const (
	invoiceTypeProforma                 = "325"
	invoiceTypeStandard                 = "380"
	invoiceTypeCreditNote               = "381"
	invoiceTypeDebitNote                = "383"
	invoiceTypeCorrective               = "384"
	invoiceTypeSelfBilled               = "389"
	invoiceTypePartial                  = "326"
	invoiceTypePartialConstruction      = "875"
	invoiceTypePartialFinalConstruction = "876"
	invoiceTypeFinalConstruction        = "877"
)

// Convert a date string to a cal.Date
func ParseDate(date string) (cal.Date, error) {
	t, err := time.Parse("20060102", date)
	if err != nil {
		return cal.Date{}, err
	}

	return cal.MakeDate(t.Year(), t.Month(), t.Day()), nil
}

// Map UBL rate to GOBL equivalent
func FindTaxKey(taxType string) cbc.Key {
	switch taxType {
	case standardSalesTax:
		return tax.RateStandard
	case zeroRatedGoodsTax:
		return tax.RateZero
	case taxExempt:
		return tax.RateExempt
	}
	return tax.RateStandard
}

// Map CII invoice type to GOBL equivalent
// Source https://unece.org/fileadmin/DAM/trade/untdid/d16b/tred/tred1001.htm
func TypeCodeParse(typeCode string) cbc.Key {
	switch typeCode {
	case invoiceTypeStandard:
		return bill.InvoiceTypeStandard
	case invoiceTypeCreditNote:
		return bill.InvoiceTypeCreditNote
	case invoiceTypeCorrective:
		return bill.InvoiceTypeCorrective
	case invoiceTypeSelfBilled:
		return bill.InvoiceTypeProforma
	case invoiceTypeDebitNote:
		return bill.InvoiceTypeDebitNote
	case invoiceTypePartial:
		return keyInvoiceTypePartial
	case invoiceTypePartialConstruction:
		return keyInvoiceTypePartialConstruction
	case invoiceTypePartialFinalConstruction:
		return keyInvoiceTypePartialFinalConstruction
	case invoiceTypeFinalConstruction:
		return keyInvoiceTypeFinalConstruction
	}
	return bill.InvoiceTypeOther
}

// Map UN/ECE code to GOBL equivalent
func UnitFromUNECE(unece cbc.Code) org.Unit {
	for _, def := range org.UnitDefinitions {
		if def.UNECE == unece {
			return def.Unit
		}
	}
	// If no match is found, return the original UN/ECE code as a Unit
	return org.Unit(unece)
}

// Map UBL payment means to GOBL equivalent
func PaymentMeansTypeCodeParse(typeCode string) cbc.Key {
	switch typeCode {
	case paymentMeansCash:
		return pay.MeansKeyCash
	case paymentMeansCheque:
		return pay.MeansKeyCheque
	case paymentMeansCreditTransfer:
		return pay.MeansKeyCreditTransfer
	case paymentMeansBankAccount:
		return pay.MeansKeyDebitTransfer
	case paymentMeansCard:
		return pay.MeansKeyCard
	case paymentMeansSEPACreditTransfer:
		return keyPaymentMeansSEPACreditTransfer
	case paymentMeansSEPADirectDebit:
		return keyPaymentMeansSEPADirectDebit
	case paymentMeansDirectDebit:
		return pay.MeansKeyDirectDebit
	default:
		return pay.MeansKeyOther
	}
}
