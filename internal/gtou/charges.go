package gtou

import (
	"github.com/invopop/gobl.ubl/document"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/tax"
)

func (c *Converter) newCharges(inv *bill.Invoice) error {
	if inv.Charges == nil && inv.Discounts == nil {
		return nil
	}
	c.doc.AllowanceCharge = make([]document.AllowanceCharge, len(inv.Charges)+len(inv.Discounts))
	for i, ch := range inv.Charges {
		c.doc.AllowanceCharge[i] = makeCharge(ch, string(inv.Currency))
	}
	for i, d := range inv.Discounts {
		c.doc.AllowanceCharge[i+len(inv.Charges)] = makeDiscount(d, string(inv.Currency))
	}
	return nil
}

func makeCharge(ch *bill.Charge, ccy string) document.AllowanceCharge {
	c := document.AllowanceCharge{
		ChargeIndicator: true,
		Amount: document.Amount{
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
		p := ch.Percent.String()
		c.MultiplierFactorNumeric = &p
	}
	if ch.Taxes != nil {
		c.TaxCategory = makeTaxCategory(ch.Taxes)
	}

	return c
}

func makeDiscount(d *bill.Discount, ccy string) document.AllowanceCharge {
	c := document.AllowanceCharge{
		ChargeIndicator: false,
		Amount: document.Amount{
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
		p := d.Percent.String()
		c.MultiplierFactorNumeric = &p
	}
	if d.Taxes != nil {
		c.TaxCategory = makeTaxCategory(d.Taxes)
	}

	return c
}

func makeTaxCategory(taxes tax.Set) []*document.TaxCategory {
	set := []*document.TaxCategory{}
	for _, t := range taxes {
		category := document.TaxCategory{}
		category.TaxScheme = &document.TaxScheme{ID: t.Category.String()}
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
