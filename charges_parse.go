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

// taxCategoryInfo holds tax category information from TaxTotal
type taxCategoryInfo struct {
	exemptionReasonCode string
}

// buildTaxCategoryMap builds a map of tax category information from TaxTotal
func (ui *Invoice) buildTaxCategoryMap() map[string]*taxCategoryInfo {
	categoryMap := make(map[string]*taxCategoryInfo)

	for _, taxTotal := range ui.TaxTotal {
		for _, subtotal := range taxTotal.TaxSubtotal {
			if subtotal.TaxCategory.ID != nil && subtotal.TaxCategory.TaxScheme != nil {
				key := subtotal.TaxCategory.TaxScheme.ID + ":" + *subtotal.TaxCategory.ID
				info := &taxCategoryInfo{}
				if subtotal.TaxCategory.TaxExemptionReasonCode != nil {
					info.exemptionReasonCode = *subtotal.TaxCategory.TaxExemptionReasonCode
				}
				categoryMap[key] = info
			}
		}
	}

	return categoryMap
}

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

		// Check if there is a base amount
		if ac.BaseAmount != nil {
			base, err := num.AmountFromString(ac.BaseAmount.Value)
			if err != nil {
				return nil, err
			}
			ch.Base = &base
		}
	}
	if len(ac.TaxCategory) > 0 && ac.TaxCategory[0].TaxScheme != nil {
		ch.Taxes = tax.Set{
			{
				Category: cbc.Code(ac.TaxCategory[0].TaxScheme.ID),
			},
		}

		// Add tax category ID to extensions
		if ac.TaxCategory[0].ID != nil {
			if ch.Taxes[0].Ext == nil {
				ch.Taxes[0].Ext = tax.Extensions{}
			}
			ch.Taxes[0].Ext[untdid.ExtKeyTaxCategory] = cbc.Code(*ac.TaxCategory[0].ID)

			// Look up exemption code from TaxTotal
			key := ac.TaxCategory[0].TaxScheme.ID + ":" + *ac.TaxCategory[0].ID
			if info, ok := taxCategoryMap[key]; ok && info.exemptionReasonCode != "" {
				ch.Taxes[0].Ext[cef.ExtKeyVATEX] = cbc.Code(info.exemptionReasonCode)
			}
		}

		if ac.TaxCategory[0].Percent != nil {
			if !strings.HasSuffix(*ac.TaxCategory[0].Percent, "%") {
				*ac.TaxCategory[0].Percent += "%"
			}
			p, err := num.PercentageFromString(*ac.TaxCategory[0].Percent)
			if err != nil {
				return nil, err
			}

			// Skip setting percent if it's 0% and tax category is not "Z" (zero-rated)
			// This prevents GOBL from normalizing to "zero" tax rate for exempt/reverse-charge cases
			if !p.IsZero() || (ac.TaxCategory[0].ID != nil && *ac.TaxCategory[0].ID == "Z") {
				ch.Taxes[0].Percent = &p
			}
		}
	}
	return ch, nil
}

func goblDiscount(ac *AllowanceCharge, taxCategoryMap map[string]*taxCategoryInfo) (*bill.Discount, error) {
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

		// Check if there is a base amount
		if ac.BaseAmount != nil {
			base, err := num.AmountFromString(ac.BaseAmount.Value)
			if err != nil {
				return nil, err
			}
			d.Base = &base
		}
	}
	if len(ac.TaxCategory) > 0 && ac.TaxCategory[0].TaxScheme != nil {
		d.Taxes = tax.Set{
			{
				Category: cbc.Code(ac.TaxCategory[0].TaxScheme.ID),
			},
		}

		// Add tax category ID to extensions
		if ac.TaxCategory[0].ID != nil {
			if d.Taxes[0].Ext == nil {
				d.Taxes[0].Ext = tax.Extensions{}
			}
			d.Taxes[0].Ext[untdid.ExtKeyTaxCategory] = cbc.Code(*ac.TaxCategory[0].ID)

			// Look up exemption code from TaxTotal
			key := ac.TaxCategory[0].TaxScheme.ID + ":" + *ac.TaxCategory[0].ID
			if info, ok := taxCategoryMap[key]; ok && info.exemptionReasonCode != "" {
				d.Taxes[0].Ext[cef.ExtKeyVATEX] = cbc.Code(info.exemptionReasonCode)
			}
		}

		if ac.TaxCategory[0].Percent != nil {
			if !strings.HasSuffix(*ac.TaxCategory[0].Percent, "%") {
				*ac.TaxCategory[0].Percent += "%"
			}
			percent, err := num.PercentageFromString(*ac.TaxCategory[0].Percent)
			if err != nil {
				return nil, err
			}

			// Skip setting percent if it's 0% and tax category is not "Z" (zero-rated)
			// This prevents GOBL from normalizing to "zero" tax rate for exempt/reverse-charge cases
			if !percent.IsZero() || (ac.TaxCategory[0].ID != nil && *ac.TaxCategory[0].ID == "Z") {
				d.Taxes[0].Percent = &percent
			}
		}
	}
	return d, nil
}

func goblLineCharge(ac *AllowanceCharge) (*bill.LineCharge, error) {
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

		// Check if there is a base amount
		if ac.BaseAmount != nil {
			base, err := num.AmountFromString(ac.BaseAmount.Value)
			if err != nil {
				return nil, err
			}
			ch.Base = &base
		}
	}
	return ch, nil
}

func goblLineDiscount(ac *AllowanceCharge) (*bill.LineDiscount, error) {
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

		// Check if there is a base amount
		if ac.BaseAmount != nil {
			base, err := num.AmountFromString(ac.BaseAmount.Value)
			if err != nil {
				return nil, err
			}
			d.Base = &base
		}
	}
	return d, nil
}
