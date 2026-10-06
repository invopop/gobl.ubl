package ubl

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
)

// TestWritesSubLinesOnlyExtended covers every context: sub-invoice lines only
// exist in EXTENDED-CTC-FR, so no other context may write them.
func TestWritesSubLinesOnlyExtended(t *testing.T) {
	l := &bill.Line{
		Breakdown: []*bill.SubLine{{Quantity: num.MakeAmount(1, 0), Item: &org.Item{Name: "Part"}}},
	}
	for _, ctx := range contexts {
		assert.Equal(t, ctx.Is(ContextPeppolFranceExtended), writesSubLines(ctx, l), ctx.CustomizationID)
	}
}
