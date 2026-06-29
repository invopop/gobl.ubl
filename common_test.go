package ubl

import (
	"testing"

	"github.com/invopop/gobl/cbc"
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
			result := normalizeNumericString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGoblTaxSchemeCategory(t *testing.T) {
	// OIOUBL's VAT scheme code maps back to the GOBL VAT category.
	assert.Equal(t, cbc.Code("VAT"), goblTaxSchemeCategory("63"))
	// Other profiles already use the GOBL code — passes through unchanged.
	assert.Equal(t, cbc.Code("VAT"), goblTaxSchemeCategory("VAT"))
	assert.Equal(t, cbc.Code("OSS"), goblTaxSchemeCategory("OSS"))
}

func TestGoblTaxCategoryCode(t *testing.T) {
	// OIOUBL category names map back to the UNTDID 5305 codes.
	assert.Equal(t, cbc.Code("S"), goblTaxCategoryCode("StandardRated"))
	assert.Equal(t, cbc.Code("Z"), goblTaxCategoryCode("ZeroRated"))
	assert.Equal(t, cbc.Code("AE"), goblTaxCategoryCode("ReverseCharge"))
	// Already-UNTDID values pass through unchanged.
	assert.Equal(t, cbc.Code("S"), goblTaxCategoryCode("S"))
	assert.Equal(t, cbc.Code("E"), goblTaxCategoryCode("E"))
}
