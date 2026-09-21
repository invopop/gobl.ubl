package ubl

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	zatca "github.com/invopop/gobl.sa.zatca/addon"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
)

// Define tests for the ParseDate function
func TestParseDate(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{"Valid date", "2023-05-15", "2023-05-15", false},
		{"Invalid date", "2023-13-45", "", true},
		{"Empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDate(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result.String())
			}
		})
	}
}

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

// Define tests for the unit helpers
func TestUnits(t *testing.T) {
	tests := []struct {
		name  string
		input string
		unit  cbc.Key
		code  cbc.Code
	}{
		{"Known UNTDID code", "HUR", "h", "HUR"},
		{"Known UNTDID code", "SEC", "s", "SEC"},
		{"Known UNTDID code", "MTR", "m", "MTR"},
		{"Known UNTDID code", "GRM", "g", "GRM"},
		{"Unknown UNTDID code", "XYZ", "", "XYZ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext, unit := goblUnit(tax.Extensions{}, cbc.Code(tt.input))
			assert.Equal(t, tt.unit, unit)
			// The code the document was written with is always recorded.
			assert.Equal(t, tt.code, ext.Get(untdid.ExtKeyUnit))
			assert.Equal(t, tt.code, untdidUnit(ext, unit))
		})
	}

	t.Run("without extension", func(t *testing.T) {
		assert.Equal(t, cbc.Code("HUR"), untdidUnit(tax.Extensions{}, org.UnitHour))
	})

	t.Run("no unit at all", func(t *testing.T) {
		assert.Equal(t, cbc.CodeEmpty, untdidUnit(tax.Extensions{}, cbc.KeyEmpty))
	})

	t.Run("labels", func(t *testing.T) {
		assert.Equal(t, "kg", unitLabel(org.UnitKilogram, "KGM"))
		assert.Equal(t, "XYZ", unitLabel(cbc.KeyEmpty, "XYZ"))
		assert.Equal(t, "", unitLabel(cbc.KeyEmpty, cbc.CodeEmpty))
	})
}

// Define tests for the FormatKey function
func TestFormatKey(t *testing.T) {
	assert.Equal(t, cbc.Key("test"), formatKey("Test"))
	assert.Equal(t, cbc.Key("test-key-2"), formatKey("Test Key 2"))
	assert.Equal(t, cbc.Key("multiple-spaces"), formatKey("Multiple   Spaces"))
	assert.Equal(t, cbc.Key("numbers-123"), formatKey("Numbers 123"))
	assert.Equal(t, cbc.Key("trailing-space"), formatKey("Trailing Space  "))
	assert.Equal(t, cbc.Key("mixed-case-with-123-numbers"), formatKey("MiXeD cAsE wItH 123 NuMbErS"))
}

