package ubl

import (
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
	TaxableAmount Amount      `xml:"cbc:TaxableAmount,omitempty"`
	TaxAmount     Amount      `xml:"cbc:TaxAmount"`
	TaxCategory   TaxCategory `xml:"cac:TaxCategory"`
}

// TaxCategory represents a tax category
type TaxCategory struct {
	ID                     *string    `xml:"cbc:ID,omitempty"`
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
	if t.Rounding != nil {
		ui.LegalMonetaryTotal.PayableRoundingAmount = &Amount{Value: t.Rounding.String(), CurrencyID: &currency}
	}
	if t.Advances != nil {
		ui.LegalMonetaryTotal.PrepaidAmount = &Amount{Value: t.Advances.String(), CurrencyID: &currency}
	}
	if t.Due != nil {
		ui.LegalMonetaryTotal.PayableAmount = &Amount{Value: t.Due.String(), CurrencyID: &currency}
	}

	ui.TaxTotal = []TaxTotal{
		{
			TaxAmount: Amount{Value: t.Tax.String(), CurrencyID: &currency},
		},
	}

	// BT-111
	if inv.Currency.String() != rCurrency {
		if rate := cur.MatchExchangeRate(inv.ExchangeRates, inv.Currency, inv.RegimeDef().Currency); rate != nil {
			taxInAccCurrency := rate.Convert(t.Tax)
			accTaxTotal := TaxTotal{
				TaxAmount: Amount{
					Value:      taxInAccCurrency.String(),
					CurrencyID: &rCurrency,
				},
			}
			ui.TaxTotal = append(ui.TaxTotal, accTaxTotal)
		}
	} else if ctx.Is(ContextZATCA) {
		// BR-KSA-EN16931-09
		ui.TaxTotal = append(ui.TaxTotal, TaxTotal{
			TaxAmount: Amount{Value: t.Tax.String(), CurrencyID: &currency},
		})
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
				taxCat := TaxCategory{}

				if k := r.Ext.Get(untdid.ExtKeyTaxCategory).String(); k != "" {
					taxCat.ID = &k
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
				} else if taxCat.ID == nil || *taxCat.ID != "O" {
					// Default to 0% when not outside scope
					p := "0"
					taxCat.Percent = &p
				}

				if cat.Code != cbc.CodeEmpty {
					taxCat.TaxScheme = &TaxScheme{ID: cat.Code.String()}
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
				schemeID := subtotal.TaxCategory.TaxScheme.ID
				categoryID := *subtotal.TaxCategory.ID
				key := buildTaxCategoryKey(schemeID, categoryID, subtotal.TaxCategory.Percent)
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
				Category: cbc.Code(tc.TaxScheme.ID),
				Text:     cleanString(*tc.TaxExemptionReason),
				Ext:      tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyTaxCategory: cbc.Code(*tc.ID)}),
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

// goblExchangeRates derives the exchange rate from two TaxTotal blocks
// when DocumentCurrencyCode differs from TaxCurrencyCode.
func goblExchangeRates(docCurrency, taxCurrency cur.Code, taxTotals []TaxTotal) []*cur.ExchangeRate {
	if len(taxTotals) < 2 {
		return nil
	}

	docAmount, err := num.AmountFromString(normalizeNumericString(taxTotals[0].TaxAmount.Value))
	if err != nil || docAmount.IsZero() {
		return nil
	}
	taxAmount, err := num.AmountFromString(normalizeNumericString(taxTotals[1].TaxAmount.Value))
	if err != nil {
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
