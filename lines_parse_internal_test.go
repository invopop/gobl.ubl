package ubl

import (
	"testing"

	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestGoblLinePriceDetails(t *testing.T) {
	read := func(p *Price) (*org.Item, error) {
		price, err := goblLinePrice(p)
		if err != nil {
			return nil, err
		}
		item := &org.Item{Price: &price}
		return item, goblLinePriceDetails(p, item)
	}
	price := func(amount string) *Price {
		return &Price{PriceAmount: Amount{Value: amount}}
	}
	allowance := func(amount, base string) *AllowanceCharge {
		return &AllowanceCharge{Amount: Amount{Value: amount}, BaseAmount: &Amount{Value: base}}
	}

	t.Run("a list price less the discount", func(t *testing.T) {
		p := price("108.00")
		p.AllowanceCharge = allowance("12.00", "120.00")
		item, err := read(p)
		require.NoError(t, err)
		assert.Equal(t, "120.00", item.List.String())
		assert.Equal(t, "12.00", item.Discount.String())
	})
	t.Run("a list price with no discount", func(t *testing.T) {
		p := price("120.00")
		p.AllowanceCharge = allowance("0.00", "120.00")
		item, err := read(p)
		require.NoError(t, err)
		assert.Equal(t, "120.00", item.List.String())
		assert.Nil(t, item.Discount)
	})
	t.Run("a list price that does not come to the net price", func(t *testing.T) {
		p := price("100.00")
		p.AllowanceCharge = allowance("12.00", "120.00")
		item, err := read(p)
		require.NoError(t, err)
		assert.Nil(t, item.List)
		assert.Nil(t, item.Discount)
	})
	t.Run("a price charge", func(t *testing.T) {
		p := price("108.00")
		p.AllowanceCharge = allowance("12.00", "120.00")
		p.AllowanceCharge.ChargeIndicator = true
		item, err := read(p)
		require.NoError(t, err)
		assert.Nil(t, item.List)
	})
	t.Run("a base quantity of one", func(t *testing.T) {
		p := price("10.00")
		p.BaseQuantity = &Quantity{Value: "1"}
		item, err := read(p)
		require.NoError(t, err)
		assert.Nil(t, item.Per)
	})
	t.Run("invalid amounts", func(t *testing.T) {
		p := price("10.00")
		p.BaseQuantity = &Quantity{Value: "x"}
		_, err := read(p)
		assert.Error(t, err)

		p = price("10.00")
		p.AllowanceCharge = allowance("1", "x")
		_, err = read(p)
		assert.Error(t, err)

		p = price("10.00")
		p.AllowanceCharge = allowance("x", "11")
		_, err = read(p)
		assert.Error(t, err)

		_, err = read(price("x"))
		assert.Error(t, err)
		assert.False(t, goblHasPriceDetails(price("x")))
		p = price("10.00")
		p.BaseQuantity = &Quantity{Value: "x"}
		assert.False(t, goblHasPriceDetails(p))
	})
	t.Run("a parent with price details does not fold", func(t *testing.T) {
		p := price("10.00")
		p.BaseQuantity = &Quantity{Value: "10"}
		assert.True(t, goblHasPriceDetails(p))
		parent := &InvoiceLine{Price: p, InvoicedQuantity: &Quantity{Value: "1"}}
		assert.False(t, goblCanFold(parent, nil, map[string]*taxCategoryInfo{}))
	})
}
