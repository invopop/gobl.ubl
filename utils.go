package ubl

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
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

// noteCodePattern matches the #CODE#text format used in UBL notes to encode
// UNTDID 4451 text subject qualifier codes, e.g. "#AAI#some text".
var noteCodePattern = regexp.MustCompile(`^#([A-Z0-9]+)#(.*)$`)

// parseNote converts a raw UBL note string into a GOBL Note. If the string
// matches the #CODE#text format the code is stored as the untdid text-subject ext.
func parseNote(text string) *org.Note {
	text = cleanString(text)
	if m := noteCodePattern.FindStringSubmatch(text); m != nil {
		return &org.Note{
			Ext:  tax.Extensions{untdid.ExtKeyTextSubject: cbc.Code(m[1])},
			Text: m[2],
		}
	}
	return &org.Note{Text: text}
}

func formatNote(note *org.Note) string {
	if note == nil {
		return ""
	}

	if code := note.Ext.Get(untdid.ExtKeyTextSubject); code != cbc.CodeEmpty {
		return fmt.Sprintf("#%s#%s", code, note.Text)
	}
	return note.Text
}
