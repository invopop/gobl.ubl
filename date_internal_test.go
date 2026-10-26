package ubl

import (
	"testing"

	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
)

func TestFormatDate(t *testing.T) {
	t.Run("a date is written in the UBL format", func(t *testing.T) {
		assert.Equal(t, "2024-01-15", formatDate(cal.MakeDate(2024, 1, 15)))
	})

	t.Run("a zero date is written as nothing", func(t *testing.T) {
		assert.Equal(t, "", formatDate(cal.Date{}))
	})
}

func TestFormatDatePtr(t *testing.T) {
	// cal.Period start and end are optional since GOBL v0.505, so a period may
	// carry only one of the two.
	t.Run("a date is written in the UBL format", func(t *testing.T) {
		assert.Equal(t, "2024-02-29", formatDatePtr(cal.NewDate(2024, 2, 29)))
	})

	t.Run("an absent date is written as nothing", func(t *testing.T) {
		assert.Equal(t, "", formatDatePtr(nil))
	})

	t.Run("a zero date is written as nothing", func(t *testing.T) {
		assert.Equal(t, "", formatDatePtr(&cal.Date{}))
	})
}

func TestFormatNote(t *testing.T) {
	t.Run("a plain note is its own text", func(t *testing.T) {
		assert.Equal(t, "Delivered on time", formatNote(&org.Note{Text: "Delivered on time"}))
	})

	t.Run("a subject code is encoded into the text", func(t *testing.T) {
		note := &org.Note{
			Text: "Some remarks",
			Ext:  tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyTextSubject: "AAI"}),
		}
		assert.Equal(t, "#AAI#Some remarks", formatNote(note))
	})

	t.Run("no note at all", func(t *testing.T) {
		assert.Equal(t, "", formatNote(nil))
	})

	t.Run("round trips through parseNote", func(t *testing.T) {
		for _, text := range []string{"#AAI#Some remarks", "plain text"} {
			assert.Equal(t, text, formatNote(parseNote(text)), "round trip of %q", text)
		}
	})
}

func TestContactName(t *testing.T) {
	tests := []struct {
		name string
		in   *org.Name
		want string
	}{
		{"given and surname", &org.Name{Given: "Jane", Surname: "Sample"}, "Jane Sample"},
		{"surname only", &org.Name{Surname: "Sample"}, "Sample"},
		{"given only", &org.Name{Given: "Jane"}, "Jane"},
		{"an empty name", &org.Name{}, ""},
		{"no name at all", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, contactName(tt.in))
		})
	}
}

func TestNormalizeTaxPercent(t *testing.T) {
	tests := []struct {
		name string
		in   *string
		want string
	}{
		{"a whole percentage loses its trailing zeros", ptrTo("21.00"), "21"},
		{"a fractional percentage is kept", ptrTo("8.50"), "8.5"},
		{"surrounding space is trimmed", ptrTo(" 21.0 "), "21"},
		{"a leading decimal point is filled in", ptrTo(".5"), "0.5"},
		{"zero", ptrTo("0"), "0"},
		{"a value that is not a number is passed through", ptrTo("N/A"), "N/A"},
		{"no percentage at all", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeTaxPercent(tt.in))
		})
	}
}

func ptrTo(s string) *string { return &s }
