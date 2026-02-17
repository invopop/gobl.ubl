package ubl

import (
	"testing"

	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
)

// Define tests for the ParseDate function
func TestParseDate(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{"Valid date", "2023-05-15", "2023-05-15", false},
		{"Invalid date", "2023-13-45", "", true},
		{"Empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDate(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result.String())
			}
		})
	}
}

// Define tests for the TypeCodeParse function
func TestTypeCodeParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Proforma invoice", "325", "proforma"},
		{"Standard invoice", "380", "standard"},
		{"Credit note", "381", "credit-note"},
		{"Debit note", "383", "debit-note"},
		{"Corrective invoice", "384", "corrective"},
		{"Self-billed invoice", "389", "standard"},
		{"Partial invoice", "326", "standard"},
		{"Self-billed credit note", "261", "credit-note"},
		{"Unknown type code", "999", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := typeCodeParse(tt.input)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

// Define tests for the TagCodeParse function
func TestTagCodeParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []cbc.Key
	}{
		{"Self-billed invoice", "389", []cbc.Key{tax.TagSelfBilled}},
		{"Partial invoice", "326", []cbc.Key{tax.TagPartial}},
		{"Self-billed credit note", "261", []cbc.Key{tax.TagSelfBilled}},
		{"Standard invoice - no tag", "380", nil},
		{"Credit note - no tag", "381", nil},
		{"Unknown code - no tag", "999", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tagCodeParse(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Define tests for the UnitFromUNECE function
func TestUnitFromUNECE(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Known UNECE code", "HUR", "h"},
		{"Known UNECE code", "SEC", "s"},
		{"Known UNECE code", "MTR", "m"},
		{"Known UNECE code", "GRM", "g"},
		{"Unknown UNECE code", "XYZ", "XYZ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblUnitFromUNECE(cbc.Code(tt.input))
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

// Define tests for the FormatKey function
func TestFormatKey(t *testing.T) {
	assert.Equal(t, cbc.Key("test"), formatKey("Test"))
	assert.Equal(t, cbc.Key("test-key-2"), formatKey("Test Key 2"))
	assert.Equal(t, cbc.Key("multiple-spaces"), formatKey("Multiple   Spaces"))
	assert.Equal(t, cbc.Key("numbers-123"), formatKey("Numbers 123"))
	assert.Equal(t, cbc.Key("trailing-space"), formatKey("Trailing Space  "))
	assert.Equal(t, cbc.Key("mixed-case-with-123-numbers"), formatKey("MiXeD cAsE wItH 123 NuMbErS"))
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

			result := calculateRequiredPrecision(price, baseQty)
			assert.Equal(t, tt.expected, result)
		})
	}
}
