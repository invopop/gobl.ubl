package ubl

import (
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
)

// OIOUBL emits a non-VAT excise duty as a cac:TaxTotal/cac:TaxSubtotal carrying
// the constant taxcategoryid-1.1 "Excise" category and the duty-type taxschemeid
// code, rather than as a cac:AllowanceCharge. GOBL models the duty as a VAT-rated
// charge whose Key is the taxschemeid duty code; VAT therefore already lands on
// the duty-inclusive base (the charge folds into the line total).
const oioubl21TaxCategoryExcise = "Excise"

// oioubl21Excise is a duty charge resolved into the values OIOUBL needs: the
// taxschemeid duty-type code, the scheme name (the charge reason) and the amount.
type oioubl21Excise struct {
	scheme string
	name   string
	amount num.Amount
}

// chargeExciseScheme returns the OIOUBL taxschemeid duty-type code a charge
// carries in its Key, or "" for an ordinary charge. An excise duty is keyed with
// its numeric taxschemeid code (e.g. "16"); GOBL's own charge keys are alphabetic
// slugs (stamp-duty, handling, …), so an all-digit key marks the duty. A
// single-digit code is stored zero-padded ("9" → "09"), since cbc.Key requires a
// two-character minimum for digits; the leading zero is stripped to recover the
// wire value.
func chargeExciseScheme(key cbc.Key) string {
	s := key.String()
	if s == "" || !isAllDigits(s) {
		return ""
	}
	if code := strings.TrimLeft(s, "0"); code != "" {
		return code
	}
	return "0"
}

// exciseSchemeKey builds the charge Key for an OIOUBL taxschemeid duty code,
// zero-padding a single digit so it is a valid cbc.Key.
func exciseSchemeKey(code string) cbc.Key {
	if len(code) == 1 {
		return cbc.Key("0" + code)
	}
	return cbc.Key(code)
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// collectOIOUBL21Excise gathers every excise duty across the document- and
// line-level charges. Discounts are never excise (a duty is always a charge).
func collectOIOUBL21Excise(inv *bill.Invoice, currency string) []oioubl21Excise {
	var out []oioubl21Excise
	for _, ch := range inv.Charges {
		if s := chargeExciseScheme(ch.Key); s != "" {
			out = append(out, oioubl21Excise{scheme: s, name: ch.Reason, amount: roundToCurrency(ch.Amount, currency)})
		}
	}
	for _, l := range inv.Lines {
		out = append(out, collectLineExcise(l, currency)...)
	}
	return out
}

// collectLineExcise gathers the excise duties on a single line, used to mirror
// them as a line-level cac:TaxTotal so the wire records which line each duty
// belongs to (the document-level totals drive the monetary reconciliation).
func collectLineExcise(line *bill.Line, currency string) []oioubl21Excise {
	var out []oioubl21Excise
	for _, ch := range line.Charges {
		if s := chargeExciseScheme(ch.Key); s != "" {
			out = append(out, oioubl21Excise{scheme: s, name: ch.Reason, amount: roundToCurrency(ch.Amount, currency)})
		}
	}
	return out
}

// makeOIOUBL21ExciseTaxTotals builds one cac:TaxTotal per duty, mirroring the
// official ERST samples: category "Excise", the duty-type code in the scheme, the
// duty name from the charge reason, and TaxTypeCode derived from the amount
// (StandardRated when positive, ZeroRated when zero — the duty's own rate, not a
// VAT statement). TaxableAmount equals the duty amount, as the duty is levied
// outright rather than as a percentage of a base.
func makeOIOUBL21ExciseTaxTotals(excises []oioubl21Excise, currency string) []TaxTotal {
	var totals []TaxTotal
	for _, e := range excises {
		amt := Amount{Value: e.amount.String(), CurrencyID: &currency}
		typeCode := oioubl21TaxCategoryStandardRated
		if e.amount.IsZero() {
			typeCode = oioubl21TaxCategoryZeroRated
		}
		scheme := &TaxScheme{
			ID:          IDType{SchemeID: ptr(oioublSchemeTaxScheme), SchemeAgencyID: ptr(oioublAgencyID), Value: e.scheme},
			TaxTypeCode: &IDType{ListAgencyID: ptr(oioublAgencyID), ListID: ptr(oioublListTaxType), Value: typeCode},
		}
		if e.name != "" {
			scheme.Name = ptr(e.name)
		}
		totals = append(totals, TaxTotal{
			TaxAmount: amt,
			TaxSubtotal: []TaxSubtotal{{
				TaxableAmount: amt,
				TaxAmount:     amt,
				TaxCategory: TaxCategory{
					ID:        oioubl21CategoryID(&IDType{Value: oioubl21TaxCategoryExcise}),
					TaxScheme: scheme,
				},
			}},
		})
	}
	return totals
}

// exciseLineChargesFromTaxTotals reconstructs a bill.LineCharge for every
// cac:TaxTotal/Excise subtotal in the given totals, the inverse of
// makeOIOUBL21ExciseTaxTotals: the taxschemeid duty code becomes the charge Key,
// reason from the scheme name, amount from the subtotal. Non-excise (VAT)
// subtotals are ignored.
func exciseLineChargesFromTaxTotals(totals []TaxTotal) ([]*bill.LineCharge, error) {
	var charges []*bill.LineCharge
	for _, tt := range totals {
		for i := range tt.TaxSubtotal {
			st := &tt.TaxSubtotal[i]
			if st.TaxCategory.ID == nil || st.TaxCategory.ID.Value != oioubl21TaxCategoryExcise {
				continue
			}
			if st.TaxCategory.TaxScheme == nil {
				continue
			}
			amount, err := num.AmountFromString(normalizeNumericString(st.TaxAmount.Value))
			if err != nil {
				return nil, err
			}
			ch := &bill.LineCharge{
				Key:    exciseSchemeKey(st.TaxCategory.TaxScheme.ID.Value),
				Amount: amount,
			}
			if st.TaxCategory.TaxScheme.Name != nil {
				ch.Reason = *st.TaxCategory.TaxScheme.Name
			}
			charges = append(charges, ch)
		}
	}
	return charges, nil
}

// goblAddExciseCharges reconstructs the GOBL charges OIOUBL carried as
// cac:TaxTotal/Excise subtotals. An excise duty is a bill.LineCharge (it must
// ride a line: a document-level charge cannot reconcile OIOUBL's monetary
// totals, the VAT-base bump that keeps F-LIB402/F-INV133 satisfied living only
// in the line path). We emit the duty as both a line-level and a document-level
// TaxTotal, so the line-level blocks (parsed per line in goblConvertLine)
// preserve which line each duty belongs to. This handles the document-level
// totals as a fallback: only when no line carried a line-level excise block (a
// sender that emitted excise at document level alone), the duties are attached
// to the first line, since the wire gives no other linkage.
func (ui *Invoice) goblAddExciseCharges(out *bill.Invoice, ctx Context) error {
	if !ctx.Is(ContextOIOUBL21) || len(out.Lines) == 0 {
		return nil
	}
	for _, l := range out.Lines {
		for _, ch := range l.Charges {
			if chargeExciseScheme(ch.Key) != "" {
				// A line already carried its excise (line-level blocks were parsed);
				// the document-level totals are their mirror, so don't re-add them.
				return nil
			}
		}
	}
	charges, err := exciseLineChargesFromTaxTotals(ui.TaxTotal)
	if err != nil {
		return err
	}
	out.Lines[0].Charges = append(out.Lines[0].Charges, charges...)
	return nil
}
