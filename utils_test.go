package ubl

import (
	"testing"

	"github.com/invopop/gobl/cbc"
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
		{"Standard invoice", "380", "standard"},
		{"Credit note", "381", "credit-note"},
		{"Corrective invoice", "384", "corrective"},
		{"Debit note", "383", "debit-note"},
		{"Unknown type code", "999", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := typeCodeParse(tt.input)
			assert.Equal(t, tt.expected, string(result))
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
