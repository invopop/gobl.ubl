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

// reconcileTotals aligns the converted document with the figures the sender
// declared, preferring the terms EN 16931 actually makes binding.
//
// The invoice line net amount (BT-131) is mandatory (BR-24) and is the only
// line figure the document totals are summed from (BR-CO-10, BR-CO-13). The
// item price base quantity (BT-149) is optional and appears in no rule at all,
// and GOBL has nowhere to store it: it can only be folded into the unit price.
// So when a line's declared amount and its base quantity disagree, the declared
// amount decides and the base quantity is dropped.
//
// A document that still does not reconcile is one whose arithmetic we cannot
// reproduce from the terms it carries. Rather than store our own figures in
// place of the sender's, it is tagged for bypass and the declared amounts are
// recorded verbatim.
func (ui *Invoice) reconcileTotals(out *bill.Invoice) error {
	ui.applyPayableRounding(out)

	if err := out.Calculate(); err != nil {
		return err
	}

	if swaps := ui.dropConflictingBaseQuantities(out); len(swaps) > 0 {
		if err := out.Calculate(); err != nil {
			return err
		}
		// A line that matches neither reading keeps the one the standard
		// describes; dropping its base quantity bought us nothing.
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

// priceSwap records a line whose base quantity was dropped, so the change can
// be undone if it did not reconcile the line.
type priceSwap struct {
	index int
	price num.Amount
}

// dropConflictingBaseQuantities removes the base quantity from any line whose
// total only matches the declared amount (BT-131) without it, and reports
// whether any line changed. Lines that match either way keep the base quantity,
// since dividing by it is the reading the standard describes.
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
		// The line disagrees with its declared amount. Retry it with the base
		// quantity left out, which is the only other reading of the price the
		// document supports.
		price, err := num.AmountFromString(normalizeNumericString(docLine.Price.PriceAmount.Value))
		if err != nil {
			continue
		}
		swaps = append(swaps, priceSwap{index: i, price: *line.Item.Price})
		line.Item.Price = &price
	}
	return swaps
}

// restoreBaseQuantities puts back the standard reading of the price on any
// swapped line that still does not match its declared amount, and reports
// whether anything was restored.
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

// declaredTotalsAgree reports whether every declared amount in the document is
// reproduced by the calculated one: the line amounts (BT-131), the sum of them
// (BT-106), the total without VAT (BT-109), the VAT total (BT-110), the total
// with VAT (BT-112) and the amount due for payment (BT-115).
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

	if declared, ok := goblDeclaredTaxTotal(ui.TaxTotal); ok && !declared.Equals(t.Tax) {
		return false
	}
	return true
}

// goblPayableAmount returns the total the amount due for payment (BT-115) is
// built from. GOBL keeps the amount still owed in Due once advances are
// deducted, and only falls back to Payable when there are none — the same
// choice the outbound mapping makes.
func goblPayableAmount(t *bill.Totals) num.Amount {
	if t.Due != nil {
		return *t.Due
	}
	return t.Payable
}

// applyDeclaredTotals records the sender's own figures and tags the document so
// GOBL leaves them alone. Under the bypass tag calculation stops before any
// total is derived, so every amount has to be supplied here.
func (ui *Invoice) applyDeclaredTotals(out *bill.Invoice) error {
	// Declared amounts carry whatever precision the sender wrote them with, so
	// align them with the currency before they become the document's own.
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
		// The sum stays as calculated: no business term carries the line amount
		// before allowances and charges, so the sender's own total is all we can
		// state with authority.
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
	if v, ok := declared(mt.TaxInclusiveAmount); ok {
		t.TotalWithTax = v
		t.Payable = v
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
	// BT-115 is what remains to be paid, which GOBL keeps in Due whenever
	// advances have been deducted.
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
		// The breakdown has to come from the document too. Leaving the
		// calculated one in place would contradict the total just set.
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

	// Anything GOBL derives from the totals is frozen once the tag is set, so
	// the payment dues have to be re-derived against the declared payable
	// rather than the one calculated before it was replaced.
	if out.Payment != nil {
		out.Payment.Terms.CalculateDues(out.Currency.Def().Zero(), t.Payable)
	}

	out.SetTags(tax.TagBypass)
	return out.Calculate()
}

// lines returns the document's lines, whichever element carries them.
func (ui *Invoice) lines() []InvoiceLine {
	if len(ui.CreditNoteLines) > 0 {
		return ui.CreditNoteLines
	}
	return ui.InvoiceLines
}

// goblDeclaredAmount parses a declared monetary amount, reporting whether one
// was present at all.
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

// goblDeclaredTaxTotal picks out the VAT total (BT-110), which is the tax total
// stated in the document's own currency.
func goblDeclaredTaxTotal(totals []TaxTotal) (num.Amount, bool) {
	for _, tt := range totals {
		if v, ok := goblDeclaredAmount(tt.TaxAmount); ok {
			return v, true
		}
	}
	return num.AmountZero, false
}

// goblDeclaredTaxBreakdown rebuilds the VAT breakdown (BG-23) from the
// document's own tax subtotals, so the categories agree with the tax total
// recorded alongside them.
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

// applyPayableRounding carries the rounding amount the sender applied to the
// amount due (BT-114) into the calculation. GOBL keeps a rounding amount that
// was supplied rather than deriving one, so setting it before calculating lets
// the amount due come out as the sender stated it.
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

// goblCategoryTotal finds the running total for a tax category, adding one if
// the category has not been seen yet. Each declared tax subtotal is a rate
// within its category, not a category of its own.
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
