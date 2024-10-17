package ubl

import (
	"github.com/invopop/gobl.ubl/structs"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

// ParseAllowanceCharges extracts the charges logic from the CII document
func ParseUtoGCharges(doc *structs.Invoice) ([]*bill.Charge, []*bill.Discount) {
	var charges []*bill.Charge
	var discounts []*bill.Discount

	for _, allowanceCharge := range doc.AllowanceCharge {
		if allowanceCharge.ChargeIndicator {
			// This is a charge
			charge := &bill.Charge{}
			if allowanceCharge.AllowanceChargeReason != nil {
				charge.Reason = *allowanceCharge.AllowanceChargeReason
			}
			if allowanceCharge.Amount.Value != "" {
				charge.Amount, _ = num.AmountFromString(allowanceCharge.Amount.Value)
			}
			if allowanceCharge.AllowanceChargeReasonCode != nil {
				charge.Code = *allowanceCharge.AllowanceChargeReasonCode
			}
			if allowanceCharge.BaseAmount != nil {
				basis, _ := num.AmountFromString(allowanceCharge.BaseAmount.Value)
				charge.Base = &basis
			}
			if allowanceCharge.MultiplierFactorNumeric != nil {
				percent, _ := num.PercentageFromString(*allowanceCharge.MultiplierFactorNumeric + "%")
				charge.Percent = &percent
			}
			if allowanceCharge.TaxCategory != nil && allowanceCharge.TaxCategory.TaxScheme != nil {
				charge.Taxes = tax.Set{
					{
						Category: cbc.Code(*allowanceCharge.TaxCategory.TaxScheme.ID),
						Rate:     FindTaxKey(allowanceCharge.TaxCategory.ID),
					},
				}
				if allowanceCharge.TaxCategory.Percent != nil {
					percent, _ := num.PercentageFromString(*allowanceCharge.TaxCategory.Percent + "%")
					charge.Taxes[0].Percent = &percent
				}
			}
			if charges == nil {
				charges = make([]*bill.Charge, 0)
			}
			charges = append(charges, charge)
		} else {
			// This is a discount
			discount := &bill.Discount{}
			if allowanceCharge.AllowanceChargeReason != nil {
				discount.Reason = *allowanceCharge.AllowanceChargeReason
			}
			if allowanceCharge.Amount.Value != "" {
				discount.Amount, _ = num.AmountFromString(allowanceCharge.Amount.Value)
			}
			if allowanceCharge.AllowanceChargeReasonCode != nil {
				discount.Code = *allowanceCharge.AllowanceChargeReasonCode
			}
			if allowanceCharge.BaseAmount != nil {
				basis, _ := num.AmountFromString(allowanceCharge.BaseAmount.Value)
				discount.Base = &basis
			}
			if allowanceCharge.MultiplierFactorNumeric != nil {
				percent, _ := num.PercentageFromString(*allowanceCharge.MultiplierFactorNumeric + "%")
				discount.Percent = &percent
			}
			if allowanceCharge.TaxCategory != nil && allowanceCharge.TaxCategory.TaxScheme != nil {
				discount.Taxes = tax.Set{
					{
						Category: cbc.Code(*allowanceCharge.TaxCategory.TaxScheme.ID),
						Rate:     FindTaxKey(allowanceCharge.TaxCategory.ID),
					},
				}
				if allowanceCharge.TaxCategory.Percent != nil {
					percent, _ := num.PercentageFromString(*allowanceCharge.TaxCategory.Percent + "%")
					discount.Taxes[0].Percent = &percent
				}
			}
			if discounts == nil {
				discounts = make([]*bill.Discount, 0)
			}
			discounts = append(discounts, discount)
		}
	}

	return charges, discounts
}
