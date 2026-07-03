package ubl_test

import (
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// xsdValidatorSource is a tiny Java program that compiles a W3C XML Schema and
// validates one or more XML files against it, printing one "VALID:"/"INVALID:"
// line per file and exiting non-zero if any file failed. Java ships a full
// XSD 1.0 validator in the standard library, which the Go ecosystem lacks, so
// the tests below compile and exec this at run time (guarded by a skip when no
// JDK is available).
const xsdValidatorSource = `import javax.xml.XMLConstants;
import javax.xml.transform.stream.StreamSource;
import javax.xml.validation.Schema;
import javax.xml.validation.SchemaFactory;
import javax.xml.validation.Validator;
import java.io.File;

public class Validate {
  public static void main(String[] args) throws Exception {
    SchemaFactory f = SchemaFactory.newInstance(XMLConstants.W3C_XML_SCHEMA_NS_URI);
    Schema s = f.newSchema(new File(args[0]));
    boolean ok = true;
    for (int i = 1; i < args.length; i++) {
      Validator v = s.newValidator();
      try {
        v.validate(new StreamSource(new File(args[i])));
        System.out.println("VALID: " + args[i]);
      } catch (Exception e) {
        ok = false;
        System.out.println("INVALID: " + args[i] + " :: " + e.getMessage());
      }
    }
    if (!ok) {
      System.exit(1);
    }
  }
}
`

// compileXSDValidator writes the Java validator into a temp dir and compiles
// it, returning the directory to use as the java classpath. The calling test
// is skipped when no JDK is available.
func compileXSDValidator(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("java"); err != nil {
		t.Skip("java not available, skipping XSD validation")
	}
	if _, err := exec.LookPath("javac"); err != nil {
		t.Skip("javac not available, skipping XSD validation")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "Validate.java")
	require.NoError(t, os.WriteFile(src, []byte(xsdValidatorSource), 0o600))
	out, err := exec.Command("javac", src).CombinedOutput()
	require.NoError(t, err, "javac failed: %s", out)
	return dir
}

// getSchemaPath returns the path of the UBL 2.1 XSD bundle in test/data.
func getSchemaPath() string {
	return filepath.Join(getDataPath(), "schema")
}

// maindocXSD maps a UBL root element local name to its maindoc schema file.
func maindocXSD(rootLocal string) string {
	switch rootLocal {
	case "Invoice":
		return filepath.Join(getSchemaPath(), "maindoc", "UBL-Invoice-2.1.xsd")
	case "CreditNote":
		return filepath.Join(getSchemaPath(), "maindoc", "UBL-CreditNote-2.1.xsd")
	default:
		return ""
	}
}

// rootElementLocal returns the local name of the root element of an XML file.
func rootElementLocal(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close() //nolint:errcheck
	dec := xml.NewDecoder(f)
	for {
		tok, err := dec.Token()
		require.NoError(t, err, "no root element found in %s", path)
		if se, ok := tok.(xml.StartElement); ok {
			return se.Name.Local
		}
	}
}

// validateXSD runs the compiled Java validator for the given XML files against
// one maindoc schema, returning the combined output and whether every file was
// schema-valid.
func validateXSD(t *testing.T, classDir, xsdPath string, xmlPaths ...string) (string, bool) {
	t.Helper()
	args := append([]string{"-cp", classDir, "Validate", xsdPath}, xmlPaths...)
	out, err := exec.Command("java", args...).CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr, "java validator did not run: %v: %s", err, out)
		return string(out), false
	}
	return string(out), true
}

// TestXSDValidateConvertGoldens validates every generated golden under
// test/data/convert/*/out against the stock UBL 2.1 maindoc XSDs, choosing
// the Invoice or CreditNote schema by the root element of each file. This
// pins down real schema validity — element order included — which the
// byte-comparison golden tests alone cannot guarantee.
func TestXSDValidateConvertGoldens(t *testing.T) {
	classDir := compileXSDValidator(t)

	files, err := filepath.Glob(filepath.Join(getConvertPath(), "*", "out", "*.xml"))
	require.NoError(t, err)
	require.NotEmpty(t, files, "no convert goldens found")

	// Group files by root element so each schema is compiled once per group.
	groups := make(map[string][]string)
	for _, f := range files {
		root := rootElementLocal(t, f)
		require.NotEmpty(t, maindocXSD(root), "unexpected root element %q in %s", root, f)
		groups[root] = append(groups[root], f)
	}

	for root, group := range groups {
		t.Run(root, func(t *testing.T) {
			out, ok := validateXSD(t, classDir, maindocXSD(root), group...)
			if !ok {
				for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
					if strings.HasPrefix(line, "INVALID: ") {
						assert.Fail(t, "golden is not schema-valid", line)
					}
				}
			}
		})
	}
}

