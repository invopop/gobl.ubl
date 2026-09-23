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

// reconcileTotals aligns the document with the amounts the sender declared.
// BT-131 is mandatory (BR-24) and the totals are summed from it (BR-CO-10,
// BR-CO-13), while BT-149 is optional and in no rule, so BT-131 decides between
// them. What reconciles under no reading is tagged for bypass and recorded as
// sent.
func (ui *Invoice) reconcileTotals(out *bill.Invoice) error {
	ui.applyPayableRounding(out)

	if err := out.Calculate(); err != nil {
		return err
	}

	if swaps := ui.dropConflictingBaseQuantities(out); len(swaps) > 0 {
		if err := out.Calculate(); err != nil {
			return err
		}
		// A line matching neither reading keeps the standard one.
		if ui.restoreBaseQuantities(out, swaps) {
			if err := out.Calculate(); err != nil {
				return err
			}
		}
	}

	if ui.declaredTotalsAgree(out) {
		return nil
	}
	return ui.applyDeclaredTotals(out)
}

// priceSwap records a dropped base quantity so it can be put back.
type priceSwap struct {
	index int
	price num.Amount
}

// dropConflictingBaseQuantities removes the base quantity from any line that only
// matches its declared amount (BT-131) without it. Lines matching either way keep
// it, dividing being the reading the standard describes.
func (ui *Invoice) dropConflictingBaseQuantities(out *bill.Invoice) []priceSwap {
	var swaps []priceSwap
	for i, docLine := range ui.lines() {
		if i >= len(out.Lines) {
			break
		}
		line := out.Lines[i]
		if line == nil || line.Item == nil || line.Item.Price == nil || line.Total == nil {
			continue
		}
		if docLine.Price == nil || docLine.Price.BaseQuantity == nil {
			continue
		}
		declared, ok := goblDeclaredAmount(docLine.LineExtensionAmount)
		if !ok || declared.Equals(*line.Total) {
			continue
		}
		// Retry without the base quantity, the only other reading available.
		price, err := num.AmountFromString(normalizeNumericString(docLine.Price.PriceAmount.Value))
		if err != nil {
			continue
		}
		swaps = append(swaps, priceSwap{index: i, price: *line.Item.Price})
		line.Item.Price = &price
	}
	return swaps
}

// restoreBaseQuantities restores the standard price reading on swapped lines that
// still do not match.
func (ui *Invoice) restoreBaseQuantities(out *bill.Invoice, swaps []priceSwap) bool {
	lines := ui.lines()
	restored := false
	for _, s := range swaps {
		if s.index >= len(out.Lines) || s.index >= len(lines) {
			continue
		}
		line := out.Lines[s.index]
		if line == nil || line.Item == nil || line.Total == nil {
			continue
		}
		declared, ok := goblDeclaredAmount(lines[s.index].LineExtensionAmount)
		if ok && declared.Equals(*line.Total) {
			continue
		}
		price := s.price
		line.Item.Price = &price
		restored = true
	}
	return restored
}

// declaredTotalsAgree reports whether the calculated amounts reproduce every
// declared one: BT-131 per line, then BT-106, BT-109, BT-110, BT-112 and BT-115.
func (ui *Invoice) declaredTotalsAgree(out *bill.Invoice) bool {
	for i, docLine := range ui.lines() {
		if i >= len(out.Lines) {
			return false
		}
		line := out.Lines[i]
		if line == nil || line.Total == nil {
			continue
		}
		declared, ok := goblDeclaredAmount(docLine.LineExtensionAmount)
		if ok && !declared.Equals(*line.Total) {
			return false
		}
	}

	t := out.Totals
	if t == nil {
		return false
	}
	mt := ui.LegalMonetaryTotal
	pairs := []struct {
		declared Amount
		computed num.Amount
	}{
		{mt.LineExtensionAmount, t.Sum},
		{mt.TaxExclusiveAmount, t.Total},
		{mt.TaxInclusiveAmount, t.TotalWithTax},
	}
	if mt.PayableAmount != nil {
		pairs = append(pairs, struct {
			declared Amount
			computed num.Amount
		}{*mt.PayableAmount, goblPayableAmount(t)})
	}
	for _, p := range pairs {
		declared, ok := goblDeclaredAmount(p.declared)
		if ok && !declared.Equals(p.computed) {
			return false
		}
	}

	// The optional total fields are preserved on bypass, so they have to be
	// checked too: a document that disagrees on one of these would otherwise be
	// re-exported with our figure rather than the sender's. An absent total on
	// our side counts as zero.
	optional := []struct {
		declared *Amount
		computed *num.Amount
	}{
		{mt.AllowanceTotalAmount, t.Discount},
		{mt.ChargeTotalAmount, t.Charge},
		{mt.PrepaidAmount, t.Advances},
	}
	for _, p := range optional {
		if p.declared == nil {
			continue
		}
		declared, ok := goblDeclaredAmount(*p.declared)
		if !ok {
			continue
		}
		computed := num.AmountZero
		if p.computed != nil {
			computed = *p.computed
		}
		if !declared.Equals(computed) {
			return false
		}
	}

	if declared, ok := goblDeclaredTaxTotal(ui.TaxTotal); ok && !declared.Equals(t.Tax) {
		return false
	}
	return true
}

