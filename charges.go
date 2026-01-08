package ubl

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
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
	for i, ch := range inv.Charges {
		ui.AllowanceCharge[i] = makeCharge(ch, string(inv.Currency))
	}
	for i, d := range inv.Discounts {
		ui.AllowanceCharge[i+len(inv.Charges)] = makeDiscount(d, string(inv.Currency))
	}
}

func makeCharge(ch *bill.Charge, ccy string) AllowanceCharge {
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
		p := ch.Percent.Base().String()
		c.MultiplierFactorNumeric = &p
	}
	if ch.Taxes != nil {
		c.TaxCategory = makeTaxCategory(ch.Taxes)
	}

	return c
}

func makeDiscount(d *bill.Discount, ccy string) AllowanceCharge {
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
		p := d.Percent.Base().String()
		c.MultiplierFactorNumeric = &p
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
		category.TaxScheme = &TaxScheme{ID: t.Category.String()}
		if t.Percent != nil {
			p := t.Percent.StringWithoutSymbol()
			category.Percent = &p
		}
		e := t.Ext.Get(untdid.ExtKeyTaxCategory).String()
		if e != "" {
			category.ID = &e
		}
		set = append(set, &category)
	}
	return set
}
