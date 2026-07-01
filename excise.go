package ubl

import (
	oioubl "github.com/invopop/gobl.dk.oioubl/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

// OIOUBL emits a non-VAT excise duty as a cac:TaxTotal/cac:TaxSubtotal carrying
// the constant taxcategoryid-1.1 "Excise" category and the duty-type taxschemeid
// code, rather than as a cac:AllowanceCharge. GOBL models the duty as a VAT-rated
// charge tagged with the dk-oioubl-tax-scheme extension; VAT therefore already
// lands on the duty-inclusive base (the charge folds into the line total).
const oioubl21TaxCategoryExcise = "Excise"

const oioubl21TaxTypeListID = "urn:oioubl:codelist:taxtypecode-1.1"

// oioubl21Excise is a duty charge resolved into the values OIOUBL needs: the
// taxschemeid duty-type code, the scheme name (the charge reason) and the amount.
type oioubl21Excise struct {
	scheme string
	name   string
	amount num.Amount
}

// chargeExciseScheme returns the OIOUBL duty-type code a charge carries via the
// dk-oioubl-tax-scheme extension, or "" if the charge is an ordinary charge.
func chargeExciseScheme(ext tax.Extensions) string {
	return ext.Get(oioubl.ExtKeyTaxScheme).String()
}

// collectOIOUBL21Excise gathers every excise duty across the document- and
// line-level charges. Discounts are never excise (a duty is always a charge).
func collectOIOUBL21Excise(inv *bill.Invoice, currency string) []oioubl21Excise {
	var out []oioubl21Excise
	for _, ch := range inv.Charges {
		if s := chargeExciseScheme(ch.Ext); s != "" {
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
		if s := chargeExciseScheme(ch.Ext); s != "" {
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
	agency := "320"
	schemeIDAttr := "urn:oioubl:id:taxschemeid-1.1"
	listID := oioubl21TaxTypeListID
	for _, e := range excises {
		amt := Amount{Value: e.amount.String(), CurrencyID: &currency}
		typeCode := oioubl21TaxCategoryStandardRated
		if e.amount.IsZero() {
			typeCode = oioubl21TaxCategoryZeroRated
		}
		scheme := &TaxScheme{
			ID:          IDType{SchemeID: &schemeIDAttr, SchemeAgencyID: &agency, Value: e.scheme},
			TaxTypeCode: &IDType{ListAgencyID: &agency, ListID: &listID, Value: typeCode},
		}
		if e.name != "" {
			name := e.name
			scheme.Name = &name
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
// makeOIOUBL21ExciseTaxTotals: ext dk-oioubl-tax-scheme from the TaxScheme code,
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
				Amount: amount,
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					oioubl.ExtKeyTaxScheme: cbc.Code(st.TaxCategory.TaxScheme.ID.Value),
				}),
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
			if chargeExciseScheme(ch.Ext) != "" {
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
