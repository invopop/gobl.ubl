package ubl

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
)

func TestWritesSubLines(t *testing.T) {
	breakdown := []*bill.SubLine{{Quantity: num.MakeAmount(1, 0), Item: &org.Item{Name: "Part"}}}

	t.Run("a line with a breakdown", func(t *testing.T) {
		assert.True(t, writesSubLines(&bill.Line{Breakdown: breakdown}))
	})
	t.Run("a line without a breakdown", func(t *testing.T) {
		assert.False(t, writesSubLines(&bill.Line{}))
	})
	t.Run("a line with its own allowances", func(t *testing.T) {
		l := &bill.Line{Breakdown: breakdown, Discounts: []*bill.LineDiscount{{Amount: num.MakeAmount(1, 0)}}}
		assert.False(t, writesSubLines(l))
	})
}
