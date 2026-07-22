package utils

import (
	"testing"

	"github.com/invopop/gobl/num"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeNumericString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no change needed",
			input:    "123.45",
			expected: "123.45",
		},
		{
			name:     "leading space",
			input:    " 123.45",
			expected: "123.45",
		},
		{
			name:     "trailing space",
			input:    "123.45 ",
			expected: "123.45",
		},
		{
			name:     "both spaces",
			input:    " 123.45 ",
			expected: "123.45",
		},
		{
			name:     "leading decimal",
			input:    ".07",
			expected: "0.07",
		},
		{
			name:     "leading decimal with space",
			input:    " .07 ",
			expected: "0.07",
		},
		{
			name:     "percentage with spaces",
			input:    " 9.0% ",
			expected: "9.0%",
		},
		{
			name:     "percentage with leading decimal",
			input:    ".5%",
			expected: "0.5%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeNumericString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateRequiredPrecision(t *testing.T) {
	tests := []struct {
		name         string
		price        string
		baseQuantity string
		expected     uint32
	}{
		{
			name:         "base quantity of 1",
			price:        "100.00",
			baseQuantity: "1",
			expected:     2, // 2 + 0 (log10(1) = 0)
		},
		{
			name:         "base quantity of 2",
			price:        "200.00",
			baseQuantity: "2",
			expected:     3, // 2 + ceil(log10(2)) = 2 + 1
		},
		{
			name:         "base quantity of 10",
			price:        "100.00",
			baseQuantity: "10",
			expected:     3, // 2 + ceil(log10(10)) = 2 + 1
		},
		{
			name:         "base quantity of 100",
			price:        "100.00",
			baseQuantity: "100",
			expected:     4, // 2 + ceil(log10(100)) = 2 + 2
		},
		{
			name:         "base quantity of 1000",
			price:        "100.00",
			baseQuantity: "1000",
			expected:     5, // 2 + ceil(log10(1000)) = 2 + 3
		},
		{
			name:         "price with more decimals",
			price:        "100.12345",
			baseQuantity: "100",
			expected:     7, // 5 + ceil(log10(100)) = 5 + 2
		},
		{
			name:         "price with no decimals",
			price:        "100",
			baseQuantity: "100",
			expected:     2, // 0 + ceil(log10(100)) = 0 + 2
		},
		{
			name:         "fractional base quantity less than 1",
			price:        "100.00",
			baseQuantity: "0.5",
			expected:     2, // 2 + 0 (baseQtyFloat <= 1 after Rescale(0))
		},
		{
			name:         "non-power-of-10 base quantity",
			price:        "100.00",
			baseQuantity: "3",
			expected:     3, // 2 + ceil(log10(3)) = 2 + 1
		},
		{
			name:         "large base quantity",
			price:        "100.00",
			baseQuantity: "10000",
			expected:     6, // 2 + ceil(log10(10000)) = 2 + 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, err := num.AmountFromString(tt.price)
			assert.NoError(t, err)
			baseQty, err := num.AmountFromString(tt.baseQuantity)
			assert.NoError(t, err)

			result := CalculateRequiredPrecision(price, baseQty)
			assert.Equal(t, tt.expected, result)
		})
	}
}
