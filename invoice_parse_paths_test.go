package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/cef"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parsedFixture parses a UBL example from test/data/parse into its document
// structure, so tests can adjust it before converting.
func parsedFixture(t *testing.T, name string) *ubl.Invoice {
	t.Helper()
	data, err := testLoadXML(name)
	require.NoError(t, err)
	doc, err := ubl.Parse(data)
	require.NoError(t, err)
	in, ok := doc.(*ubl.Invoice)
	require.True(t, ok)
	return in
}

func convertParsed(t *testing.T, in *ubl.Invoice) *bill.Invoice {
	t.Helper()
	env, err := in.Convert()
	require.NoError(t, err)
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)
	return inv
}

func strPtr(s string) *string { return &s }

func TestParseInvoiceDates(t *testing.T) {
	in := parsedFixture(t, "en16931/ubl-example1.xml")
	in.IssueTime = "10:20:30"
	inv := convertParsed(t, in)
	require.NotNil(t, inv.IssueTime)
	assert.Equal(t, "10:20:30", inv.IssueTime.String())
}

func TestParseDeliveryPaths(t *testing.T) {
	t.Run("actual and latest dates form a period", func(t *testing.T) {
		in := parsedFixture(t, "peppol/base-example.xml")
		in.Delivery = []*ubl.Delivery{{
			ActualDeliveryDate: strPtr("2024-01-01"),
			LatestDeliveryDate: strPtr("2024-01-31"),
		}}
		inv := convertParsed(t, in)
		require.NotNil(t, inv.Delivery)
		assert.Equal(t, "2024-01-01", inv.Delivery.Date.String())
		require.NotNil(t, inv.Delivery.Period)
		assert.Equal(t, "2024-01-01", inv.Delivery.Period.Start.String())
		assert.Equal(t, "2024-01-31", inv.Delivery.Period.End.String())
	})

	t.Run("estimated delivery period alone", func(t *testing.T) {
		in := parsedFixture(t, "peppol/base-example.xml")
		in.Delivery = []*ubl.Delivery{{
			EstimatedDeliveryPeriod: &ubl.Period{StartDate: "2024-02-01", EndDate: "2024-02-29"},
		}}
		inv := convertParsed(t, in)
		require.NotNil(t, inv.Delivery)
		require.NotNil(t, inv.Delivery.Period)
		assert.Equal(t, "2024-02-01", inv.Delivery.Period.Start.String())
		assert.Equal(t, "2024-02-29", inv.Delivery.Period.End.String())
	})

	t.Run("estimated delivery period with party", func(t *testing.T) {
		in := parsedFixture(t, "peppol/base-example.xml")
		in.Delivery = []*ubl.Delivery{{
			EstimatedDeliveryPeriod: &ubl.Period{StartDate: "2024-02-01", EndDate: "2024-02-29"},
			DeliveryParty: &ubl.Party{
				PartyName: &ubl.PartyName{Name: "Warehouse"},
			},
		}}
		inv := convertParsed(t, in)
		require.NotNil(t, inv.Delivery)
		require.NotNil(t, inv.Delivery.Receiver)
		assert.Equal(t, "Warehouse", inv.Delivery.Receiver.Name)
		require.NotNil(t, inv.Delivery.Period)
		assert.Equal(t, "2024-02-01", inv.Delivery.Period.Start.String())
		assert.Equal(t, "2024-02-29", inv.Delivery.Period.End.String())
	})

	t.Run("unnamed delivery party is dropped", func(t *testing.T) {
		in := parsedFixture(t, "peppol/base-example.xml")
		in.Delivery = []*ubl.Delivery{{
			ActualDeliveryDate: strPtr("2024-01-01"),
			DeliveryParty: &ubl.Party{
				PartyLegalEntity: &ubl.PartyLegalEntity{RegistrationName: strPtr("")},
			},
		}}
		inv := convertParsed(t, in)
		require.NotNil(t, inv.Delivery)
		assert.Nil(t, inv.Delivery.Receiver)
	})

	t.Run("delivery terms", func(t *testing.T) {
		in := parsedFixture(t, "peppol/base-example.xml")
		in.DeliveryTerms = &ubl.DeliveryTerms{ID: "EXW"}
		inv := convertParsed(t, in)
		require.NotNil(t, inv.Delivery)
		require.Len(t, inv.Delivery.Identities, 1)
		assert.Equal(t, "EXW", inv.Delivery.Identities[0].Code.String())
	})
}

func TestParseBillingReferences(t *testing.T) {
	in := parsedFixture(t, "peppol/base-example.xml")
	in.BillingReference = []*ubl.BillingReference{
		{SelfBilledInvoiceDocumentReference: &ubl.Reference{ID: ubl.IDType{Value: "SB-1"}}},
		{CreditNoteDocumentReference: &ubl.Reference{ID: ubl.IDType{Value: "CN-1"}}},
		{AdditionalDocumentReference: &ubl.Reference{ID: ubl.IDType{Value: "AD-1"}, IssueDate: "2024-01-15"}},
	}
	inv := convertParsed(t, in)
	require.Len(t, inv.Preceding, 3)
	assert.Equal(t, "SB-1", inv.Preceding[0].Code.String())
	assert.Equal(t, "CN-1", inv.Preceding[1].Code.String())
	assert.Equal(t, "AD-1", inv.Preceding[2].Code.String())
	assert.Equal(t, "2024-01-15", inv.Preceding[2].IssueDate.String())
}

