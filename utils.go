package ubl

import (
	"regexp"
	"strings"

	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
)

// cleanString strips the Unicode replacement character (U+FFFD) which can
// appear in badly-encoded XML documents and causes canonical JSON
// serialization to fail.
func cleanString(s string) string {
	return strings.ReplaceAll(s, "\uFFFD", "")
}

// formatKey formats a string to comply with GOBL key requirements.
func formatKey(key string) cbc.Key {
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, " ", "-")
	re := regexp.MustCompile(`[^a-z0-9-+]`)
	key = re.ReplaceAllString(key, "")
	key = strings.Trim(key, "-+")
	re = regexp.MustCompile(`[-+]{2,}`)
	key = re.ReplaceAllString(key, "-")
	return cbc.Key(key)
}

// goblUnitFromUNECE maps UN/ECE code to GOBL equivalent.
func goblUnitFromUNECE(unece cbc.Code) org.Unit {
	for _, def := range org.UnitDefinitions {
		if def.UNECE == unece {
			return def.Unit
		}
	}
	return org.Unit(unece)
}
