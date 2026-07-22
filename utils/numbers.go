package utils

import (
	"math"
	"strconv"
	"strings"

	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
)

// NormalizeNumericString cleans up numeric strings to ensure they can be parsed correctly.
// It handles:
// - Leading/trailing whitespace (e.g., " 123.45 " -> "123.45")
// - Numbers starting with decimal point (e.g., ".07" -> "0.07")
func NormalizeNumericString(s string) string {
	// Trim whitespace
	s = strings.TrimSpace(s)

	// Add leading zero if string starts with decimal point
	if strings.HasPrefix(s, ".") {
		s = "0" + s
	}

	return s
}

// NormalizeTaxPercent converts a percent string to a canonical form by stripping trailing zeros,
// so that "20", "20.0", and "20.00" all map to "20".
func NormalizeTaxPercent(percent *string) string {
	if percent == nil {
		return ""
	}
	s := NormalizeNumericString(*percent)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// RescaleToCurrency rounds the amount to the natural precision of the given
// currency code (e.g. 2 for EUR, 0 for JPY). Falls back to the amount's
// existing precision if the currency code is unknown.
func RescaleToCurrency(a num.Amount, ccy string) string {
	if def := currency.Code(ccy).Def(); def != nil {
		return def.Rescale(a).String()
	}
	return a.String()
}

// CalculateRequiredPrecision determines the decimal precision needed when
// dividing a price by a base quantity to avoid rounding errors.
// Formula: price_decimals + ceil(log10(base_quantity))
// Example: price with 2 decimals divided by 100 needs 2 + 2 = 4 decimals
func CalculateRequiredPrecision(price, baseQuantity num.Amount) uint32 {
	priceExp := price.Exp()

	// Convert baseQuantity to a whole number to calculate needed decimal places
	baseQtyNormalized := baseQuantity.Rescale(0)
	baseQtyFloat := math.Abs(float64(baseQtyNormalized.Value()))

	additionalDecimals := uint32(0)
	if baseQtyFloat > 1 {
		// log10(100) = 2, log10(1000) = 3, etc.
		additionalDecimals = uint32(math.Ceil(math.Log10(baseQtyFloat)))
	}

	return priceExp + additionalDecimals
}
