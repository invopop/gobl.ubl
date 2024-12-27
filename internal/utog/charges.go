package utog

import (
	"strings"

	"github.com/invopop/gobl.ubl/document"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

// ParseAllowanceCharges extracts the charges logic from the CII document
func (c *Converter) getCharges(doc *document.Invoice) error {
	var charges []*bill.Charge
	var discounts []*bill.Discount

	for _, allowanceCharge := range doc.AllowanceCharge {
		if allowanceCharge.ChargeIndicator {
			charge, err := c.parseCharge(&allowanceCharge)
			if err != nil {
				return err
			}
			if charges == nil {
				charges = make([]*bill.Charge, 0)
			}
			charges = append(charges, charge)
		} else {
			discount, err := c.parseDiscount(&allowanceCharge)
			if err != nil {
				return err
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

func (c *Converter) parseCharge(ac *document.AllowanceCharge) (*bill.Charge, error) {
	ch := &bill.Charge{}
	if ac.AllowanceChargeReason != nil {
		ch.Reason = *ac.AllowanceChargeReason
	}
	if ac.Amount.Value != "" {
		a, err := num.AmountFromString(ac.Amount.Value)
		if err != nil {
			return nil, err
		}
		ch.Amount = a
	}
	if ac.AllowanceChargeReasonCode != nil {
		ch.Ext = tax.Extensions{
			untdid.ExtKeyCharge: cbc.Code(*ac.AllowanceChargeReasonCode),
		}
	}
	if ac.BaseAmount != nil {
		b, err := num.AmountFromString(ac.BaseAmount.Value)
		if err != nil {
			return nil, err
		}
		ch.Base = &b
	}
	if ac.MultiplierFactorNumeric != nil {
		if !strings.HasSuffix(*ac.MultiplierFactorNumeric, "%") {
			*ac.MultiplierFactorNumeric += "%"
		}
		p, err := num.PercentageFromString(*ac.MultiplierFactorNumeric)
		if err != nil {
			return nil, err
		}
		ch.Percent = &p
	}
	if len(ac.TaxCategory) > 0 && ac.TaxCategory[0].TaxScheme != nil {
		ch.Taxes = tax.Set{
			{
				Category: cbc.Code(ac.TaxCategory[0].TaxScheme.ID),
			},
		}
		if ac.TaxCategory[0].Percent != nil {
			if !strings.HasSuffix(*ac.TaxCategory[0].Percent, "%") {
				*ac.TaxCategory[0].Percent += "%"
			}
			p, err := num.PercentageFromString(*ac.TaxCategory[0].Percent)
			if err != nil {
				return nil, err
			}
			ch.Taxes[0].Percent = &p
		}
	}
	return ch, nil
}

func (c *Converter) parseDiscount(ac *document.AllowanceCharge) (*bill.Discount, error) {
	d := &bill.Discount{}
	if ac.AllowanceChargeReason != nil {
		d.Reason = *ac.AllowanceChargeReason
	}
	if ac.Amount.Value != "" {
		a, err := num.AmountFromString(ac.Amount.Value)
		if err != nil {
			return nil, err
		}
		d.Amount = a
	}
	if ac.AllowanceChargeReasonCode != nil {
		d.Ext = tax.Extensions{
			untdid.ExtKeyAllowance: cbc.Code(*ac.AllowanceChargeReasonCode),
		}
	}
	if ac.BaseAmount != nil {
		b, err := num.AmountFromString(ac.BaseAmount.Value)
		if err != nil {
			return nil, err
		}
		d.Base = &b
	}
	if ac.MultiplierFactorNumeric != nil {
		if !strings.HasSuffix(*ac.MultiplierFactorNumeric, "%") {
			*ac.MultiplierFactorNumeric += "%"
		}
		p, err := num.PercentageFromString(*ac.MultiplierFactorNumeric)
		if err != nil {
			return nil, err
		}
		d.Percent = &p
	}
	if len(ac.TaxCategory) > 0 && ac.TaxCategory[0].TaxScheme != nil {
		d.Taxes = tax.Set{
			{
				Category: cbc.Code(ac.TaxCategory[0].TaxScheme.ID),
			},
		}
		if ac.TaxCategory[0].Percent != nil {
			if !strings.HasSuffix(*ac.TaxCategory[0].Percent, "%") {
				*ac.TaxCategory[0].Percent += "%"
			}
			percent, err := num.PercentageFromString(*ac.TaxCategory[0].Percent)
			if err != nil {
				return nil, err
			}
			d.Taxes[0].Percent = &percent
		}
	}
	return d, nil
}

func getLineCharge(ac *document.AllowanceCharge) (*bill.LineCharge, error) {
	amount, err := num.AmountFromString(ac.Amount.Value)
	if err != nil {
		return nil, err
	}
	ch := &bill.LineCharge{
		Amount: amount,
	}
	if ac.AllowanceChargeReasonCode != nil {
		ch.Ext = tax.Extensions{
			untdid.ExtKeyCharge: cbc.Code(*ac.AllowanceChargeReasonCode),
		}
	}
	if ac.AllowanceChargeReason != nil {
		ch.Reason = *ac.AllowanceChargeReason
	}
	if ac.MultiplierFactorNumeric != nil {
		if !strings.HasSuffix(*ac.MultiplierFactorNumeric, "%") {
			*ac.MultiplierFactorNumeric += "%"
		}
		percent, err := num.PercentageFromString(*ac.MultiplierFactorNumeric)
		if err != nil {
			return nil, err
		}
		ch.Percent = &percent
	}
	return ch, nil
}

func getLineDiscount(ac *document.AllowanceCharge) (*bill.LineDiscount, error) {
	a, err := num.AmountFromString(ac.Amount.Value)
	if err != nil {
		return nil, err
	}
	d := &bill.LineDiscount{
		Amount: a,
	}
	if ac.AllowanceChargeReasonCode != nil {
		d.Ext = tax.Extensions{
			untdid.ExtKeyAllowance: cbc.Code(*ac.AllowanceChargeReasonCode),
		}
	}
	if ac.AllowanceChargeReason != nil {
		d.Reason = *ac.AllowanceChargeReason
	}
	if ac.MultiplierFactorNumeric != nil {
		if !strings.HasSuffix(*ac.MultiplierFactorNumeric, "%") {
			*ac.MultiplierFactorNumeric += "%"
		}
		p, err := num.PercentageFromString(*ac.MultiplierFactorNumeric)
		if err != nil {
			return nil, err
		}
		d.Percent = &p
	}
	return d, nil
}
