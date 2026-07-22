package ubl

import (
	"testing"

	zatca "github.com/invopop/gobl.sa.zatca/addon"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
)

// Define tests for the TypeCodeParse function
func TestTypeCodeParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Proforma invoice", "325", "proforma"},
		{"Standard invoice", "380", "standard"},
		{"Credit note", "381", "credit-note"},
		{"Debit note", "383", "debit-note"},
		{"Corrective invoice", "384", "corrective"},
		{"Self-billed invoice", "389", "standard"},
		{"Partial invoice", "326", "standard"},
		{"Self-billed credit note", "261", "credit-note"},
		{"Unknown type code", "999", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := typeCodeParse(&IDType{Value: tt.input})
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

// Define tests for the TagCodeParse function
func TestTagCodeParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []cbc.Key
	}{
		{"Self-billed invoice", "389", []cbc.Key{tax.TagSelfBilled}},
		{"Partial invoice", "326", []cbc.Key{tax.TagPartial}},
		{"Self-billed credit note", "261", []cbc.Key{tax.TagSelfBilled}},
		{"Standard invoice - no tag", "380", nil},
		{"Credit note - no tag", "381", nil},
		{"Unknown code - no tag", "999", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tagCodeParse(&IDType{Value: tt.input}, Context{})
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTypeCodeParseZATCA confirms the document type is derived from the UNTDID
// value alone: the KSA-2 transaction-type flags carried in Name never change it.
func TestTypeCodeParseZATCA(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		code     string
		expected string
	}{
		{"Standard, plain code", "388", "0100000", "standard"},
		{"Standard with third-party/nominal/self-billed flags", "388", "0111001", "standard"},
		{"Simplified standard", "388", "0200000", "standard"},
		{"Credit note with flags", "381", "0210001", "credit-note"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := tt.code
			result := typeCodeParse(&IDType{Value: tt.value, Name: &code})
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

// TestTagCodeParseZATCA confirms every KSA-2 transaction-type flag is restored
// as its tag, mirroring the addon's normalizeInvoiceType (the convert direction).
func TestTagCodeParseZATCA(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected []cbc.Key
	}{
		{"Standard - no flags", "0100000", nil},
		{"Simplified", "0200000", []cbc.Key{tax.TagSimplified}},
		{"Summary", "0100010", []cbc.Key{zatca.TagSummary}},
		{"Export", "0100100", []cbc.Key{tax.TagExport}},
		{"Third-party, nominal, self-billed", "0111001", []cbc.Key{zatca.TagThirdParty, zatca.TagNominal, tax.TagSelfBilled}},
		{"Simplified, export, summary", "0200110", []cbc.Key{tax.TagSimplified, tax.TagExport, zatca.TagSummary}},
		{"All flags set", "0211111", []cbc.Key{tax.TagSimplified, zatca.TagThirdParty, zatca.TagNominal, tax.TagExport, zatca.TagSummary, tax.TagSelfBilled}},
		{"Empty code - no tags", "", nil},
		{"Malformed short code - no tags", "010000", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := tt.code
			result := tagCodeParse(&IDType{Name: &code}, ContextZATCA)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Define tests for the UnitFromUNECE function
func TestUnitFromUNECE(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Known UNECE code", "HUR", "h"},
		{"Known UNECE code", "SEC", "s"},
		{"Known UNECE code", "MTR", "m"},
		{"Known UNECE code", "GRM", "g"},
		{"Unknown UNECE code", "XYZ", "XYZ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goblUnitFromUNECE(cbc.Code(tt.input))
			assert.Equal(t, tt.expected, string(result))
		})
	}
}
