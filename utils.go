package ubl

import (
	"fmt"
	"reflect"
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

// cleanDocument strips replacement characters from every text field a parsed
// document carries. Senders whose own encoding broke upstream ship U+FFFD in
// place of the character they lost, and GOBL's canonical JSON refuses any
// document holding one, so a single field is enough to lose the whole invoice.
// Cleaning the parsed fields rather than the raw payload leaves the document's
// own bytes alone, and reaches the fields no explicit cleanString call covers.
func cleanDocument(doc any) {
	cleanValue(reflect.ValueOf(doc))
}

func cleanValue(v reflect.Value) {
	switch v.Kind() { //nolint:exhaustive // only the kinds a document can hold
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			cleanValue(v.Elem())
		}
	case reflect.Struct:
		for i := range v.NumField() {
			cleanValue(v.Field(i))
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			cleanValue(v.Index(i))
		}
	case reflect.String:
		// Unexported fields cannot be set, and need no cleaning.
		if !v.CanSet() {
			return
		}
		if s := v.String(); strings.Contains(s, "\uFFFD") {
			v.SetString(cleanString(s))
		}
	}
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

// goblUnit records the UNTDID unit code in the extensions and returns them
// alongside the matching GOBL unit key, which is empty when GOBL has no unit
// for the code. The code the document was written with is always kept, as GOBL
// will preserve it rather than derive it back from the key.
func goblUnit(ext tax.Extensions, code cbc.Code) (tax.Extensions, cbc.Key) {
	return ext.Set(untdid.ExtKeyUnit, code), untdid.UnitKey(code)
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
