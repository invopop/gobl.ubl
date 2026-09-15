package ubl

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

// replacementCharRef matches the XML character references that decode to
// U+FFFD. They are plain ASCII in the document, so they survive a byte-level
// clean and only become the replacement character once the XML is decoded.
var replacementCharRef = regexp.MustCompile(`&#(?:[xX]0*[fF][fF][fF][dD]|0*65533);`)

// cleanString drops what a sender's broken encoding leaves behind: bytes that
// are not valid UTF-8, which the XML decoder rejects, and U+FFFD, which gobl's
// canonical JSON rejects, written literally or as a character reference.
// Neither is recoverable. Applied to the whole document before decoding, and
// idempotent.
//
// The U+FFFD half is a stopgap for invopop/gobl#975.
func cleanString(s string) string {
	s = replacementCharRef.ReplaceAllString(s, "")
	if utf8.ValidString(s) && !strings.ContainsRune(s, utf8.RuneError) {
		return s
	}
	s = strings.ToValidUTF8(s, "")
	return strings.ReplaceAll(s, string(utf8.RuneError), "")
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