func TestParseTaxExchangeRate(t *testing.T) {
	rate := func(source, target, amount string) *ubl.ExchangeRate {
		return &ubl.ExchangeRate{
			SourceCurrencyCode: strPtr(source),
			TargetCurrencyCode: strPtr(target),
			CalculationRate:    strPtr(amount),
			Date:               strPtr("2024-01-02"),
		}
	}

	t.Run("matching rate is used", func(t *testing.T) {
		in := parsedFixture(t, "peppol/base-example.xml")
		in.TaxCurrencyCode = "USD"
		in.TaxExchangeRate = rate(in.DocumentCurrencyCode, "USD", "1.10")
		inv := convertParsed(t, in)
		require.Len(t, inv.ExchangeRates, 1)
		assert.Equal(t, "USD", inv.ExchangeRates[0].To.String())
		assert.Equal(t, "1.10", inv.ExchangeRates[0].Amount.String())
		require.NotNil(t, inv.ExchangeRates[0].At)
		assert.Equal(t, "2024-01-02T00:00:00", inv.ExchangeRates[0].At.String())
	})

	for name, er := range map[string]func(in *ubl.Invoice) *ubl.ExchangeRate{
		"source mismatch": func(_ *ubl.Invoice) *ubl.ExchangeRate { return rate("GBP", "USD", "1.10") },
		"target mismatch": func(in *ubl.Invoice) *ubl.ExchangeRate { return rate(in.DocumentCurrencyCode, "GBP", "1.10") },
		"invalid rate":    func(in *ubl.Invoice) *ubl.ExchangeRate { return rate(in.DocumentCurrencyCode, "USD", "x") },
		"missing rate": func(in *ubl.Invoice) *ubl.ExchangeRate {
			r := rate(in.DocumentCurrencyCode, "USD", "1.10")
			r.CalculationRate = nil
			return r
		},
	} {
		t.Run(name+" is ignored", func(t *testing.T) {
			in := parsedFixture(t, "peppol/base-example.xml")
			in.TaxCurrencyCode = "USD"
			in.TaxExchangeRate = er(in)
			inv := convertParsed(t, in)
			for _, r := range inv.ExchangeRates {
				assert.NotEqual(t, "1.10", r.Amount.String())
			}
		})
	}
}

func TestParseSkipsLinesWithoutPrice(t *testing.T) {
	in := parsedFixture(t, "peppol/base-example.xml")
	require.Greater(t, len(in.InvoiceLines), 1)
	in.InvoiceLines[0].Price = nil
	inv := convertParsed(t, in)
	assert.Len(t, inv.Lines, len(in.InvoiceLines)-1)
}

func TestParseInvoiceErrors(t *testing.T) {
	tests := map[string]func(in *ubl.Invoice){
		"issue date":     func(in *ubl.Invoice) { in.IssueDate = "bad" },
		"issue time":     func(in *ubl.Invoice) { in.IssueTime = "bad" },
		"tax point date": func(in *ubl.Invoice) { in.TaxPointDate = "bad" },
		"delivery start": func(in *ubl.Invoice) {
			in.Delivery = []*ubl.Delivery{{ActualDeliveryDate: strPtr("bad"), LatestDeliveryDate: strPtr("2024-01-31")}}
		},
		"delivery end": func(in *ubl.Invoice) {
			in.Delivery = []*ubl.Delivery{{ActualDeliveryDate: strPtr("2024-01-01"), LatestDeliveryDate: strPtr("bad")}}
		},
		"delivery date": func(in *ubl.Invoice) {
			in.Delivery = []*ubl.Delivery{{ActualDeliveryDate: strPtr("bad")}}
		},
		"estimated delivery period": func(in *ubl.Invoice) {
			in.Delivery = []*ubl.Delivery{{EstimatedDeliveryPeriod: &ubl.Period{StartDate: "bad"}}}
		},
		"billing reference date": func(in *ubl.Invoice) {
			in.BillingReference = []*ubl.BillingReference{
				{AdditionalDocumentReference: &ubl.Reference{ID: ubl.IDType{Value: "AD-1"}, IssueDate: "bad"}},
			}
		},
		"line price": func(in *ubl.Invoice) { in.InvoiceLines[0].Price.PriceAmount.Value = "bad" },
		"line quantity": func(in *ubl.Invoice) {
			in.InvoiceLines[0].InvoicedQuantity = &ubl.Quantity{Value: "bad"}
		},
		"charge amount": func(in *ubl.Invoice) {
			in.AllowanceCharge = append(in.AllowanceCharge, ubl.AllowanceCharge{
				ChargeIndicator: true,
				Amount:          ubl.Amount{Value: "bad"},
			})
		},
		"discount amount": func(in *ubl.Invoice) {
			in.AllowanceCharge = append(in.AllowanceCharge, ubl.AllowanceCharge{
				Amount: ubl.Amount{Value: "bad"},
			})
		},
		"charge base amount": func(in *ubl.Invoice) {
			in.AllowanceCharge = append(in.AllowanceCharge, ubl.AllowanceCharge{
				ChargeIndicator: true,
				Amount:          ubl.Amount{Value: "10"},
				BaseAmount:      &ubl.Amount{Value: "bad"},
			})
		},
		"charge multiplier": func(in *ubl.Invoice) {
			in.AllowanceCharge = append(in.AllowanceCharge, ubl.AllowanceCharge{
				ChargeIndicator:         true,
				Amount:                  ubl.Amount{Value: "10"},
				BaseAmount:              &ubl.Amount{Value: "100"},
				MultiplierFactorNumeric: strPtr("bad"),
			})
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			in := parsedFixture(t, "peppol/base-example.xml")
			mutate(in)
			_, err := in.Convert()
			assert.Error(t, err)
		})
	}
}

