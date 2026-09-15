package ubl

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/invopop/gobl/addons/eu/en16931"
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

// goblItemUnit records a UN/ECE unit code on the item. The raw code is kept
// in the untdid-unit extension and mapped to a GOBL unit key when one exists.
func goblItemUnit(item *org.Item, code cbc.Code) {
	if code == cbc.CodeEmpty {
		return
	}
	item.Ext = item.Ext.Set(untdid.ExtKeyUnit, code)
	if unit := en16931.UnitFromUNTDID(code); unit != cbc.KeyEmpty {
		item.Unit = unit
	}
}

// unitCodeUNTDID returns the UN/ECE unit code for an item, preferring the
// untdid-unit extension set by the EN 16931 addon.
func unitCodeUNTDID(item *org.Item) string {
	code := item.Ext.Get(untdid.ExtKeyUnit)
	if code == cbc.CodeEmpty {
		code = en16931.UnitToUNTDID(item.Unit)
	}
	return code.String()
}

// attrUnitCode returns the UN/ECE unit code for an attribute unit, falling
// back to "ZZ" (mutually defined) when GOBL has no mapping.
func attrUnitCode(unit cbc.Key) string {
	if code := en16931.UnitToUNTDID(unit); code != cbc.CodeEmpty {
		return code.String()
	}
	return "ZZ"
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
