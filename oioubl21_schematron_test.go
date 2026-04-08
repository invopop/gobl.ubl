package ubl_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/stretchr/testify/require"
)

func TestOIOUBL21Schematron(t *testing.T) {
	requireSaxonAndSchematron(t, oioubl21InvoiceSchematronPath())
	requireSaxonAndSchematron(t, oioubl21CreditNoteSchematronPath())

	fixtures := []string{
		"oioubl21-invoice-minimal.json",
		"oioubl21-credit-note-minimal.json",
		"oioubl30-invoice-minimal.json",
		"oioubl30-invoice-example.json",
	}
	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			xmlData := renderOIOUBL21Fixture(t, fixture)
			xsl := oioubl21InvoiceSchematronPath()
			if strings.Contains(string(xmlData), "<CreditNote ") {
				xsl = oioubl21CreditNoteSchematronPath()
			}
			svrl := runSchematron(t, fixture, xsl, xmlData)
			assertNoFindings(t, svrl)
		})
	}
}

func TestOIOUBL21SchematronNegativeInvoiceEndpoint(t *testing.T) {
	requireSaxonAndSchematron(t, oioubl21InvoiceSchematronPath())
	xmlData := renderOIOUBL21Fixture(t, "oioubl21-invoice-minimal.json")
	bad := strings.Replace(
		string(xmlData),
		`<cbc:EndpointID schemeID="GLN">5790000436101</cbc:EndpointID>`,
		`<cbc:EndpointID schemeID="GLN"></cbc:EndpointID>`,
		1,
	)
	svrl := runSchematron(t, "oioubl21-invoice-minimal-negative-endpoint.json", oioubl21InvoiceSchematronPath(), []byte(bad))
	require.Contains(t, svrl, "[F-INV031]", "expected invoice endpoint validation failure code")
}

func TestOIOUBL21SchematronNegativeCreditNoteEndpoint(t *testing.T) {
	requireSaxonAndSchematron(t, oioubl21CreditNoteSchematronPath())
	xmlData := renderOIOUBL21Fixture(t, "oioubl21-credit-note-minimal.json")
	bad := strings.Replace(
		string(xmlData),
		`<cbc:EndpointID schemeID="GLN">5790000436057</cbc:EndpointID>`,
		`<cbc:EndpointID schemeID="GLN"></cbc:EndpointID>`,
		1,
	)
	svrl := runSchematron(t, "oioubl21-credit-note-negative-endpoint.json", oioubl21CreditNoteSchematronPath(), []byte(bad))
	require.Contains(t, svrl, "[F-CRN040]", "expected credit note endpoint validation failure code")
}

func requireSaxonAndSchematron(t *testing.T, xsl string) {
	t.Helper()
	if _, err := exec.LookPath("saxon"); err != nil {
		t.Skip("saxon not installed; skipping OIOUBL 2.1 schematron validation")
	}
	if _, err := os.Stat(xsl); err != nil {
		t.Skipf("OIOUBL 2.1 schematron not found at %s", xsl)
	}
}

func renderOIOUBL21Fixture(t *testing.T, fixture string) []byte {
	t.Helper()
	env, err := loadTestEnvelope(fixture)
	require.NoError(t, err)
	doc, err := ubl.ConvertInvoice(env, ubl.WithContext(ubl.ContextOIOUBL21))
	require.NoError(t, err)
	xmlData, err := ubl.Bytes(doc)
	require.NoError(t, err)
	return xmlData
}

func runSchematron(t *testing.T, name, xsl string, xmlData []byte) string {
	t.Helper()
	tmp := t.TempDir()
	base := strings.TrimSuffix(name, filepath.Ext(name))
	xmlPath := filepath.Join(tmp, fmt.Sprintf("%s.xml", base))
	svrlPath := filepath.Join(tmp, fmt.Sprintf("%s.svrl.xml", base))
	require.NoError(t, os.WriteFile(xmlPath, xmlData, 0o644))

	cmd := exec.Command("saxon",
		"-xsl:"+xsl,
		"-s:"+xmlPath,
		"-o:"+svrlPath,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "saxon failed: %s", string(out))

	svrl, err := os.ReadFile(svrlPath)
	require.NoError(t, err)
	return string(svrl)
}

func assertNoFindings(t *testing.T, svrl string) {
	t.Helper()
	require.NotContains(t, svrl, "failed-assert", "schematron failed assertions")
	require.NotContains(t, svrl, "[F-", "schematron fatal findings present")
	require.NotContains(t, svrl, "[E-", "schematron error findings present")
	require.NotContains(t, svrl, "[W-", "schematron warning findings present")
}

func oioubl21InvoiceSchematronPath() string {
	if p := os.Getenv("OIOUBL21_INVOICE_SCHEMATRON_XSL"); p != "" {
		return p
	}
	if p := os.Getenv("OIOUBL21_SCHEMATRON_XSL"); p != "" {
		return p
	}
	return filepath.Join(filepath.Dir(getRootFolder()), "OIOUBL Schematron", "OIOUBL_Invoice_Schematron.xsl")
}

func oioubl21CreditNoteSchematronPath() string {
	if p := os.Getenv("OIOUBL21_CREDITNOTE_SCHEMATRON_XSL"); p != "" {
		return p
	}
	return filepath.Join(filepath.Dir(getRootFolder()), "OIOUBL Schematron", "OIOUBL_CreditNote_Schematron.xsl")
}
