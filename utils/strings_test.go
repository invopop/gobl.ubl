package utils

import (
	"testing"

	"github.com/invopop/gobl/cbc"
	"github.com/stretchr/testify/assert"
)

// Define tests for the FormatKey function
func TestFormatKey(t *testing.T) {
	assert.Equal(t, cbc.Key("test"), FormatKey("Test"))
	assert.Equal(t, cbc.Key("test-key-2"), FormatKey("Test Key 2"))
	assert.Equal(t, cbc.Key("multiple-spaces"), FormatKey("Multiple   Spaces"))
	assert.Equal(t, cbc.Key("numbers-123"), FormatKey("Numbers 123"))
	assert.Equal(t, cbc.Key("trailing-space"), FormatKey("Trailing Space  "))
	assert.Equal(t, cbc.Key("mixed-case-with-123-numbers"), FormatKey("MiXeD cAsE wItH 123 NuMbErS"))
}
