package ubl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoblCanFold(t *testing.T) {
	ref := func(status string) []*LineBillingReference {
		return []*LineBillingReference{{
			InvoiceDocumentReference: &LineDocumentStatusReference{DocumentStatusCode: status},
		}}
	}
	line := func(status, qty, amount string) *InvoiceLine {
		l := &InvoiceLine{
			InvoicedQuantity:    &Quantity{Value: qty},
			LineExtensionAmount: Amount{Value: amount},
			Price:               &Price{PriceAmount: Amount{Value: "1"}},
			Item: &Item{ClassifiedTaxCategory: &ClassifiedTaxCategory{
				ID:        &IDType{Value: "S"},
				TaxScheme: &TaxScheme{ID: IDType{Value: "VAT"}},
			}},
		}
		if status != "" {
			l.hierarchy = ref(status)[0]
		}
		return l
	}
	group := func() *InvoiceLine { return line(lineStatusGroup, "1", "10.00") }
	detail := func() *InvoiceLine { return line(lineStatusDetail, "1", "10.00") }
	taxes := map[string]*taxCategoryInfo{}

	assert.True(t, goblCanFold(group(), []*InvoiceLine{detail()}, taxes))

	tests := []struct {
		name   string
		parent *InvoiceLine
		kids   func() []*InvoiceLine
	}{
		{"a GROUP line with its own allowances", func() *InvoiceLine {
			g := group()
			g.AllowanceCharge = []*AllowanceCharge{{}}
			return g
		}(), func() []*InvoiceLine { return []*InvoiceLine{detail()} }},
		{"an untyped line without a price", func() *InvoiceLine {
			l := line("", "1", "10.00")
			l.Price = nil
			return l
		}(), func() []*InvoiceLine { return []*InvoiceLine{line(lineStatusInformation, "1", "0")} }},
		{"an INFORMATION parent", line(lineStatusInformation, "1", "0"), func() []*InvoiceLine {
			return []*InvoiceLine{line(lineStatusInformation, "1", "0")}
		}},
		{"a zero parent quantity", line(lineStatusGroup, "0", "10.00"), func() []*InvoiceLine {
			return []*InvoiceLine{detail()}
		}},
		{"a nested GROUP line", group(), func() []*InvoiceLine { return []*InvoiceLine{group()} }},
		{"a DETAIL line under a priced line", line("", "1", "10.00"), func() []*InvoiceLine {
			return []*InvoiceLine{detail()}
		}},
		{"a DETAIL line without a tax category", group(), func() []*InvoiceLine {
			d := detail()
			d.Item = nil
			return []*InvoiceLine{d}
		}},
		{"an allowance that does not divide by the quantity", line(lineStatusGroup, "3", "10.00"), func() []*InvoiceLine {
			d := line(lineStatusDetail, "3", "10.00")
			d.AllowanceCharge = []*AllowanceCharge{{Amount: Amount{Value: "1.00"}}}
			return []*InvoiceLine{d}
		}},
		{"a DETAIL line without an amount", group(), func() []*InvoiceLine {
			return []*InvoiceLine{line(lineStatusDetail, "1", "")}
		}},
		{"only INFORMATION lines under a GROUP line", group(), func() []*InvoiceLine {
			return []*InvoiceLine{line(lineStatusInformation, "1", "0")}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, goblCanFold(tt.parent, tt.kids(), taxes))
		})
	}
}

func TestGoblLineQuantity(t *testing.T) {
	q, ok := goblLineQuantity(&InvoiceLine{CreditedQuantity: &Quantity{Value: "2"}})
	assert.True(t, ok)
	assert.Equal(t, "2", q.String())

	_, ok = goblLineQuantity(&InvoiceLine{InvoicedQuantity: &Quantity{Value: "x"}})
	assert.False(t, ok)
}
