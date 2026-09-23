package ubl

import (
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/cef"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

// goblAddCharges adds the invoice charges to the gobl output.
func (ui *Invoice) goblAddCharges(out *bill.Invoice) error {
	var charges []*bill.Charge
	var discounts []*bill.Discount

	// Build tax category map from TaxTotal
	taxCategoryMap := ui.buildTaxCategoryMap()

	for _, allowanceCharge := range ui.AllowanceCharge {
		if allowanceCharge.ChargeIndicator {
			charge, err := goblCharge(&allowanceCharge, taxCategoryMap)
			if err != nil {
				return err
			}
			if charges == nil {
				charges = make([]*bill.Charge, 0)
			}
			charges = append(charges, charge)
		} else {
			discount, err := goblDiscount(&allowanceCharge, taxCategoryMap)
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
		out.Charges = charges
	}
	if discounts != nil {
		out.Discounts = discounts
	}
	return nil
}

func goblCharge(ac *AllowanceCharge, taxCategoryMap map[string]*taxCategoryInfo) (*bill.Charge, error) {
	ch := &bill.Charge{}
	if ac.AllowanceChargeReason != nil {
		ch.Reason = cleanString(*ac.AllowanceChargeReason)
	}
	if ac.Amount.Value != "" {
		a, err := num.AmountFromString(normalizeNumericString(ac.Amount.Value))
		if err != nil {
			return nil, err
		}
		ch.Amount = a
	}
	if ac.AllowanceChargeReasonCode != nil {
		ch.Ext = tax.ExtensionsOf(cbc.CodeMap{
			untdid.ExtKeyCharge: cbc.Code(*ac.AllowanceChargeReasonCode),
		})
	}
	base, percent, err := goblACBasis(ac, ch.Amount)
	if err != nil {
		return nil, err
	}
	ch.Base, ch.Percent = base, percent
	if len(ac.TaxCategory) > 0 && ac.TaxCategory[0].TaxScheme != nil {
		ch.Taxes = tax.Set{
			{
				Category: cbc.Code(ac.TaxCategory[0].TaxScheme.ID.Value),
			},
		}

		// Add tax category ID to extensions
		if ac.TaxCategory[0].ID != nil {
			ch.Taxes[0].Ext = ch.Taxes[0].Ext.Set(untdid.ExtKeyTaxCategory, cbc.Code(ac.TaxCategory[0].ID.Value))

			// Look up exemption code from TaxTotal
			key := buildTaxCategoryKey(ac.TaxCategory[0].TaxScheme.ID.Value, ac.TaxCategory[0].ID.Value, ac.TaxCategory[0].Percent)
			if info, ok := taxCategoryMap[key]; ok && info.exemptionReasonCode != "" {
				ch.Taxes[0].Ext = ch.Taxes[0].Ext.Set(cef.ExtKeyVATEX, cbc.Code(info.exemptionReasonCode))
			}
		}

		if ac.TaxCategory[0].Percent != nil {
			percent := normalizeNumericString(*ac.TaxCategory[0].Percent)
			if !strings.HasSuffix(percent, "%") {
				percent += "%"
			}
			p, err := num.PercentageFromString(percent)
			if err != nil {
				return nil, err
			}

			// Skip setting percent if it's 0% and tax category is not "Z" (zero-rated)
			// This prevents GOBL from normalizing to "zero" tax rate for exempt/reverse-charge cases
			if !p.IsZero() || (ac.TaxCategory[0].ID != nil && ac.TaxCategory[0].ID.Value == "Z") {
				ch.Taxes[0].Percent = &p
			}
		}
	}
	return ch, nil
}

func goblDiscount(ac *AllowanceCharge, taxCategoryMap map[string]*taxCategoryInfo) (*bill.Discount, error) {
	d := &bill.Discount{}
	if ac.AllowanceChargeReason != nil {
		d.Reason = cleanString(*ac.AllowanceChargeReason)
	}
	if ac.Amount.Value != "" {
		a, err := num.AmountFromString(normalizeNumericString(ac.Amount.Value))
		if err != nil {
			return nil, err
		}
		d.Amount = a
	}
	if ac.AllowanceChargeReasonCode != nil {
		d.Ext = tax.ExtensionsOf(cbc.CodeMap{
			untdid.ExtKeyAllowance: cbc.Code(*ac.AllowanceChargeReasonCode),
		})
	}
	base, percent, err := goblACBasis(ac, d.Amount)
	if err != nil {
		return nil, err
	}
	d.Base, d.Percent = base, percent
	if len(ac.TaxCategory) > 0 && ac.TaxCategory[0].TaxScheme != nil {
		d.Taxes = tax.Set{
			{
				Category: cbc.Code(ac.TaxCategory[0].TaxScheme.ID.Value),
			},
		}

		// Add tax category ID to extensions
		if ac.TaxCategory[0].ID != nil {
			d.Taxes[0].Ext = d.Taxes[0].Ext.Set(untdid.ExtKeyTaxCategory, cbc.Code(ac.TaxCategory[0].ID.Value))

			// Look up exemption code from TaxTotal
			key := buildTaxCategoryKey(ac.TaxCategory[0].TaxScheme.ID.Value, ac.TaxCategory[0].ID.Value, ac.TaxCategory[0].Percent)
			if info, ok := taxCategoryMap[key]; ok && info.exemptionReasonCode != "" {
				d.Taxes[0].Ext = d.Taxes[0].Ext.Set(cef.ExtKeyVATEX, cbc.Code(info.exemptionReasonCode))
			}
		}

		if ac.TaxCategory[0].Percent != nil {
			percentStr := normalizeNumericString(*ac.TaxCategory[0].Percent)
			if !strings.HasSuffix(percentStr, "%") {
				percentStr += "%"
			}
			percent, err := num.PercentageFromString(percentStr)
			if err != nil {
				return nil, err
			}

			// Skip setting percent if it's 0% and tax category is not "Z" (zero-rated)
			// This prevents GOBL from normalizing to "zero" tax rate for exempt/reverse-charge cases
			if !percent.IsZero() || (ac.TaxCategory[0].ID != nil && ac.TaxCategory[0].ID.Value == "Z") {
				d.Taxes[0].Percent = &percent
			}
		}
	}
	return d, nil
}

func goblLineCharge(ac *AllowanceCharge) (*bill.LineCharge, error) {
	amount, err := num.AmountFromString(normalizeNumericString(ac.Amount.Value))
	if err != nil {
		return nil, err
	}
	ch := &bill.LineCharge{
		Amount: amount,
	}
	if ac.AllowanceChargeReasonCode != nil {
		ch.Ext = tax.ExtensionsOf(cbc.CodeMap{
			untdid.ExtKeyCharge: cbc.Code(*ac.AllowanceChargeReasonCode),
		})
	}
	if ac.AllowanceChargeReason != nil {
		ch.Reason = cleanString(*ac.AllowanceChargeReason)
	}
	base, percent, err := goblACBasis(ac, ch.Amount)
	if err != nil {
		return nil, err
	}
	ch.Base, ch.Percent = base, percent
	return ch, nil
}

func goblLineDiscount(ac *AllowanceCharge) (*bill.LineDiscount, error) {
	a, err := num.AmountFromString(normalizeNumericString(ac.Amount.Value))
	if err != nil {
		return nil, err
	}
	d := &bill.LineDiscount{
		Amount: a,
	}
	if ac.AllowanceChargeReasonCode != nil {
		d.Ext = tax.ExtensionsOf(cbc.CodeMap{
			untdid.ExtKeyAllowance: cbc.Code(*ac.AllowanceChargeReasonCode),
		})
	}
	if ac.AllowanceChargeReason != nil {
		d.Reason = cleanString(*ac.AllowanceChargeReason)
	}
	base, percent, err := goblACBasis(ac, d.Amount)
	if err != nil {
		return nil, err
	}
	d.Base, d.Percent = base, percent
	return d, nil
}

// goblACBasis parses the base amount (BT-137 at line level, BT-142 at document
// level) and decides if the multiplier can be used alongside the declared amount.
func goblACBasis(ac *AllowanceCharge, amount num.Amount) (*num.Amount, *num.Percentage, error) {
	var base *num.Amount
	if ac.BaseAmount != nil {
		b, err := num.AmountFromString(normalizeNumericString(ac.BaseAmount.Value))
		if err != nil {
			return nil, nil, err
		}
		base = &b
	}
	// GOBL requires a percentage wherever a base is set, and without one the
	// base has nothing to apply to, so they are only ever returned together.
	if ac.MultiplierFactorNumeric == nil {
		return nil, nil, nil
	}
	multiplier := normalizeNumericString(*ac.MultiplierFactorNumeric)
	p, err := num.PercentageFromString(strings.TrimSuffix(multiplier, "%") + "%")
	if err != nil {
		return nil, nil, err
	}
	if ac.Amount.Value == "" {
		// Without a declared amount the multiplier is the only way to derive one.
		return base, &p, nil
	}
	if base == nil || !p.Of(*base).Rescale(amount.Exp()).Equals(amount) {
		return nil, nil, nil
	}
	return base, &p, nil
}
