// Package utils holds generic helpers with no UBL-specific logic, shared
// across gobl.ubl and country overlay converters.
package utils

import (
	"regexp"
	"strings"

	"github.com/invopop/gobl/cbc"
)

// CleanString strips the Unicode replacement character (U+FFFD) which can
// appear in badly-encoded XML documents and causes canonical JSON
// serialization to fail.
func CleanString(s string) string {
	return strings.ReplaceAll(s, "\uFFFD", "")
}

// FormatKey formats a string to comply with GOBL key requirements.
func FormatKey(key string) cbc.Key {
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, " ", "-")
	re := regexp.MustCompile(`[^a-z0-9-+]`)
	key = re.ReplaceAllString(key, "")
	key = strings.Trim(key, "-+")
	re = regexp.MustCompile(`[-+]{2,}`)
	key = re.ReplaceAllString(key, "-")
	return cbc.Key(key)
}