// goblPayableAmount returns what BT-115 is built from: Due once advances are
// deducted, Payable otherwise, matching the outbound mapping.
func goblPayableAmount(t *bill.Totals) num.Amount {
	if t.Due != nil {
		return *t.Due
	}
	return t.Payable
}

// applyDeclaredTotals records the sender's figures and tags the document so GOBL
// leaves them alone. Calculation stops under the tag, so every amount the
// document would otherwise derive is supplied here.
func (ui *Invoice) applyDeclaredTotals(out *bill.Invoice) error {
	// Align the sender's precision with the currency's.
	exp := out.Currency.Def().Zero().Exp()
	declared := func(a Amount) (num.Amount, bool) {
		v, ok := goblDeclaredAmount(a)
		if !ok {
			return v, false
		}
		return v.RescaleUp(exp), true
	}

	for i, docLine := range ui.lines() {
		if i >= len(out.Lines) {
			break
		}
		line := out.Lines[i]
		if line == nil {
			continue
		}
		v, ok := declared(docLine.LineExtensionAmount)
		if !ok {
			continue
		}
		// The sum stays calculated: no term carries the amount before
		// allowances and charges.
		line.Total = &v
	}

	t := out.Totals
	if t == nil {
		t = new(bill.Totals)
		out.Totals = t
	}
	mt := ui.LegalMonetaryTotal
	if v, ok := declared(mt.LineExtensionAmount); ok {
		t.Sum = v
	}
	if v, ok := declared(mt.TaxExclusiveAmount); ok {
		t.Total = v
	}
	taxInclusive, hasTaxInclusive := declared(mt.TaxInclusiveAmount)
	if hasTaxInclusive {
		t.TotalWithTax = taxInclusive
		t.Payable = taxInclusive
	}
	if mt.PrepaidAmount != nil {
		if v, ok := declared(*mt.PrepaidAmount); ok {
			t.Advances = &v
		}
	}
	if mt.PayableRoundingAmount != nil {
		if v, ok := declared(*mt.PayableRoundingAmount); ok {
			t.Rounding = &v
		}
	}
	// BT-115 is what remains to pay, which GOBL keeps in Due after advances.
	if mt.PayableAmount != nil {
		if v, ok := declared(*mt.PayableAmount); ok {
			if t.Advances != nil {
				t.Due = &v
			} else {
				t.Payable = v
			}
		}
	}
	if v, ok := goblDeclaredTaxTotal(ui.TaxTotal); ok {
		t.Tax = v.RescaleUp(exp)
		// BT-112 is mandatory but not every document carries it. Without one,
		// the calculated total would survive beside declared components it no
		// longer agrees with, so derive it instead.
		if !hasTaxInclusive {
			t.TotalWithTax = t.Total.MatchPrecision(t.Tax).Add(t.Tax)
			t.Payable = t.TotalWithTax
		}
		// The breakdown must come from the document too, or it contradicts
		// the total just set.
		t.Taxes = ui.goblDeclaredTaxBreakdown(exp)
	}
	if mt.AllowanceTotalAmount != nil {
		if v, ok := declared(*mt.AllowanceTotalAmount); ok {
			t.Discount = &v
		}
	}
	if mt.ChargeTotalAmount != nil {
		if v, ok := declared(*mt.ChargeTotalAmount); ok {
			t.Charge = &v
		}
	}

	// Derived values freeze once the tag is set, so re-derive the dues against
	// the declared payable.
	if out.Payment != nil {
		out.Payment.Terms.CalculateDues(out.Currency.Def().Zero(), t.Payable)
	}

	out.SetTags(tax.TagBypass)
	return out.Calculate()
}

