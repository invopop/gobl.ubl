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

// goblUnit converts a UNTDID unit code into the matching GOBL unit key,
// leaving the code in the extensions only when GOBL has no unit for it.
func goblUnit(ext tax.Extensions, code cbc.Code) (tax.Extensions, cbc.Key) {
	unit, ext := untdid.NormalizeUnit(cbc.KeyEmpty, ext.Set(untdid.ExtKeyUnit, code))
	return ext, unit
}

// untdidUnit returns the UNTDID unit code for a unit and its extensions,
// preferring the code preserved in the extensions over the mapping from the
// GOBL unit key.
func untdidUnit(ext tax.Extensions, unit cbc.Key) cbc.Code {
	if code := ext.Get(untdid.ExtKeyUnit); code != cbc.CodeEmpty {
		return code
	}
	return untdid.UnitCode(unit)
}

// unitLabel describes a unit for presentation, falling back to the UNTDID code
// when the unit has no GOBL key.
func unitLabel(unit cbc.Key, code cbc.Code) string {
	if unit != cbc.KeyEmpty {
		return unit.String()
	}
	return code.String()
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
			Ext:  tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyTextSubject: cbc.Code(m[1])}),
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
