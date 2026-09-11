package ubl_test

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCurrencyRounding covers the EN 16931 BR-DEC-* rules and UBL-DT-01, which
// cap nearly every monetary amount at the currency's precision. GOBL's
// `precise` rounding rule — the default for every regime but Greece, and what
// RemoveIncludedTaxes switches a document to since GOBL v0.505 — keeps line
// amounts at a higher precision, so the converter has to round the document
// before writing it out.
func TestCurrencyRounding(t *testing.T) {
	t.Run("line total with more decimals than the currency", func(t *testing.T) {
		env := loadTestEnvelope(t, "peppol/invoice-complete.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// 3 × 10.555 = 31.665, one decimal more than EUR allows.
		price := num.MakeAmount(10555, 3)
		inv.Lines = inv.Lines[:1]
		inv.Lines[0].Quantity = num.MakeAmount(3, 0)
		inv.Lines[0].Item.Price = &price
		inv.Lines[0].Discounts = nil
		inv.Lines[0].Charges = nil
		require.NoError(t, env.Calculate())
		require.Equal(t, "31.665", inv.Lines[0].Total.String())

		doc, err := ubl.ConvertInvoice(env, ubl.WithContext(ubl.ContextPeppol))
		require.NoError(t, err)

		// BT-131 / BR-DEC-23
		assert.Equal(t, "31.67", doc.InvoiceLines[0].LineExtensionAmount.Value)
		// BT-106: the document total agrees with the rounded line (BR-CO-10).
		assert.Equal(t, "31.67", doc.LegalMonetaryTotal.LineExtensionAmount.Value)
		// BT-146: the unit price keeps its extra decimals, which it is allowed.
		assert.Equal(t, "10.555", doc.InvoiceLines[0].Price.PriceAmount.Value)
	})

	t.Run("prices include tax", func(t *testing.T) {
		doc := testInvoiceFrom(t, "peppol/invoice-prices-include-vat.json")

		require.Len(t, doc.InvoiceLines, 2)
		assert.Equal(t, "6.73", doc.InvoiceLines[0].LineExtensionAmount.Value)
		assert.Equal(t, "50.88", doc.InvoiceLines[1].LineExtensionAmount.Value)

		// The rounded lines add up to the document total (BR-CO-10), and the
		// cent the rounding gained is declared as BT-114 so that the payable
		// amount still matches the tax-inclusive original.
		assert.Equal(t, "57.61", doc.LegalMonetaryTotal.LineExtensionAmount.Value)
		require.NotNil(t, doc.LegalMonetaryTotal.PayableRoundingAmount)
		assert.Equal(t, "-0.01", doc.LegalMonetaryTotal.PayableRoundingAmount.Value)
		require.NotNil(t, doc.LegalMonetaryTotal.PayableAmount)
		assert.Equal(t, "69.70", doc.LegalMonetaryTotal.PayableAmount.Value)
	})

	t.Run("every constrained amount fits the currency", func(t *testing.T) {
		for _, name := range []string{
			"peppol/invoice-prices-include-vat.json",
			"peppol/invoice-complete.json",
			"peppol/peppol-1-advance.json",
			"peppol/invoice-minimal.json",
		} {
			t.Run(name, func(t *testing.T) {
				assertAmountsFitCurrency(t, testInvoiceFrom(t, name))
			})
		}
	})
}

// assertAmountsFitCurrency checks every amount the BR-DEC-* rules constrain to
// two decimals. Unit prices (BT-146, BT-148) are exempt and so not included.
func assertAmountsFitCurrency(t *testing.T, doc *ubl.Invoice) {
	t.Helper()

	check := func(field string, a *ubl.Amount) {
		if a == nil {
			return
		}
		if _, frac, found := strings.Cut(a.Value, "."); found {
			assert.LessOrEqual(t, len(frac), 2, "%s has more than 2 decimals: %s", field, a.Value)
		}
	}

	m := doc.LegalMonetaryTotal
	check("BT-106 LineExtensionAmount", &m.LineExtensionAmount)
	check("BT-109 TaxExclusiveAmount", &m.TaxExclusiveAmount)
	check("BT-112 TaxInclusiveAmount", &m.TaxInclusiveAmount)
	check("BT-107 AllowanceTotalAmount", m.AllowanceTotalAmount)
	check("BT-108 ChargeTotalAmount", m.ChargeTotalAmount)
	check("BT-113 PrepaidAmount", m.PrepaidAmount)
	check("BT-114 PayableRoundingAmount", m.PayableRoundingAmount)
	check("BT-115 PayableAmount", m.PayableAmount)

	for _, tt := range doc.TaxTotal {
		check("BT-110 TaxAmount", &tt.TaxAmount)
		for _, st := range tt.TaxSubtotal {
			check("BT-116 TaxableAmount", &st.TaxableAmount)
			check("BT-117 TaxAmount", &st.TaxAmount)
		}
	}

	for _, ac := range doc.AllowanceCharge {
		check("BT-92/99 Amount", &ac.Amount)
		check("BT-93/100 BaseAmount", ac.BaseAmount)
	}

	lines := doc.InvoiceLines
	if len(lines) == 0 {
		lines = doc.CreditNoteLines
	}
	for _, l := range lines {
		check("BT-131 LineExtensionAmount", &l.LineExtensionAmount)
		for _, ac := range l.AllowanceCharge {
			check("BT-136/141 Amount", &ac.Amount)
			check("BT-137/142 BaseAmount", ac.BaseAmount)
		}
	}
}

// TestTotalsArithmetic checks the BR-CO identities the EN 16931 schematron
// enforces on every generated document. They are what makes rounding a
// document-wide decision rather than a per-field one: once a line net amount
// (BT-131) is rounded to the currency, the sums that derive from it have no
// freedom left, so the converter cannot round amounts independently on the way
// out without breaking BR-CO-10.
func TestTotalsArithmetic(t *testing.T) {
	contexts := map[string]ubl.Context{
		"en16931":            ubl.ContextEN16931,
		"peppol":             ubl.ContextPeppol,
		"peppol-self-billed": ubl.ContextPeppolSelfBilled,
		"xrechnung":          ubl.ContextXRechnung,
		"france-cius":        ubl.ContextPeppolFranceCIUS,
		"france-extended":    ubl.ContextPeppolFranceExtended,
		"zatca":              ubl.ContextZATCA,
	}

	for dir, ctx := range contexts {
		examples, err := filepath.Glob(filepath.Join(getConvertPath(), dir, jsonPattern))
		require.NoError(t, err)

		for _, example := range examples {
			name := filepath.Base(example)
			t.Run(dir+"/"+name, func(t *testing.T) {
				doc, err := testInvoiceFromContext(filepath.Join(dir, name), ctx)
				require.NoError(t, err)
				assertTotalsAddUp(t, doc)
			})
		}
	}
}

// TestTotalsArithmeticManyLines covers the case the rounding drift grows with:
// many lines whose tax-exclusive amount is not exactly representable. The
// payable amount must survive untouched however many lines there are, with the
// difference declared as the payable rounding amount (BT-114).
func TestTotalsArithmeticManyLines(t *testing.T) {
	for _, lines := range []int{1, 10, 50, 200} {
		t.Run(strconv.Itoa(lines)+" lines", func(t *testing.T) {
			env := loadTestEnvelope(t, "peppol/invoice-prices-include-vat.json")
			inv, ok := env.Extract().(*bill.Invoice)
			require.True(t, ok)

			// 9.99 including 21% VAT is 8.2562 net: never exact in cents.
			base := inv.Lines[0]
			inv.Lines = make([]*bill.Line, 0, lines)
			for range lines {
				l, item := *base, *base.Item
				price := num.MakeAmount(999, 2)
				item.Price = &price
				l.Item = &item
				l.Quantity = num.MakeAmount(1, 0)
				l.Discounts, l.Charges = nil, nil
				inv.Lines = append(inv.Lines, &l)
			}
			require.NoError(t, env.Calculate())
			payable := inv.Totals.Payable

			doc, err := ubl.ConvertInvoice(env, ubl.WithContext(ubl.ContextPeppol))
			require.NoError(t, err)

			assertTotalsAddUp(t, doc)
			assertAmountsFitCurrency(t, doc)

			// The amount owed is unchanged, however the cents fall.
			require.NotNil(t, doc.LegalMonetaryTotal.PayableAmount)
			assert.Equal(t, payable.String(), doc.LegalMonetaryTotal.PayableAmount.Value)
		})
	}
}

// assertTotalsAddUp checks BR-CO-10, BR-CO-13, BR-CO-14, BR-CO-15 and
// BR-CO-16, the arithmetic the document totals have to satisfy.
func assertTotalsAddUp(t *testing.T, doc *ubl.Invoice) {
	t.Helper()

	amount := func(a *ubl.Amount) num.Amount {
		if a == nil {
			return num.MakeAmount(0, 2)
		}
		out, err := num.AmountFromString(a.Value)
		require.NoError(t, err)
		return out
	}

	lines := doc.InvoiceLines
	if len(lines) == 0 {
		lines = doc.CreditNoteLines
	}

	// BR-CO-10: the sum of line net amounts (BT-106) is the sum of BT-131.
	sum := num.MakeAmount(0, 2)
	for _, l := range lines {
		sum = sum.Add(amount(&l.LineExtensionAmount))
	}
	m := doc.LegalMonetaryTotal
	assert.Equal(t, sum.String(), m.LineExtensionAmount.Value, "BR-CO-10: BT-106 is the sum of the line net amounts")

	// BR-CO-13: BT-109 = BT-106 - BT-107 + BT-108.
	exclusive := amount(&m.LineExtensionAmount).
		Subtract(amount(m.AllowanceTotalAmount)).
		Add(amount(m.ChargeTotalAmount))
	assert.Equal(t, exclusive.String(), m.TaxExclusiveAmount.Value, "BR-CO-13: BT-109 is BT-106 less allowances plus charges")

	if len(doc.TaxTotal) == 0 {
		return
	}

	// BR-CO-14: BT-110 = the sum of the category tax amounts (BT-117).
	if subtotals := doc.TaxTotal[0].TaxSubtotal; len(subtotals) > 0 {
		categories := num.MakeAmount(0, 2)
		for _, s := range subtotals {
			categories = categories.Add(amount(&s.TaxAmount))
		}
		assert.Equal(t, categories.String(), doc.TaxTotal[0].TaxAmount.Value, "BR-CO-14: BT-110 is the sum of the category tax amounts")
	}

	// BR-CO-15: BT-112 = BT-109 + BT-110.
	inclusive := amount(&m.TaxExclusiveAmount).Add(amount(&doc.TaxTotal[0].TaxAmount))
	assert.Equal(t, inclusive.String(), m.TaxInclusiveAmount.Value, "BR-CO-15: BT-112 is BT-109 plus the total tax")

	// BR-CO-16: BT-115 = BT-112 - BT-113 + BT-114.
	payable := amount(&m.TaxInclusiveAmount).
		Subtract(amount(m.PrepaidAmount)).
		Add(amount(m.PayableRoundingAmount))
	require.NotNil(t, m.PayableAmount)
	assert.Equal(t, payable.String(), m.PayableAmount.Value, "BR-CO-16: BT-115 is BT-112 less prepaid plus rounding")
}