func TestParseDeclaredTotalsBypass(t *testing.T) {
	// bypass makes the declared payable amount disagree with the
	// calculation, so the declared totals are recorded as sent.
	bypass := func(in *ubl.Invoice) {
		mt := &in.LegalMonetaryTotal
		mt.PayableAmount = &ubl.Amount{Value: "999999.99"}
		mt.PayableRoundingAmount = &ubl.Amount{Value: "0.00"}
	}

	t.Run("records the declared rounding and tax breakdown", func(t *testing.T) {
		in := parsedFixture(t, "peppol/base-example.xml")
		bypass(in)
		require.NotEmpty(t, in.TaxTotal[0].TaxSubtotal)
		st := &in.TaxTotal[0].TaxSubtotal[0]
		st.TaxCategory.TaxExemptionReasonCode = strPtr("VATEX-EU-O")
		in.TaxTotal[0].TaxSubtotal = append(in.TaxTotal[0].TaxSubtotal,
			ubl.TaxSubtotal{TaxableAmount: ubl.Amount{Value: "1"}, TaxAmount: ubl.Amount{Value: "1"}},
			ubl.TaxSubtotal{
				TaxableAmount: ubl.Amount{Value: "bad"},
				TaxAmount:     ubl.Amount{Value: "1"},
				TaxCategory:   ubl.TaxCategory{TaxScheme: &ubl.TaxScheme{ID: ubl.IDType{Value: "VAT"}}},
			},
			ubl.TaxSubtotal{
				TaxableAmount: ubl.Amount{Value: "1"},
				TaxAmount:     ubl.Amount{Value: "bad"},
				TaxCategory:   ubl.TaxCategory{TaxScheme: &ubl.TaxScheme{ID: ubl.IDType{Value: "VAT"}}},
			},
		)

		inv := convertParsed(t, in)
		assert.Contains(t, inv.GetTags(), tax.TagBypass)
		require.NotNil(t, inv.Totals.Rounding)
		assert.True(t, inv.Totals.Rounding.IsZero())
		assert.Equal(t, "999999.99", inv.Totals.Payable.String())
		require.NotNil(t, inv.Totals.Taxes)
		require.Len(t, inv.Totals.Taxes.Categories, 1)
		rate := inv.Totals.Taxes.Categories[0].Rates[0]
		assert.Equal(t, "VATEX-EU-O", rate.Ext.Get(cef.ExtKeyVATEX).String())
	})

	t.Run("without usable subtotals", func(t *testing.T) {
		in := parsedFixture(t, "peppol/base-example.xml")
		bypass(in)
		for i := range in.TaxTotal {
			in.TaxTotal[i].TaxSubtotal = nil
		}
		inv := convertParsed(t, in)
		assert.Contains(t, inv.GetTags(), tax.TagBypass)
		assert.Equal(t, "999999.99", inv.Totals.Payable.String())
	})

	t.Run("without declared amounts", func(t *testing.T) {
		in := parsedFixture(t, "peppol/base-example.xml")
		bypass(in)
		in.LegalMonetaryTotal.TaxExclusiveAmount = ubl.Amount{}
		in.InvoiceLines[0].LineExtensionAmount = ubl.Amount{}
		for i := range in.TaxTotal {
			in.TaxTotal[i].TaxAmount = ubl.Amount{}
			in.TaxTotal[i].TaxSubtotal = nil
		}
		inv := convertParsed(t, in)
		assert.Contains(t, inv.GetTags(), tax.TagBypass)
		assert.Equal(t, "999999.99", inv.Totals.Payable.String())
	})

	t.Run("declared tax total disagrees", func(t *testing.T) {
		in := parsedFixture(t, "peppol/base-example.xml")
		in.TaxTotal[0].TaxAmount.Value = "1.00"
		inv := convertParsed(t, in)
		assert.Contains(t, inv.GetTags(), tax.TagBypass)
		assert.Equal(t, "1.00", inv.Totals.Tax.String())
	})
}
