package utils

import (
	"fmt"
	"regexp"

	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

// noteCodePattern matches the #CODE#text format used in UBL notes to encode
// UNTDID 4451 text subject qualifier codes, e.g. "#AAI#some text".
var noteCodePattern = regexp.MustCompile(`^#([A-Z0-9]+)#(.*)$`)

// ParseNote converts a raw UBL note string into a GOBL Note. If the string
// matches the #CODE#text format the code is stored as the untdid text-subject ext.
func ParseNote(text string) *org.Note {
	text = CleanString(text)
	if m := noteCodePattern.FindStringSubmatch(text); m != nil {
		return &org.Note{
			Ext:  tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyTextSubject: cbc.Code(m[1])}),
			Text: m[2],
		}
	}
	return &org.Note{Text: text}
}

// FormatNote renders a GOBL note as its UBL note text.
func FormatNote(note *org.Note) string {
	if note == nil {
		return ""
	}

	if code := note.Ext.Get(untdid.ExtKeyTextSubject); code != cbc.CodeEmpty {
		return fmt.Sprintf("#%s#%s", code, note.Text)
	}
	return note.Text
}