// lines returns the document lines that produced a converted line, in the same
// order, so they pair index for index with out.Lines.
func (ui *Invoice) lines() []*InvoiceLine {
	if len(ui.CreditNoteLines) > 0 {
		return convertibleLines(ui.CreditNoteLines)
	}
	return convertibleLines(ui.InvoiceLines)
}

// goblDeclaredAmount parses a declared amount, reporting whether one was there.
func goblDeclaredAmount(a Amount) (num.Amount, bool) {
	if a.Value == "" {
		return num.AmountZero, false
	}
	v, err := num.AmountFromString(normalizeNumericString(a.Value))
	if err != nil {
		return num.AmountZero, false
	}
	return v, true
}

// goblDeclaredTaxTotal picks out BT-110, stated in the document's own currency.
func goblDeclaredTaxTotal(totals []TaxTotal) (num.Amount, bool) {
	for _, tt := range totals {
		if v, ok := goblDeclaredAmount(tt.TaxAmount); ok {
			return v, true
		}
	}
	return num.AmountZero, false
}

// goblDeclaredTaxBreakdown rebuilds BG-23 from the document's own tax subtotals,
// so it agrees with the tax total recorded alongside it.
func (ui *Invoice) goblDeclaredTaxBreakdown(exp uint32) *tax.Total {
	total := new(tax.Total)
	for _, tt := range ui.TaxTotal {
		for _, st := range tt.TaxSubtotal {
			if st.TaxCategory.TaxScheme == nil {
				continue
			}
			base, ok := goblDeclaredAmount(st.TaxableAmount)
			if !ok {
				continue
			}
			amount, ok := goblDeclaredAmount(st.TaxAmount)
			if !ok {
				continue
			}
			rate := &tax.RateTotal{
				Base:   base.RescaleUp(exp),
				Amount: amount.RescaleUp(exp),
			}
			ext := make(cbc.CodeMap)
			if st.TaxCategory.ID != nil {
				ext[untdid.ExtKeyTaxCategory] = cbc.Code(st.TaxCategory.ID.Value)
			}
			if st.TaxCategory.TaxExemptionReasonCode != nil {
				ext[cef.ExtKeyVATEX] = cbc.Code(*st.TaxCategory.TaxExemptionReasonCode)
			}
			if len(ext) > 0 {
				rate.Ext = tax.ExtensionsOf(ext)
			}
			if st.TaxCategory.Percent != nil {
				p, err := num.PercentageFromString(strings.TrimSuffix(normalizeNumericString(*st.TaxCategory.Percent), "%") + "%")
				if err == nil {
					rate.Percent = &p
				}
			}
			cat := goblCategoryTotal(total, cbc.Code(st.TaxCategory.TaxScheme.ID.Value))
			cat.Rates = append(cat.Rates, rate)
			cat.Amount = cat.Amount.MatchPrecision(rate.Amount).Add(rate.Amount)
			total.Sum = total.Sum.MatchPrecision(rate.Amount).Add(rate.Amount)
		}
	}
	if len(total.Categories) == 0 {
		return nil
	}
	return total
}

// applyPayableRounding carries BT-114 into the calculation. GOBL keeps a supplied
// rounding amount rather than deriving one.
func (ui *Invoice) applyPayableRounding(out *bill.Invoice) {
	mt := ui.LegalMonetaryTotal
	if mt.PayableRoundingAmount == nil {
		return
	}
	v, ok := goblDeclaredAmount(*mt.PayableRoundingAmount)
	if !ok || v.IsZero() {
		return
	}
	if out.Totals == nil {
		out.Totals = new(bill.Totals)
	}
	out.Totals.Rounding = &v
}

// goblCategoryTotal finds or adds a category's running total. Each declared
// subtotal is a rate within its category, not a category of its own.
func goblCategoryTotal(total *tax.Total, code cbc.Code) *tax.CategoryTotal {
	for _, c := range total.Categories {
		if c.Code == code {
			return c
		}
	}
	cat := &tax.CategoryTotal{Code: code}
	total.Categories = append(total.Categories, cat)
	return cat
}
