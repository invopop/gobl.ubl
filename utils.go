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

// cleanString strips the traces of a sender mishandling its own character
// encoding, neither of which is recoverable here — the original characters are
// gone before the document reaches us:
//
//   - byte sequences that are not valid UTF-8, which the XML decoder rejects
//     outright with "invalid UTF-8";
//   - the Unicode replacement character (U+FFFD), which a sender emits when its
//     own conversion has already given up. It is valid UTF-8, so it reaches
//     gobl, where canonical JSON refuses it and the document fails to digest.
//
// Parse applies this to the whole document before decoding, so a field nobody
// thought to wrap cannot reintroduce the problem. It stays safe to call on
// individual values too, and is idempotent.
//
// The U+FFFD half is a stopgap: gobl/c14n rejects a valid code point, fixed
// upstream in invopop/gobl#975. Drop it once the gobl dependency carries that.
func cleanString(s string) string {
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
