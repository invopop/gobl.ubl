package ubl

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReorderCreditNoteTaxPointDate(t *testing.T) {
	t.Run("indented: moves TaxPointDate before the type code", func(t *testing.T) {
		in := "<CreditNote>\n" +
			"  <cbc:ID>C1</cbc:ID>\n" +
			"  <cbc:CreditNoteTypeCode>381</cbc:CreditNoteTypeCode>\n" +
			"  <cbc:Note>note</cbc:Note>\n" +
			"  <cbc:TaxPointDate>2024-06-15</cbc:TaxPointDate>\n" +
			"  <cbc:DocumentCurrencyCode>DKK</cbc:DocumentCurrencyCode>\n" +
			"</CreditNote>"
		want := "<CreditNote>\n" +
			"  <cbc:ID>C1</cbc:ID>\n" +
			"  <cbc:TaxPointDate>2024-06-15</cbc:TaxPointDate>\n" +
			"  <cbc:CreditNoteTypeCode>381</cbc:CreditNoteTypeCode>\n" +
			"  <cbc:Note>note</cbc:Note>\n" +
			"  <cbc:DocumentCurrencyCode>DKK</cbc:DocumentCurrencyCode>\n" +
			"</CreditNote>"
		assert.Equal(t, want, string(reorderCreditNoteTaxPointDate([]byte(in))))
	})

	t.Run("compact: moves TaxPointDate before the type code", func(t *testing.T) {
		in := "<CreditNote><cbc:ID>C1</cbc:ID>" +
			"<cbc:CreditNoteTypeCode>381</cbc:CreditNoteTypeCode>" +
			"<cbc:Note>note</cbc:Note>" +
			"<cbc:TaxPointDate>2024-06-15</cbc:TaxPointDate>" +
			"<cbc:DocumentCurrencyCode>DKK</cbc:DocumentCurrencyCode></CreditNote>"
		want := "<CreditNote><cbc:ID>C1</cbc:ID>" +
			"<cbc:TaxPointDate>2024-06-15</cbc:TaxPointDate>" +
			"<cbc:CreditNoteTypeCode>381</cbc:CreditNoteTypeCode>" +
			"<cbc:Note>note</cbc:Note>" +
			"<cbc:DocumentCurrencyCode>DKK</cbc:DocumentCurrencyCode></CreditNote>"
		assert.Equal(t, want, string(reorderCreditNoteTaxPointDate([]byte(in))))
	})

	t.Run("no type code: unchanged", func(t *testing.T) {
		in := "<Invoice>\n  <cbc:TaxPointDate>2024-06-15</cbc:TaxPointDate>\n</Invoice>"
		assert.Equal(t, in, string(reorderCreditNoteTaxPointDate([]byte(in))))
	})

	t.Run("already ordered: unchanged", func(t *testing.T) {
		in := "<CreditNote>\n  <cbc:TaxPointDate>2024-06-15</cbc:TaxPointDate>\n  <cbc:CreditNoteTypeCode>381</cbc:CreditNoteTypeCode>\n</CreditNote>"
		assert.Equal(t, in, string(reorderCreditNoteTaxPointDate([]byte(in))))
	})

	t.Run("result keeps every element exactly once", func(t *testing.T) {
		in := "<CreditNote>\n  <cbc:CreditNoteTypeCode>381</cbc:CreditNoteTypeCode>\n  <cbc:TaxPointDate>2024-06-15</cbc:TaxPointDate>\n</CreditNote>"
		out := string(reorderCreditNoteTaxPointDate([]byte(in)))
		assert.Equal(t, 1, strings.Count(out, "<cbc:TaxPointDate>"))
		assert.Equal(t, 1, strings.Count(out, "<cbc:CreditNoteTypeCode>"))
		assert.Less(t, strings.Index(out, "<cbc:TaxPointDate>"), strings.Index(out, "<cbc:CreditNoteTypeCode>"))
	})
}
