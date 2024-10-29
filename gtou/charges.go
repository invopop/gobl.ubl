package gtou

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/tax"
)

func (c *Converter) newCharges(inv *bill.Invoice) error {
	if inv.Charges == nil || inv.Discounts == nil {
		return nil
	}
	c.doc.AllowanceCharge = make([]AllowanceCharge, len(inv.Charges)+len(inv.Discounts))
	for i, charge := range inv.Charges {
		c.doc.AllowanceCharge[i] = makeCharge(charge, string(inv.Currency))
	}
	for i, discount := range inv.Discounts {
		c.doc.AllowanceCharge[i+len(inv.Charges)] = makeDiscount(discount, string(inv.Currency))
	}
	return nil
}

func makeCharge(charge *bill.Charge, currency string) AllowanceCharge {
	c := AllowanceCharge{
		ChargeIndicator: true,
		Amount: Amount{
			Value:      charge.Amount.String(),
			CurrencyID: &currency,
		},
	}
	if charge.Reason != "" {
		c.AllowanceChargeReason = &charge.Reason
	}
	if charge.Code != "" {
		c.AllowanceChargeReasonCode = &charge.Code
	}
	if charge.Percent != nil {
		p := charge.Percent.String()
		c.MultiplierFactorNumeric = &p
	}
	if charge.Taxes != nil {
		c.TaxCategory = makeTaxCategory(charge.Taxes)
	}

	return c
}

func makeDiscount(discount *bill.Discount, currency string) AllowanceCharge {
	c := AllowanceCharge{
		ChargeIndicator: false,
		Amount: Amount{
			Value:      discount.Amount.String(),
			CurrencyID: &currency,
		},
	}
	if discount.Reason != "" {
		c.AllowanceChargeReason = &discount.Reason
	}
	if discount.Code != "" {
		c.AllowanceChargeReasonCode = &discount.Code
	}
	if discount.Percent != nil {
		p := discount.Percent.String()
		c.MultiplierFactorNumeric = &p
	}
	if discount.Taxes != nil {
		c.TaxCategory = makeTaxCategory(discount.Taxes)
	}

	return c
}

func makeTaxCategory(taxes tax.Set) *[]TaxCategory {
	set := []TaxCategory{}
	for _, tax := range taxes {
		category := TaxCategory{}
		c := tax.Category.String()
		category.TaxScheme = &TaxScheme{ID: &c}
		if tax.Percent != nil {
			p := tax.Percent.StringWithoutSymbol()
			category.Percent = &p
		}
		if tax.Rate != "" {
			rate := findTaxCode(tax.Rate)
			category.ID = &rate
		}
		set = append(set, category)
	}
	return &set
}
