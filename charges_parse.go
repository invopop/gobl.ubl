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

	// OIOUBL emits MultiplierFactorNumeric as the decimal factor (0.05 for 5%)
	// rather than the percent number (5) used by other profiles.
	oioubl := ui.CustomizationID == ContextOIOUBL21.CustomizationID

	for _, allowanceCharge := range ui.AllowanceCharge {
		if allowanceCharge.ChargeIndicator {
			charge, err := goblCharge(&allowanceCharge, taxCategoryMap, oioubl)
			if err != nil {
				return err
			}
			if charges == nil {
				charges = make([]*bill.Charge, 0)
			}
			charges = append(charges, charge)
		} else {
			discount, err := goblDiscount(&allowanceCharge, taxCategoryMap, oioubl)
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

// goblAllowancePercent parses an AllowanceCharge MultiplierFactorNumeric into a
// GOBL percentage, or returns nil when none is present. OIOUBL stores the
// decimal factor (0.05 = 5%), which PercentageFromString reads directly; other
// profiles store the percent number (5), which needs the % suffix.
func goblAllowancePercent(ac *AllowanceCharge, oioubl bool) (*num.Percentage, error) {
	if ac.MultiplierFactorNumeric == nil {
		return nil, nil
	}
	multiplier := normalizeNumericString(*ac.MultiplierFactorNumeric)
	if !oioubl && !strings.HasSuffix(multiplier, "%") {
		multiplier += "%"
	}
	p, err := num.PercentageFromString(multiplier)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func goblCharge(ac *AllowanceCharge, taxCategoryMap map[string]*taxCategoryInfo, oioubl bool) (*bill.Charge, error) {
	ch := &bill.Charge{}
	if ac.AllowanceChargeReason != nil {
		ch.Reason = *ac.AllowanceChargeReason
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
	if ac.BaseAmount != nil {
		b, err := num.AmountFromString(normalizeNumericString(ac.BaseAmount.Value))
		if err != nil {
			return nil, err
		}
		ch.Base = &b
	}
	pct, err := goblAllowancePercent(ac, oioubl)
	if err != nil {
		return nil, err
	}
	if pct != nil {
		ch.Percent = pct

		// Check if there is a base amount
		if ac.BaseAmount != nil {
			base, err := num.AmountFromString(normalizeNumericString(ac.BaseAmount.Value))
			if err != nil {
				return nil, err
			}
			ch.Base = &base
		}
	}
	if len(ac.TaxCategory) > 0 && ac.TaxCategory[0].TaxScheme != nil {
		ch.Taxes = tax.Set{
			{
				Category: goblTaxSchemeCategory(ac.TaxCategory[0].TaxScheme.ID.Value),
			},
		}

		// Add tax category ID to extensions
		if ac.TaxCategory[0].ID != nil {
			ch.Taxes[0].Ext = ch.Taxes[0].Ext.Set(untdid.ExtKeyTaxCategory, goblTaxCategoryCode(ac.TaxCategory[0].ID.Value))

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

func goblDiscount(ac *AllowanceCharge, taxCategoryMap map[string]*taxCategoryInfo, oioubl bool) (*bill.Discount, error) {
	d := &bill.Discount{}
	if ac.AllowanceChargeReason != nil {
		d.Reason = *ac.AllowanceChargeReason
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
	if ac.BaseAmount != nil {
		b, err := num.AmountFromString(normalizeNumericString(ac.BaseAmount.Value))
		if err != nil {
			return nil, err
		}
		d.Base = &b
	}
	pct, err := goblAllowancePercent(ac, oioubl)
	if err != nil {
		return nil, err
	}
	if pct != nil {
		d.Percent = pct

		// Check if there is a base amount
		if ac.BaseAmount != nil {
			base, err := num.AmountFromString(normalizeNumericString(ac.BaseAmount.Value))
			if err != nil {
				return nil, err
			}
			d.Base = &base
		}
	}
	if len(ac.TaxCategory) > 0 && ac.TaxCategory[0].TaxScheme != nil {
		d.Taxes = tax.Set{
			{
				Category: goblTaxSchemeCategory(ac.TaxCategory[0].TaxScheme.ID.Value),
			},
		}

		// Add tax category ID to extensions
		if ac.TaxCategory[0].ID != nil {
			d.Taxes[0].Ext = d.Taxes[0].Ext.Set(untdid.ExtKeyTaxCategory, goblTaxCategoryCode(ac.TaxCategory[0].ID.Value))

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

func goblLineCharge(ac *AllowanceCharge, oioubl bool) (*bill.LineCharge, error) {
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
		ch.Reason = *ac.AllowanceChargeReason
	}
	pct, err := goblAllowancePercent(ac, oioubl)
	if err != nil {
		return nil, err
	}
	if pct != nil {
		ch.Percent = pct

		// Check if there is a base amount
		if ac.BaseAmount != nil {
			base, err := num.AmountFromString(normalizeNumericString(ac.BaseAmount.Value))
			if err != nil {
				return nil, err
			}
			ch.Base = &base
		}
	}
	return ch, nil
}

func goblLineDiscount(ac *AllowanceCharge, oioubl bool) (*bill.LineDiscount, error) {
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
		d.Reason = *ac.AllowanceChargeReason
	}
	pct, err := goblAllowancePercent(ac, oioubl)
	if err != nil {
		return nil, err
	}
	if pct != nil {
		d.Percent = pct

		// Check if there is a base amount
		if ac.BaseAmount != nil {
			base, err := num.AmountFromString(normalizeNumericString(ac.BaseAmount.Value))
			if err != nil {
				return nil, err
			}
			d.Base = &base
		}
	}
	return d, nil
}
