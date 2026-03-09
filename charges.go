package ubl

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

// AllowanceCharge represents an allowance or charge
type AllowanceCharge struct {
	ChargeIndicator           bool           `xml:"cbc:ChargeIndicator"`
	AllowanceChargeReasonCode *string        `xml:"cbc:AllowanceChargeReasonCode"`
	AllowanceChargeReason     *string        `xml:"cbc:AllowanceChargeReason"`
	MultiplierFactorNumeric   *string        `xml:"cbc:MultiplierFactorNumeric"`
	Amount                    Amount         `xml:"cbc:Amount"`
	BaseAmount                *Amount        `xml:"cbc:BaseAmount"`
	TaxCategory               []*TaxCategory `xml:"cac:TaxCategory"`
}

func (ui *Invoice) addCharges(inv *bill.Invoice) {
	if inv.Charges == nil && inv.Discounts == nil {
		return
	}
	ui.AllowanceCharge = make([]AllowanceCharge, len(inv.Charges)+len(inv.Discounts))
	// Use invoice sum (before discounts) as base amount for percentage calculations
	baseAmount := inv.Totals.Sum
	for i, ch := range inv.Charges {
		ui.AllowanceCharge[i] = makeCharge(ch, string(inv.Currency), baseAmount)
	}
	for i, d := range inv.Discounts {
		ui.AllowanceCharge[i+len(inv.Charges)] = makeDiscount(d, string(inv.Currency), baseAmount)
	}
}

func makeCharge(ch *bill.Charge, ccy string, baseAmount num.Amount) AllowanceCharge {
	c := AllowanceCharge{
		ChargeIndicator: true,
		Amount: Amount{
			Value:      ch.Amount.String(),
			CurrencyID: &ccy,
		},
	}
	if ch.Reason != "" {
		c.AllowanceChargeReason = &ch.Reason
	}
	e := ch.Ext.Get(untdid.ExtKeyCharge).String()
	if e != "" {
		c.AllowanceChargeReasonCode = &e
	}
	if ch.Percent != nil {
		p := ch.Percent.StringWithoutSymbol()
		c.MultiplierFactorNumeric = &p
		// Add BaseAmount when percentage is provided
		c.BaseAmount = &Amount{
			Value:      baseAmount.String(),
			CurrencyID: &ccy,
		}
	}
	if ch.Taxes != nil {
		c.TaxCategory = makeTaxCategory(ch.Taxes)
	}

	return c
}

func makeDiscount(d *bill.Discount, ccy string, baseAmount num.Amount) AllowanceCharge {
	c := AllowanceCharge{
		ChargeIndicator: false,
		Amount: Amount{
			Value:      d.Amount.String(),
			CurrencyID: &ccy,
		},
	}
	if d.Reason != "" {
		c.AllowanceChargeReason = &d.Reason
	}
	e := d.Ext.Get(untdid.ExtKeyAllowance).String()
	if e != "" {
		c.AllowanceChargeReasonCode = &e
	}
	if d.Percent != nil {
		p := d.Percent.StringWithoutSymbol()
		c.MultiplierFactorNumeric = &p
		// Add BaseAmount when percentage is provided
		c.BaseAmount = &Amount{
			Value:      baseAmount.String(),
			CurrencyID: &ccy,
		}
	}
	if d.Taxes != nil {
		c.TaxCategory = makeTaxCategory(d.Taxes)
	}

	return c
}

func makeTaxCategory(taxes tax.Set) []*TaxCategory {
	set := []*TaxCategory{}
	for _, t := range taxes {
		category := TaxCategory{}
		category.TaxScheme = &TaxScheme{ID: IDType{Value: t.Category.String()}}

		e := t.Ext.Get(untdid.ExtKeyTaxCategory).String()
		if e != "" {
			category.ID = &IDType{Value: e}
		}

		// Set percent: required unless category is "O" (outside scope)
		if t.Percent != nil {
			p := t.Percent.StringWithoutSymbol()
			category.Percent = &p
		} else if category.ID == nil || category.ID.Value != "O" {
			// Default to 0% when not outside scope
			zero := "0"
			category.Percent = &zero
		}

		set = append(set, &category)
	}
	return set
}
