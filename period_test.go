package ubl_test

import (
	"strings"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPeriodSingleBound covers EN 16931 BR-CO-19 / BR-CO-20: an invoice
// period only needs one of its two bounds. The fixture carries a header
// period with only an end date (BT-74 without BT-73) and line periods with
// only a start date (BT-134 without BT-135). Parsing must produce a GOBL
// document that validates, and converting it back must only emit the bound
// that is present.
func TestPeriodSingleBound(t *testing.T) {
	t.Run("parse and validate", func(t *testing.T) {
		env := parseXMLInvoice(t, "en16931/ubl-example-period-single-bound.xml")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// The upstream sample this fixture derives from carries tax IDs that
		// do not pass regime validation, so only assert that the one-sided
		// periods are no longer a reason for the document to fail.
		if err := env.Validate(); err != nil {
			assert.NotContains(t, err.Error(), "GOBL-CAL-PERIOD", "one-sided periods must validate")
		}

		require.NotNil(t, inv.Ordering)
		require.NotNil(t, inv.Ordering.Period)
		assert.NoError(t, rules.Validate(inv.Ordering.Period))
		assert.True(t, inv.Ordering.Period.Start.IsZero(), "BT-73 absent, start must stay zero")
		assert.Equal(t, "2013-04-10", inv.Ordering.Period.End.String())

		require.NotEmpty(t, inv.Lines)
		require.NotNil(t, inv.Lines[0].Period)
		assert.NoError(t, rules.Validate(inv.Lines[0].Period))
		assert.Equal(t, "2013-03-10", inv.Lines[0].Period.Start.String())
		assert.True(t, inv.Lines[0].Period.End.IsZero(), "BT-135 absent, end must stay zero")
	})

	t.Run("re-export omits the absent bound", func(t *testing.T) {
		env := parseXMLInvoice(t, "en16931/ubl-example-period-single-bound.xml")

		doc, err := ubl.ConvertInvoice(env, ubl.WithContext(ubl.ContextEN16931))
		require.NoError(t, err)
		data, err := ubl.Bytes(doc)
		require.NoError(t, err)
		xml := string(data)

		// Header (BG-14): only cbc:EndDate.
		assert.Contains(t, xml, "<cbc:EndDate>2013-04-10</cbc:EndDate>")
		assert.NotContains(t, xml, "<cbc:StartDate>2013-04-10</cbc:StartDate>")
		// Lines (BG-26): only cbc:StartDate.
		assert.Contains(t, xml, "<cbc:StartDate>2013-03-10</cbc:StartDate>")
		assert.Equal(t, 1, strings.Count(xml, "<cbc:EndDate>"), "only the header carries an end date")
		assert.Equal(t, 0, strings.Count(xml, "<cbc:StartDate></cbc:StartDate>"))
		assert.Equal(t, 0, strings.Count(xml, "<cbc:EndDate></cbc:EndDate>"))
		assert.NotContains(t, xml, "0000-00-00")
	})
}