func TestCalculateRequiredPrecision(t *testing.T) {
	tests := []struct {
		name         string
		price        string
		baseQuantity string
		expected     uint32
	}{
		{
			name:         "base quantity of 1",
			price:        "100.00",
			baseQuantity: "1",
			expected:     2, // 2 + 0 (log10(1) = 0)
		},
		{
			name:         "base quantity of 2",
			price:        "200.00",
			baseQuantity: "2",
			expected:     3, // 2 + ceil(log10(2)) = 2 + 1
		},
		{
			name:         "base quantity of 10",
			price:        "100.00",
			baseQuantity: "10",
			expected:     3, // 2 + ceil(log10(10)) = 2 + 1
		},
		{
			name:         "base quantity of 100",
			price:        "100.00",
			baseQuantity: "100",
			expected:     4, // 2 + ceil(log10(100)) = 2 + 2
		},
		{
			name:         "base quantity of 1000",
			price:        "100.00",
			baseQuantity: "1000",
			expected:     5, // 2 + ceil(log10(1000)) = 2 + 3
		},
		{
			name:         "price with more decimals",
			price:        "100.12345",
			baseQuantity: "100",
			expected:     7, // 5 + ceil(log10(100)) = 5 + 2
		},
		{
			name:         "price with no decimals",
			price:        "100",
			baseQuantity: "100",
			expected:     2, // 0 + ceil(log10(100)) = 0 + 2
		},
		{
			name:         "fractional base quantity less than 1",
			price:        "100.00",
			baseQuantity: "0.5",
			expected:     2, // 2 + 0 (baseQtyFloat <= 1 after Rescale(0))
		},
		{
			name:         "non-power-of-10 base quantity",
			price:        "100.00",
			baseQuantity: "3",
			expected:     3, // 2 + ceil(log10(3)) = 2 + 1
		},
		{
			name:         "large base quantity",
			price:        "100.00",
			baseQuantity: "10000",
			expected:     6, // 2 + ceil(log10(10000)) = 2 + 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, err := num.AmountFromString(tt.price)
			assert.NoError(t, err)
			baseQty, err := num.AmountFromString(tt.baseQuantity)
			assert.NoError(t, err)

			result := calculateRequiredPrecision(price, baseQty)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanString(t *testing.T) {
	t.Run("leaves clean text untouched", func(t *testing.T) {
		in := "Première vérification"
		assert.Equal(t, in, cleanString(in))
	})

	t.Run("drops the replacement character", func(t *testing.T) {
		// A sender emits U+FFFD when its own encoding conversion has already
		// given up. It is valid UTF-8, so it reaches gobl, where canonical
		// JSON refuses it and the document fails to digest.
		assert.Equal(t, "Premire", cleanString("Premi\uFFFDre"))
	})

	t.Run("drops a decoded character reference", func(t *testing.T) {
		// &#xFFFD; in the document arrives here already decoded.
		assert.Equal(t, "bad  char", cleanString("bad \uFFFD char"))
	})

	t.Run("is idempotent", func(t *testing.T) {
		in := "Premi\uFFFDre"
		assert.Equal(t, cleanString(in), cleanString(cleanString(in)))
	})

	t.Run("preserves valid multi-byte text", func(t *testing.T) {
		in := "1000 m² – 20 €"
		assert.Equal(t, in, cleanString(in))
	})
}

// elementText matches an element holding non-empty text, capturing its local
// name so the injection can be done one element name at a time.
var elementText = regexp.MustCompile(`<([a-zA-Z]+:)?([A-Za-z]+)>([^<>]*[A-Za-z][^<>]*)</`)

// freeText names the UBL elements that carry text a human typed, which is
// where a sender's broken encoding shows up. Codes, identifiers and other
// controlled vocabularies are deliberately excluded: they cannot carry an
// accent, so they cannot arrive mangled, and wrapping them would be noise.
var freeText = map[string]bool{
	"Note":                  true,
	"Name":                  true,
	"RegistrationName":      true,
	"Description":           true,
	"DocumentDescription":   true,
	"AllowanceChargeReason": true,
	"TaxExemptionReason":    true,
	"CompanyLegalForm":      true,
	"StreetName":            true,
	"AdditionalStreetName":  true,
	"CityName":              true,
	"CitySubdivisionName":   true,
	"CountrySubentity":      true,
	"BuildingNumber":        true,
	"ElectronicMail":        true,
	"InstructionNote":       true,
	// BuyerReference, AccountingCost, PaymentID and SalesOrderID are excluded:
	// they map to cbc.Code, so they follow the same rule as any other code.
}

// TestReplacementCharCoverage guards the cleanString calls. It injects U+FFFD
// into one free-text element at a time across every fixture and fails if the
// marker reaches GOBL, either surviving into a field or failing the digest.
//
// A new free-text field that nobody remembered to wrap shows up here rather
// than on a customer invoice.
func TestReplacementCharCoverage(t *testing.T) {
	files := fixtures(t)
	if len(files) == 0 {
		t.Fatal("no fixtures found; the guard would pass vacuously")
	}

	var gaps []gap
	seen := map[string]bool{}
	checked := 0

	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if _, err := parseFixture(raw); err != nil {
			continue // fixture is not a clean baseline; it proves nothing
		}
		for _, elem := range textElements(raw) {
			if !freeText[elem] {
				continue
			}
			one := regexp.MustCompile(`(<([a-zA-Z]+:)?` + elem + `>)([^<>]*[A-Za-z][^<>]*)(</)`)
			dirty := one.ReplaceAllString(string(raw), "${1}${3}�${4}")
			if dirty == string(raw) {
				continue
			}
			checked++
			doc, err := parseFixture([]byte(dirty))
			if err != nil {
				if v := failingValue(err); v != "" {
					record(&gaps, seen, gap{elem, v, filepath.Base(f)})
				}
				continue
			}
			for _, hit := range findReplacementChar(reflect.ValueOf(doc)) {
				record(&gaps, seen, gap{elem, hit, filepath.Base(f)})
			}
		}
	}

	t.Logf("checked %d element injections across %d fixtures", checked, len(files))
	if len(gaps) == 0 {
		return
	}
	sort.Slice(gaps, func(i, j int) bool { return gaps[i].elem < gaps[j].elem })
	var b strings.Builder
	fmt.Fprintf(&b, "%d element(s) carry U+FFFD into GOBL; each needs a cleanString call:\n", len(gaps))
	for _, g := range gaps {
		fmt.Fprintf(&b, "  %-28s %-46s [%s]\n", g.elem, g.value, g.src)
	}
	t.Error(b.String())
}

type gap struct{ elem, value, src string }

func record(gaps *[]gap, seen map[string]bool, g gap) {
	k := g.elem + "|" + g.value
	if seen[k] {
		return
	}
	seen[k] = true
	*gaps = append(*gaps, g)
}

func textElements(raw []byte) []string {
	set := map[string]bool{}
	for _, m := range elementText.FindAllStringSubmatch(string(raw), -1) {
		if strings.TrimSpace(m[3]) != "" {
			set[m[2]] = true
		}
	}
	out := make([]string, 0, len(set))
	for e := range set {
		out = append(out, e)
	}
	sort.Strings(out)
	return out
}

// failingValue pulls the offending string out of a canonical JSON error, which
// is what names the field that went unwrapped.
func failingValue(err error) string {
	const marker = "json: unsupported value: "
	i := strings.Index(err.Error(), marker)
	if i < 0 {
		return ""
	}
	v := err.Error()[i+len(marker):]
	if len(v) > 44 {
		v = v[:44] + "…"
	}
	return v
}

func findReplacementChar(v reflect.Value) []string {
	var out []string
	var walk func(reflect.Value, string)
	walk = func(v reflect.Value, path string) {
		switch v.Kind() {
		case reflect.Pointer, reflect.Interface:
			if !v.IsNil() {
				walk(v.Elem(), path)
			}
		case reflect.Struct:
			for i := range v.NumField() {
				if f := v.Type().Field(i); f.IsExported() {
					walk(v.Field(i), path+"."+f.Name)
				}
			}
		case reflect.Slice, reflect.Array:
			for i := range v.Len() {
				walk(v.Index(i), path+"[]")
			}
		case reflect.Map:
			for _, k := range v.MapKeys() {
				walk(v.MapIndex(k), path+"["+fmt.Sprint(k.Interface())+"]")
			}
		case reflect.String:
			if strings.ContainsRune(v.String(), '�') {
				out = append(out, path)
			}
		}
	}
	walk(v, "")
	return out
}

func fixtures(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, pat := range []string{"test/data/parse/*.xml", "test/data/parse/*/*.xml"} {
		m, _ := filepath.Glob(pat)
		out = append(out, m...)
	}
	return out
}

func parseFixture(raw []byte) (any, error) {
	doc, err := Parse(raw)
	if err != nil {
		return nil, err
	}
	var env *gobl.Envelope
	switch d := doc.(type) {
	case *Invoice:
		env, err = d.Convert()
	case *ApplicationResponse:
		env, err = d.Convert()
	default:
		return nil, fmt.Errorf("unhandled document %T", doc)
	}
	if err != nil {
		return nil, err
	}
	return env.Document, nil
}
