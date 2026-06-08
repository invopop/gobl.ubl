package ubl

import (
	"strconv"

	oioubl "github.com/invopop/gobl.dk.oioubl/oioubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/cef"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	cur "github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

// TaxTotal represents a tax total
type TaxTotal struct {
	TaxAmount      Amount        `xml:"cbc:TaxAmount"`
	RoundingAmount *Amount       `xml:"cbc:RoundingAmount,omitempty"`
	TaxSubtotal    []TaxSubtotal `xml:"cac:TaxSubtotal"`
}

// TaxSubtotal represents a tax subtotal
type TaxSubtotal struct {
	TaxableAmount                Amount      `xml:"cbc:TaxableAmount,omitempty"`
	TaxAmount                    Amount      `xml:"cbc:TaxAmount"`
	TransactionCurrencyTaxAmount *Amount     `xml:"cbc:TransactionCurrencyTaxAmount,omitempty"`
	TaxCategory                  TaxCategory `xml:"cac:TaxCategory"`
}

// TaxCategory represents a tax category
type TaxCategory struct {
	ID                     *IDType    `xml:"cbc:ID,omitempty"`
	Percent                *string    `xml:"cbc:Percent,omitempty"`
	TaxExemptionReasonCode *string    `xml:"cbc:TaxExemptionReasonCode,omitempty"`
	TaxExemptionReason     *string    `xml:"cbc:TaxExemptionReason,omitempty"`
	TaxScheme              *TaxScheme `xml:"cac:TaxScheme,omitempty"`
}

// MonetaryTotal represents the monetary totals of the invoice
type MonetaryTotal struct {
	LineExtensionAmount   Amount  `xml:"cbc:LineExtensionAmount"`
	TaxExclusiveAmount    Amount  `xml:"cbc:TaxExclusiveAmount"`
	TaxInclusiveAmount    Amount  `xml:"cbc:TaxInclusiveAmount"`
	AllowanceTotalAmount  *Amount `xml:"cbc:AllowanceTotalAmount,omitempty"`
	ChargeTotalAmount     *Amount `xml:"cbc:ChargeTotalAmount,omitempty"`
	PrepaidAmount         *Amount `xml:"cbc:PrepaidAmount,omitempty"`
	PayableRoundingAmount *Amount `xml:"cbc:PayableRoundingAmount,omitempty"`
	PayableAmount         *Amount `xml:"cbc:PayableAmount,omitempty"`
}

// addOIOUBL21MonetaryTotal rebuilds LegalMonetaryTotal for OIOUBL 2.1, whose
// line LineExtensionAmount is gross (Price×Qty, F-INV348). The document
// LineExtensionAmount becomes the gross line sum, and line-level
// allowances/charges fold into the document Allowance/ChargeTotalAmount
// (F-INV129/F-INV130). Reconciles: gross − allowances + charges = net base.
func (ui *Invoice) addOIOUBL21MonetaryTotal(inv *bill.Invoice, ctx Context, currency string) {
	t := inv.Totals
	exp := t.Sum.Exp()
	grossSum := num.MakeAmount(0, exp)
	lineDiscounts := num.MakeAmount(0, exp)
	lineCharges := num.MakeAmount(0, exp)
	for _, l := range inv.Lines {
		if l.Sum != nil {
			grossSum = grossSum.Add(roundToCurrency(*l.Sum, currency))
		}
		for _, d := range l.Discounts {
			lineDiscounts = lineDiscounts.Add(d.Amount)
		}
		for _, c := range l.Charges {
			lineCharges = lineCharges.Add(c.Amount)
		}
		// Promote line allowances/charges to document-level AllowanceCharge
		// so they sum to Allowance/ChargeTotalAmount (F-INV129/F-INV130).
		for _, ac := range makeLineCharges(l.Charges, l.Discounts, currency, l.Sum, ctx, l.Taxes) {
			ui.AllowanceCharge = append(ui.AllowanceCharge, *ac)
		}
	}
	ui.LegalMonetaryTotal.LineExtensionAmount = Amount{Value: grossSum.String(), CurrencyID: &currency}
	allow := lineDiscounts
	if t.Discount != nil {
		allow = allow.Add(*t.Discount)
	}
	if !allow.IsZero() {
		ui.LegalMonetaryTotal.AllowanceTotalAmount = &Amount{Value: allow.String(), CurrencyID: &currency}
	}
	chg := lineCharges
	if t.Charge != nil {
		chg = chg.Add(*t.Charge)
	}
	if !chg.IsZero() {
		ui.LegalMonetaryTotal.ChargeTotalAmount = &Amount{Value: chg.String(), CurrencyID: &currency}
	}
	// OIOUBL rounds per line then sums (F-INV128/F-INV133); GOBL end-rounds,
	// which can differ by a cent on fractional quantities. Recompute the
	// inclusive/payable totals from the rounded components so they reconcile.
	incl := grossSum.Add(t.Tax).Add(chg).Subtract(allow)
	if t.Rounding != nil {
		incl = incl.Add(*t.Rounding)
	}
	ui.LegalMonetaryTotal.TaxInclusiveAmount = Amount{Value: incl.String(), CurrencyID: &currency}
	pay := incl
	if t.Advances != nil {
		pay = pay.Subtract(*t.Advances)
	}
	ui.LegalMonetaryTotal.PayableAmount = &Amount{Value: pay.String(), CurrencyID: &currency}
}

// addOIOUBL21PrepaidPayments emits a cac:PrepaidPayment per GOBL advance. OIOUBL
// requires the PaidAmount elements to sum to LegalMonetaryTotal/PrepaidAmount
// (F-INV131); Peppol and EN 16931 carry the prepaid amount in the total only,
// so this is OIOUBL-specific.
func (ui *Invoice) addOIOUBL21PrepaidPayments(inv *bill.Invoice, currency string) {
	if inv.Payment == nil {
		return
	}
	for i, adv := range inv.Payment.Advances {
		if adv == nil {
			continue
		}
		pp := PrepaidPayment{
			ID:         strconv.Itoa(i + 1),
			PaidAmount: &Amount{Value: adv.Amount.String(), CurrencyID: &currency},
		}
		if adv.Date != nil {
			d := formatDate(*adv.Date)
			pp.ReceivedDate = &d
		}
		if adv.Ref != "" {
			ref := adv.Ref
			pp.InstructionID = &ref
		}
		ui.PrepaidPayment = append(ui.PrepaidPayment, pp)
	}
}

func (ui *Invoice) addTotals(inv *bill.Invoice, ctx Context) {
	if inv == nil || inv.Totals == nil {
		return
	}
	t := inv.Totals

	currency := inv.Currency.String()
	rCurrency := inv.RegimeDef().Currency.String()

	ui.LegalMonetaryTotal = MonetaryTotal{
		LineExtensionAmount: Amount{Value: t.Sum.String(), CurrencyID: &currency},
		TaxExclusiveAmount:  Amount{Value: t.Total.String(), CurrencyID: &currency},
		TaxInclusiveAmount:  Amount{Value: t.TotalWithTax.String(), CurrencyID: &currency},
		PayableAmount:       &Amount{Value: t.Payable.String(), CurrencyID: &currency},
	}

	if t.Discount != nil {
		ui.LegalMonetaryTotal.AllowanceTotalAmount = &Amount{Value: t.Discount.String(), CurrencyID: &currency}
	}
	if t.Charge != nil {
		ui.LegalMonetaryTotal.ChargeTotalAmount = &Amount{Value: t.Charge.String(), CurrencyID: &currency}
	}

	if ctx.Is(ContextOIOUBL21) {
		ui.addOIOUBL21MonetaryTotal(inv, ctx, currency)
		ui.addOIOUBL21PrepaidPayments(inv, currency)
	}
	if t.Rounding != nil {
		ui.LegalMonetaryTotal.PayableRoundingAmount = &Amount{Value: t.Rounding.String(), CurrencyID: &currency}
	}
	if t.Advances != nil {
		ui.LegalMonetaryTotal.PrepaidAmount = &Amount{Value: t.Advances.String(), CurrencyID: &currency}
	}
	if t.Due != nil && !ctx.Is(ContextOIOUBL21) {
		ui.LegalMonetaryTotal.PayableAmount = &Amount{Value: t.Due.String(), CurrencyID: &currency}
	}

	ui.TaxTotal = []TaxTotal{
		{
			TaxAmount: Amount{Value: t.Tax.String(), CurrencyID: &currency},
		},
	}

	var accRate *cur.ExchangeRate
	if inv.Currency != inv.RegimeDef().Currency {
		accRate = cur.MatchExchangeRate(inv.ExchangeRates, inv.Currency, inv.RegimeDef().Currency)
	}

	// OIOUBL states the accounting-currency tax per subtotal (see below) rather
	// than in a second TaxTotal, so its accounting total is built there instead.
	if !ctx.Is(ContextOIOUBL21) {
		// BT-111
		if inv.Currency.String() != rCurrency {
			if accRate != nil {
				ui.TaxTotal = append(ui.TaxTotal, TaxTotal{
					TaxAmount: Amount{
						Value:      accRate.Convert(t.Tax).String(),
						CurrencyID: &rCurrency,
					},
				})
			}
		} else if ctx.Is(ContextZATCA) {
			// BR-KSA-EN16931-09
			ui.TaxTotal = append(ui.TaxTotal, TaxTotal{
				TaxAmount: Amount{Value: t.Tax.String(), CurrencyID: &currency},
			})
		}
	}

	if t.Taxes != nil && len(t.Taxes.Categories) > 0 {
		for _, cat := range t.Taxes.Categories {
			for _, r := range cat.Rates {
				subtotal := TaxSubtotal{
					TaxAmount: Amount{Value: r.Amount.String(), CurrencyID: &currency},
				}
				if r.Base != (num.Amount{}) {
					subtotal.TaxableAmount = Amount{Value: r.Base.String(), CurrencyID: &currency}
				}
				// F-INV018 / F-CRN013: when DocumentCurrencyCode differs from the
				// tax currency, OIOUBL carries the tax in the tax currency here.
				// F-INV339 fixes its currencyID to the TaxCurrencyCode.
				if ctx.Is(ContextOIOUBL21) && accRate != nil {
					subtotal.TransactionCurrencyTaxAmount = &Amount{
						Value:      accRate.Convert(r.Amount).String(),
						CurrencyID: &rCurrency,
					}
				}
				taxCat := TaxCategory{}

				if k := oioubl21TaxCategoryID(r.Ext); k != "" {
					taxCat.ID = &IDType{Value: k}
				}
				if v := r.Ext.Get(cef.ExtKeyVATEX).String(); v != "" {
					taxCat.TaxExemptionReasonCode = &v
				}

				if inv.Tax != nil {
					if note := findTaxNote(inv.Tax.Notes, cat.Code, r); note != nil {
						taxCat.TaxExemptionReason = &note.Text
					}
				}

				// Set percent: required unless category is "O" (outside scope)
				if r.Percent != nil {
					p := r.Percent.StringWithoutSymbol()
					taxCat.Percent = &p
				} else if taxCat.ID == nil || taxCat.ID.Value != "O" {
					// Default to 0% when not outside scope
					p := "0"
					taxCat.Percent = &p
				}

				if cat.Code != cbc.CodeEmpty {
					taxCat.TaxScheme = &TaxScheme{ID: IDType{Value: cat.Code.String()}}
				}
				subtotal.TaxCategory = taxCat
				ui.TaxTotal[0].TaxSubtotal = append(ui.TaxTotal[0].TaxSubtotal, subtotal)
			}
		}
	}
}

// taxCategoryInfo holds tax category information from TaxTotal
type taxCategoryInfo struct {
	exemptionReasonCode string
}

// buildTaxCategoryMap builds a map of tax category information from TaxTotal.
func (ui *Invoice) buildTaxCategoryMap() map[string]*taxCategoryInfo {
	categoryMap := make(map[string]*taxCategoryInfo)

	for _, taxTotal := range ui.TaxTotal {
		for _, subtotal := range taxTotal.TaxSubtotal {
			if subtotal.TaxCategory.ID != nil && subtotal.TaxCategory.TaxScheme != nil {
				key := buildTaxCategoryKey(subtotal.TaxCategory.TaxScheme.ID.Value, subtotal.TaxCategory.ID.Value, subtotal.TaxCategory.Percent)
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

// goblAddTaxNotes extracts tax notes from UBL TaxTotal subtotals and adds them
// to the invoice's Tax.Notes.
func (ui *Invoice) goblAddTaxNotes(inv *bill.Invoice) {
	for _, tt := range ui.TaxTotal {
		for _, st := range tt.TaxSubtotal {
			tc := st.TaxCategory
			if tc.TaxExemptionReason == nil || tc.ID == nil || tc.TaxScheme == nil {
				continue
			}
			note := &tax.Note{
				Category: goblTaxSchemeCategory(tc.TaxScheme.ID.Value),
				Text:     cleanString(*tc.TaxExemptionReason),
				Ext:      tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyTaxCategory: goblTaxCategoryCode(tc.ID.Value)}),
			}
			inv.Tax = inv.Tax.MergeNotes(note)
		}
	}
}

// findTaxNote finds a tax note that matches the given category code and rate total
// by comparing category and the UNTDID tax category extension.
func findTaxNote(notes []*tax.Note, catCode cbc.Code, rate *tax.RateTotal) *tax.Note {
	for _, n := range notes {
		if n.Category != catCode {
			continue
		}
		if nc := n.Ext.Get(untdid.ExtKeyTaxCategory); nc != cbc.CodeEmpty && nc == rate.Ext.Get(untdid.ExtKeyTaxCategory) {
			return n
		}
	}
	return nil
}

// goblExchangeRates derives the exchange rate between DocumentCurrencyCode and
// TaxCurrencyCode. EN16931/Peppol carry the accounting-currency tax in a second
// TaxTotal; OIOUBL carries it per subtotal as TransactionCurrencyTaxAmount.
func goblExchangeRates(docCurrency, taxCurrency cur.Code, taxTotals []TaxTotal) []*cur.ExchangeRate {
	if len(taxTotals) == 0 {
		return nil
	}

	docAmount, err := num.AmountFromString(normalizeNumericString(taxTotals[0].TaxAmount.Value))
	if err != nil || docAmount.IsZero() {
		return nil
	}

	taxAmount, ok := taxCurrencyTaxAmount(taxTotals)
	if !ok {
		return nil
	}

	rate := taxAmount.Divide(docAmount)

	return []*cur.ExchangeRate{
		{
			From:   docCurrency,
			To:     taxCurrency,
			Amount: rate,
		},
	}
}

// taxCurrencyTaxAmount returns the total tax expressed in the tax currency,
// reading a second TaxTotal block (EN16931/Peppol) or, failing that, summing
// the per-subtotal TransactionCurrencyTaxAmount of the first TaxTotal (OIOUBL).
func taxCurrencyTaxAmount(taxTotals []TaxTotal) (num.Amount, bool) {
	if len(taxTotals) >= 2 {
		a, err := num.AmountFromString(normalizeNumericString(taxTotals[1].TaxAmount.Value))
		if err != nil {
			return num.Amount{}, false
		}
		return a, true
	}

	var total num.Amount
	found := false
	for _, st := range taxTotals[0].TaxSubtotal {
		if st.TransactionCurrencyTaxAmount == nil {
			continue
		}
		a, err := num.AmountFromString(normalizeNumericString(st.TransactionCurrencyTaxAmount.Value))
		if err != nil {
			return num.Amount{}, false
		}
		if found {
			total = total.Add(a)
		} else {
			total, found = a, true
		}
	}
	return total, found
}

// OIOUBL taxcategoryid-1.1 wire values (sourced from the dk-oioubl addon) and
// the serialization-only taxschemeid-1.2 VAT (Moms) code.
const (
	oioubl21TaxCategoryStandardRated = string(oioubl.ExtValueTaxCategoryStandardRated)
	oioubl21TaxCategoryZeroRated     = string(oioubl.ExtValueTaxCategoryZeroRated)
	oioubl21TaxCategoryReverseCharge = string(oioubl.ExtValueTaxCategoryReverseCharge)

	oioubl21TaxSchemeVATCode = "63" // taxschemeid-1.2 VAT (Moms)
)

// oioubl21TaxCategoryID returns the value to emit as cac:TaxCategory/cbc:ID. The
// dk-oioubl addon (required by ContextOIOUBL21) precomputes the OIOUBL
// taxcategoryid-1.1 code in the dk-oioubl-tax-category extension; other profiles
// fall back to the UNTDID category, which they use directly.
func oioubl21TaxCategoryID(ext tax.Extensions) string {
	if c := ext.Get(oioubl.ExtKeyTaxCategory); c != "" {
		return c.String()
	}
	return ext.Get(untdid.ExtKeyTaxCategory).String()
}

// applyOIOUBL21Totals stamps the taxcategoryid attributes on the document-level
// tax subtotals and allowance/charges, and re-interprets TaxExclusiveAmount as
// the total tax amount (F-INV127), not the pre-tax sum as in generic UBL. It
// runs after the whole document is assembled because the document allowance set
// is only complete once promoted line allowances have been added.
func (ui *Invoice) applyOIOUBL21Totals() {
	for i := range ui.TaxTotal {
		for j := range ui.TaxTotal[i].TaxSubtotal {
			applyOIOUBL21TaxCategory(&ui.TaxTotal[i].TaxSubtotal[j].TaxCategory)
		}
	}
	for i := range ui.AllowanceCharge {
		for _, tc := range ui.AllowanceCharge[i].TaxCategory {
			applyOIOUBL21TaxCategory(tc)
		}
	}
	if len(ui.TaxTotal) > 0 {
		ui.LegalMonetaryTotal.TaxExclusiveAmount = ui.TaxTotal[0].TaxAmount
	}
}

// oioubl21CategoryID stamps the taxcategoryid-1.1 codelist attributes onto a
// tax-category cbc:ID, defaulting an absent category to StandardRated.
func oioubl21CategoryID(id *IDType) *IDType {
	if id == nil {
		id = &IDType{Value: oioubl21TaxCategoryStandardRated}
	}
	schemeID := "urn:oioubl:id:taxcategoryid-1.1"
	schemeAgencyID := "320"
	id.SchemeID = &schemeID
	id.SchemeAgencyID = &schemeAgencyID
	return id
}

func applyOIOUBL21TaxCategory(tc *TaxCategory) {
	if tc == nil {
		return
	}
	tc.ID = oioubl21CategoryID(tc.ID)
	applyOIOUBL21TaxScheme(tc.TaxScheme)
}

func applyOIOUBL21ClassifiedTaxCategory(tc *ClassifiedTaxCategory) {
	if tc == nil {
		return
	}
	tc.ID = oioubl21CategoryID(tc.ID)
	applyOIOUBL21TaxScheme(tc.TaxScheme)
}

func applyOIOUBL21TaxScheme(ts *TaxScheme) {
	if ts == nil {
		return
	}
	schemeID := "urn:oioubl:id:taxschemeid-1.2"
	schemeAgencyID := "320"
	ts.ID = IDType{
		SchemeID:       &schemeID,
		SchemeAgencyID: &schemeAgencyID,
		Value:          oioubl21TaxSchemeVATCode,
	}
	name := "Moms"
	ts.Name = &name
}
