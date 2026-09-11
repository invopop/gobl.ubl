package ubl

import (
	"testing"

	"github.com/invopop/gobl/catalogues/untdid"
	cur "github.com/invopop/gobl/currency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoblPeriodDates(t *testing.T) {
	t.Run("both ends", func(t *testing.T) {
		p, err := goblPeriodDates(&Period{StartDate: "2024-01-01", EndDate: "2024-01-31"})
		require.NoError(t, err)
		require.NotNil(t, p)
		require.NotNil(t, p.Start)
		require.NotNil(t, p.End)
		assert.Equal(t, "2024-01-01", p.Start.String())
		assert.Equal(t, "2024-01-31", p.End.String())
	})

	// GOBL v0.505 made both ends optional, so a half-open period is valid.
	t.Run("start only", func(t *testing.T) {
		p, err := goblPeriodDates(&Period{StartDate: "2024-01-01"})
		require.NoError(t, err)
		require.NotNil(t, p)
		require.NotNil(t, p.Start)
		assert.Nil(t, p.End)
	})

	t.Run("end only", func(t *testing.T) {
		p, err := goblPeriodDates(&Period{EndDate: "2024-01-31"})
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.Nil(t, p.Start)
		require.NotNil(t, p.End)
	})

	t.Run("an empty period is no period", func(t *testing.T) {
		p, err := goblPeriodDates(&Period{})
		require.NoError(t, err)
		assert.Nil(t, p)
	})

	t.Run("a malformed start date is an error", func(t *testing.T) {
		_, err := goblPeriodDates(&Period{StartDate: "15/01/2024"})
		assert.Error(t, err)
	})

	t.Run("a malformed end date is an error", func(t *testing.T) {
		_, err := goblPeriodDates(&Period{StartDate: "2024-01-01", EndDate: "31-01-2024"})
		assert.Error(t, err)
	})
}

func TestGoblReference(t *testing.T) {
	t.Run("every field maps across", func(t *testing.T) {
		ref, err := goblReference(&Reference{
			ID:                  IDType{Value: "REF-1"},
			IssueDate:           "2024-01-15",
			DocumentType:        "Delivery note",
			DocumentTypeCode:    "130",
			DocumentDescription: "Goods delivered",
			ValidityPeriod:      &Period{StartDate: "2024-01-01", EndDate: "2024-01-31"},
		})
		require.NoError(t, err)
		assert.Equal(t, "REF-1", ref.Code.String())
		assert.Equal(t, "Delivery note", ref.Reason)
		assert.Equal(t, "Goods delivered", ref.Description)
		require.NotNil(t, ref.IssueDate)
		assert.Equal(t, "2024-01-15", ref.IssueDate.String())
		assert.Equal(t, "130", ref.Ext.Get(untdid.ExtKeyDocumentType).String())
		require.NotNil(t, ref.Period)
	})

	t.Run("the code alone is enough", func(t *testing.T) {
		ref, err := goblReference(&Reference{ID: IDType{Value: "REF-2"}})
		require.NoError(t, err)
		assert.Equal(t, "REF-2", ref.Code.String())
		assert.Nil(t, ref.IssueDate)
		assert.Nil(t, ref.Period)
		assert.Empty(t, ref.Reason)
	})

	t.Run("a malformed issue date is an error", func(t *testing.T) {
		_, err := goblReference(&Reference{ID: IDType{Value: "REF-3"}, IssueDate: "15 Jan 2024"})
		assert.Error(t, err)
	})

	t.Run("a malformed validity period is an error", func(t *testing.T) {
		_, err := goblReference(&Reference{
			ID:             IDType{Value: "REF-4"},
			ValidityPeriod: &Period{StartDate: "nonsense"},
		})
		assert.Error(t, err)
	})
}

func TestGoblExchangeRates(t *testing.T) {
	amount := func(v string) TaxTotal {
		return TaxTotal{TaxAmount: Amount{Value: v}}
	}

	t.Run("the rate is derived from the two tax totals", func(t *testing.T) {
		rates := goblExchangeRates(cur.EUR, cur.USD, []TaxTotal{amount("100.00"), amount("110.00")})
		require.Len(t, rates, 1)
		assert.Equal(t, cur.EUR, rates[0].From)
		assert.Equal(t, cur.USD, rates[0].To)
		assert.Equal(t, "1.10", rates[0].Amount.String())
	})

	t.Run("a single total carries no rate", func(t *testing.T) {
		assert.Nil(t, goblExchangeRates(cur.EUR, cur.USD, []TaxTotal{amount("100.00")}))
	})

	t.Run("no totals at all", func(t *testing.T) {
		assert.Nil(t, goblExchangeRates(cur.EUR, cur.USD, nil))
	})

	t.Run("a zero document amount would divide by zero", func(t *testing.T) {
		assert.Nil(t, goblExchangeRates(cur.EUR, cur.USD, []TaxTotal{amount("0.00"), amount("110.00")}))
	})

	t.Run("an unparseable document amount", func(t *testing.T) {
		assert.Nil(t, goblExchangeRates(cur.EUR, cur.USD, []TaxTotal{amount("n/a"), amount("110.00")}))
	})

	t.Run("an unparseable tax amount", func(t *testing.T) {
		assert.Nil(t, goblExchangeRates(cur.EUR, cur.USD, []TaxTotal{amount("100.00"), amount("n/a")}))
	})
}