// TestCreditNoteXSDOrderingDifferential proves that mapping credit notes onto
// the dedicated CreditNote layout is what makes the output schema-valid. The
// same *ubl.Invoice is marshalled twice: through ubl.Bytes (the shipped path,
// which remaps to the CreditNote element sequence) and through encoding/xml
// directly on the Invoice struct (reproducing the old invoice-ordered
// behavior). The former must pass UBL-CreditNote-2.1.xsd and the latter must
// fail it, because the invoice layout emits cbc:TaxPointDate after
// cbc:CreditNoteTypeCode while the CreditNote XSD sequence requires the
// reverse.
func TestCreditNoteXSDOrderingDifferential(t *testing.T) {
	classDir := compileXSDValidator(t)
	cnXSD := maindocXSD("CreditNote")

	eur := "EUR"
	inv := &ubl.Invoice{XMLName: xml.Name{Local: "CreditNote"}}
	inv.CACNamespace = ubl.NamespaceCAC
	inv.CBCNamespace = ubl.NamespaceCBC
	inv.QDTNamespace = ubl.NamespaceQDT
	inv.UDTNamespace = ubl.NamespaceUDT
	inv.CCTSNamespace = ubl.NamespaceCCTS
	inv.UBLNamespace = ubl.NamespaceUBLCreditNote
	inv.XSINamespace = ubl.NamespaceXSI
	inv.EXTNamespace = ubl.NamespaceEXT
	inv.ID = "CN-XSD-1"
	inv.IssueDate = "2024-02-14"

	// The ordering divergence under test: the CreditNote XSD places
	// cbc:TaxPointDate immediately before cbc:CreditNoteTypeCode, while the
	// Invoice layout emits the type code first.
	inv.TaxPointDate = "2024-02-10"
	inv.CreditNoteTypeCode = &ubl.IDType{Value: "381"}

	inv.DocumentCurrencyCode = "EUR"
	inv.AllowanceCharge = []ubl.AllowanceCharge{{
		ChargeIndicator: true,
		Amount:          ubl.Amount{CurrencyID: &eur, Value: "10.00"},
	}}
	inv.TaxTotal = []ubl.TaxTotal{{
		TaxAmount: ubl.Amount{CurrencyID: &eur, Value: "19.00"},
	}}
	inv.LegalMonetaryTotal = ubl.MonetaryTotal{
		LineExtensionAmount: ubl.Amount{CurrencyID: &eur, Value: "100.00"},
		TaxExclusiveAmount:  ubl.Amount{CurrencyID: &eur, Value: "110.00"},
		TaxInclusiveAmount:  ubl.Amount{CurrencyID: &eur, Value: "129.00"},
		PayableAmount:       &ubl.Amount{CurrencyID: &eur, Value: "129.00"},
	}
	inv.CreditNoteLines = []ubl.InvoiceLine{{
		ID:                  "1",
		CreditedQuantity:    &ubl.Quantity{UnitCode: "C62", Value: "1"},
		LineExtensionAmount: ubl.Amount{CurrencyID: &eur, Value: "100.00"},
		Item:                &ubl.Item{Name: "Development services"},
	}}

	// (a) The shipped path: remapped onto the CreditNote layout.
	fixed, err := ubl.Bytes(inv)
	require.NoError(t, err)

	// (b) The old behavior: marshalling the Invoice struct directly emits the
	// invoice element sequence under a CreditNote root.
	legacy, err := xml.MarshalIndent(inv, "", "  ")
	require.NoError(t, err)
	legacy = append([]byte(xml.Header), legacy...)

	dir := t.TempDir()
	fixedPath := filepath.Join(dir, "creditnote-fixed.xml")
	legacyPath := filepath.Join(dir, "creditnote-legacy.xml")
	require.NoError(t, os.WriteFile(fixedPath, fixed, 0o600))
	require.NoError(t, os.WriteFile(legacyPath, legacy, 0o600))

	out, ok := validateXSD(t, classDir, cnXSD, fixedPath)
	assert.True(t, ok, "credit note marshalled via ubl.Bytes must be schema-valid: %s", out)

	out, ok = validateXSD(t, classDir, cnXSD, legacyPath)
	assert.False(t, ok, "invoice-ordered marshalling must violate the CreditNote XSD: %s", out)
	assert.Contains(t, out, "TaxPointDate",
		"the schema violation should be the misplaced cbc:TaxPointDate: %s", out)
}
