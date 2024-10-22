package utog

import (
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

// ParseAllowanceCharges extracts the charges logic from the CII document
func (c *Conversor) getCharges(doc *Document) error {
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
				amount, err := num.AmountFromString(allowanceCharge.Amount.Value)
				if err != nil {
					return err
				}
				charge.Amount = amount
			}
			if allowanceCharge.AllowanceChargeReasonCode != nil {
				charge.Code = *allowanceCharge.AllowanceChargeReasonCode
			}
			if allowanceCharge.BaseAmount != nil {
				basis, err := num.AmountFromString(allowanceCharge.BaseAmount.Value)
				if err != nil {
					return err
				}
				charge.Base = &basis
			}
			if allowanceCharge.MultiplierFactorNumeric != nil {
				if !strings.HasSuffix(*allowanceCharge.MultiplierFactorNumeric, "%") {
					*allowanceCharge.MultiplierFactorNumeric += "%"
				}
				percent, err := num.PercentageFromString(*allowanceCharge.MultiplierFactorNumeric)
				if err != nil {
					return err
				}
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
					if !strings.HasSuffix(*allowanceCharge.TaxCategory.Percent, "%") {
						*allowanceCharge.TaxCategory.Percent += "%"
					}
					percent, err := num.PercentageFromString(*allowanceCharge.TaxCategory.Percent)
					if err != nil {
						return err
					}
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
				amount, err := num.AmountFromString(allowanceCharge.Amount.Value)
				if err != nil {
					return err
				}
				discount.Amount = amount
			}
			if allowanceCharge.AllowanceChargeReasonCode != nil {
				discount.Code = *allowanceCharge.AllowanceChargeReasonCode
			}
			if allowanceCharge.BaseAmount != nil {
				basis, err := num.AmountFromString(allowanceCharge.BaseAmount.Value)
				if err != nil {
					return err
				}
				discount.Base = &basis
			}
			if allowanceCharge.MultiplierFactorNumeric != nil {
				if !strings.HasSuffix(*allowanceCharge.MultiplierFactorNumeric, "%") {
					*allowanceCharge.MultiplierFactorNumeric += "%"
				}
				percent, err := num.PercentageFromString(*allowanceCharge.MultiplierFactorNumeric)
				if err != nil {
					return err
				}
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
					if !strings.HasSuffix(*allowanceCharge.TaxCategory.Percent, "%") {
						*allowanceCharge.TaxCategory.Percent += "%"
					}
					percent, err := num.PercentageFromString(*allowanceCharge.TaxCategory.Percent)
					if err != nil {
						return err
					}
					discount.Taxes[0].Percent = &percent
				}
			}
			if discounts == nil {
				discounts = make([]*bill.Discount, 0)
			}
			discounts = append(discounts, discount)
		}
	}
	if charges != nil {
		c.inv.Charges = charges
	}
	if discounts != nil {
		c.inv.Discounts = discounts
	}
	return nil
}
