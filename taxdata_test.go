package ubl_test

import (
	"encoding/xml"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ubl "github.com/invopop/gobl.ubl"
)

func taxDataParams(profile ubl.TaxDataProfile) ubl.TaxDataParams {
	p := ubl.TaxDataParams{
		TypeCode:         ubl.TaxDataTypeSubmit,
		Scope:            ubl.TaxDataScopeDomestic,
		ReporterRole:     profile.SenderRole,
		TaxAuthorityID:   "TEST-TAX",
		TaxAuthorityName: "Test Tax Authority",
		ReceiverEndpoint: "000000-TaxAuthority.TEST",
		Representative:   "000770",
		UUID:             "123e4567-e89b-42d3-a456-426614174000",
		IssuedAt:         time.Date(2027, 1, 15, 10, 30, 0, 0, time.UTC),
	}
	if profile.TransmissionUUID {
		p.TransmissionUUID = "d8269130-a816-4ab4-b2b3-089456b179e2"
	}
	return p
}

func TestTaxDataInvoiceUUID(t *testing.T) {
	// ViDA TDD BIS 1.1.0, section 2.2.5.
	vida := []struct {
		vat, tc, num, date, want string
	}{
		{"DE811569869", "380", "INV-2025-0046", "2025-02-10", "62fd54e5-8689-57a6-95a7-600c86734ea0"},
		{"BE0477472701", "380", "INV-2025-0001", "2025-01-10", "81270c84-bfb1-5a12-8314-8887ee7d3e27"},
		{"DE811569869", "389", "INV-2025-0046", "2025-02-10", "936d1327-0863-5632-ac82-30001c004e01"},
		{"FR40303265045", "261", "CN-2025-0003", "2025-02-05", "9a787cd0-ef99-5bd1-ad5d-daf020699cf2"},
	}
	for _, v := range vida {
		name, err := ubl.TaxDataInvoiceUUIDName(v.vat, v.tc, v.num, v.date)
		require.NoError(t, err)
		assert.Equal(t, v.want, ubl.TaxDataInvoiceUUID(name), v.num)
	}

	t.Run("inner whitespace is kept", func(t *testing.T) {
		a, _ := ubl.TaxDataInvoiceUUIDName("DE811569869", "380", "INV  1", "2025-02-10")
		b, _ := ubl.TaxDataInvoiceUUIDName("DE811569869", "380", "INV 1", "2025-02-10")
		assert.NotEqual(t, ubl.TaxDataInvoiceUUID(a), ubl.TaxDataInvoiceUUID(b))
	})

	t.Run("empty value is not reportable", func(t *testing.T) {
		_, err := ubl.TaxDataInvoiceUUIDName(" ", "380", "INV-1", "2025-02-10")
		assert.ErrorIs(t, err, ubl.ErrTaxDataNotReportable)
	})
}

func TestTaxDataParams(t *testing.T) {
	env, err := loadTestEnvelopeFromPath(filepath.Join(getConvertPath(), "peppol", "invoice-complete.json"))
	require.NoError(t, err)

	p := taxDataParams(ubl.TaxDataProfileViDA)
	p.TypeCode = "X"
	p.TransmissionUUID = ""
	_, err = ubl.ExportTaxData(env, ubl.TaxDataProfileViDA, p)
	require.ErrorIs(t, err, ubl.ErrTaxDataParams)
	assert.ErrorContains(t, err, "type code")
	assert.ErrorContains(t, err, "transmission UUID")
}

// TestBuildTaxData builds the TDD from the UBL XML of the invoice, as the
// receiver's provider does.
func TestBuildTaxData(t *testing.T) {
	env := envFrom("peppol/invoice-complete.json")(t)
	doc, err := ubl.Export(env, ubl.WithFormat(ubl.FormatPeppol))
	require.NoError(t, err)
	inv, err := ubl.Encode(doc)
	require.NoError(t, err)

	td, err := ubl.BuildTaxData(inv, ubl.TaxDataProfileViDA, taxDataParams(ubl.TaxDataProfileViDA))
	require.NoError(t, err)
	data, err := ubl.Encode(td)
	require.NoError(t, err)
	expected, err := os.ReadFile(filepath.Join(getDataPath(), "taxdata", "vida-invoice.xml"))
	require.NoError(t, err)
	assert.Equal(t, string(expected), string(data))

	t.Run("unreadable invoice", func(t *testing.T) {
		_, err := ubl.BuildTaxData([]byte("<Invoice>broken"), ubl.TaxDataProfileViDA, taxDataParams(ubl.TaxDataProfileViDA))
		assert.Error(t, err)
	})
}

func TestExportTaxData(t *testing.T) {
	tests := []struct {
		name    string
		env     func(t *testing.T) *gobl.Envelope
		profile ubl.TaxDataProfile
		role    string
		out     string
	}{
		{"vida invoice", envFrom("peppol/invoice-complete.json"), ubl.TaxDataProfileViDA, ubl.TaxDataReporterSender, "vida-invoice.xml"},
		{"vida credit note", envFrom("peppol/credit-note.json"), ubl.TaxDataProfileViDA, ubl.TaxDataReporterSender, "vida-credit-note.xml"},
		{"vida receiver", envFrom("peppol/invoice-complete.json"), ubl.TaxDataProfileViDA, ubl.TaxDataReporterReceiver, "vida-invoice-c3.xml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := taxDataParams(tt.profile)
			p.ReporterRole = tt.role
			td, err := ubl.ExportTaxData(tt.env(t), tt.profile, p)
			require.NoError(t, err)

			data, err := ubl.Encode(td)
			require.NoError(t, err)
			checkTaxDataElements(t, data, tt.profile)

			outPath := filepath.Join(getDataPath(), "taxdata", tt.out)
			if *updateOut {
				require.NoError(t, os.WriteFile(outPath, data, 0644))
			}
			expected, err := os.ReadFile(outPath)
			require.NoError(t, err)
			assert.Equal(t, string(expected), string(data), "Update with --update flag.")
		})
	}

	t.Run("vida without seller VAT is not reportable", func(t *testing.T) {
		env := envFrom("peppol/invoice-complete.json")(t)
		env.Extract().(*bill.Invoice).Supplier.TaxID = nil
		_, err := ubl.ExportTaxData(env, ubl.TaxDataProfileViDA, taxDataParams(ubl.TaxDataProfileViDA))
		assert.ErrorIs(t, err, ubl.ErrTaxDataNotReportable)
	})
}

func envFrom(name string) func(t *testing.T) *gobl.Envelope {
	return func(t *testing.T) *gobl.Envelope {
		env, err := loadTestEnvelopeFromPath(filepath.Join(getConvertPath(), name))
		require.NoError(t, err)
		return env
	}
}

// taxDataAllowed lists, for each element of the reported document, the
// children the TDD Schematron allows (ViDA 1.1.0 rules
// ibr-tdd-25 to ibr-tdd-85). Elements not listed are leaves or are checked by
// the XSD only.
var taxDataAllowed = map[string]string{
	"pxs:TaxData":                 "cbc:CustomizationID cbc:ProfileID cbc:UUID cbc:IssueDate cbc:IssueTime pxs:TaxDataTypeCode pxs:DocumentScope pxs:ReporterRole pxs:TaxAuthority pxs:ReportingParty pxs:ReceivingParty pxs:ReportersRepresentative pxs:ReportedTransaction",
	"pxs:TaxAuthority":            "cbc:ID cbc:Name",
	"pxs:ReportingParty":          "cbc:EndpointID",
	"pxs:ReceivingParty":          "cbc:EndpointID",
	"pxs:ReportersRepresentative": "cac:PartyIdentification",
	"pxs:ReportedTransaction":     "pxs:TransmissionUUID pxs:ReportedDocument",
	"RD":                          "cbc:CustomizationID cbc:ProfileID cbc:ID cbc:UUID cbc:IssueDate cbc:IssueTime pxs:DocumentTypeCode cbc:Note cbc:TaxPointDate cbc:DocumentCurrencyCode cbc:TaxCurrencyCode cac:InvoicePeriod cac:BillingReference cac:AccountingSupplierParty cac:AccountingCustomerParty cac:TaxRepresentativeParty cac:Delivery cac:PaymentMeans cac:AllowanceCharge cac:TaxTotal pxs:MonetaryTotal pxs:DocumentLine",
	"RD/cac:InvoicePeriod":        "cbc:StartDate cbc:EndDate cbc:DescriptionCode",
	"RD/cac:BillingReference":     "cac:InvoiceDocumentReference",
	"RD/cac:BillingReference/cac:InvoiceDocumentReference":          "cbc:ID cbc:IssueDate",
	"RD/cac:AccountingSupplierParty/cac:Party":                      "cac:PostalAddress cac:PartyTaxScheme",
	"RD/cac:AccountingCustomerParty/cac:Party":                      "cac:PostalAddress cac:PartyTaxScheme cac:PartyLegalEntity",
	"RD/cac:AccountingCustomerParty/cac:Party/cac:PartyLegalEntity": "cbc:RegistrationName",
	"RD/cac:TaxRepresentativeParty":                                 "cac:PostalAddress cac:PartyTaxScheme",
	"RD/cac:Delivery":                                               "cbc:ActualDeliveryDate",
	"RD/cac:PaymentMeans":                                           "cbc:PaymentMeansCode cbc:PaymentID cac:CardAccount cac:PayeeFinancialAccount cac:PaymentMandate",
	"RD/cac:PaymentMeans/cac:PayeeFinancialAccount":                 "cbc:ID cac:FinancialInstitutionBranch",
	"RD/cac:AllowanceCharge":                                        "cbc:ChargeIndicator cbc:AllowanceChargeReasonCode cbc:AllowanceChargeReason cbc:MultiplierFactorNumeric cbc:Amount cbc:BaseAmount cac:TaxCategory",
	"RD/cac:AllowanceCharge/cac:TaxCategory":                        "cbc:ID cbc:Percent cac:TaxScheme",
	"RD/cac:TaxTotal":                                               "cbc:TaxAmount cac:TaxSubtotal",
	"RD/cac:TaxTotal/cac:TaxSubtotal":                               "cbc:TaxableAmount cbc:TaxAmount cac:TaxCategory",
	"RD/cac:TaxTotal/cac:TaxSubtotal/cac:TaxCategory":               "cbc:ID cbc:Percent cbc:TaxExemptionReasonCode cbc:TaxExemptionReason cac:TaxScheme",
	"RD/pxs:MonetaryTotal":                                          "cbc:LineExtensionAmount cbc:TaxExclusiveAmount cbc:TaxInclusiveAmount cbc:AllowanceTotalAmount cbc:ChargeTotalAmount cbc:PrepaidAmount cbc:PayableRoundingAmount cbc:PayableAmount",
	"RD/pxs:DocumentLine":                                           "cbc:ID cbc:Note cbc:InvoicedQuantity cbc:LineExtensionAmount cac:InvoicePeriod cac:AllowanceCharge cac:Item cac:Price",
	"RD/pxs:DocumentLine/cac:InvoicePeriod":                         "cbc:StartDate cbc:EndDate",
	"RD/pxs:DocumentLine/cac:AllowanceCharge":                       "cbc:ChargeIndicator cbc:AllowanceChargeReasonCode cbc:AllowanceChargeReason cbc:MultiplierFactorNumeric cbc:Amount cbc:BaseAmount",
	"RD/pxs:DocumentLine/cac:Item":                                  "cbc:Description cbc:Name cac:CommodityClassification cac:ClassifiedTaxCategory",
	"RD/pxs:DocumentLine/cac:Item/cac:ClassifiedTaxCategory":        "cbc:ID cbc:Percent cac:TaxScheme",
	"RD/pxs:DocumentLine/cac:Price":                                 "cbc:PriceAmount cbc:BaseQuantity",
}

var taxDataPrefixes = map[string]string{
	ubl.NamespaceCBC: "cbc",
	ubl.NamespaceCAC: "cac",
}

// checkTaxDataElements walks the document and fails on any element that the
// TDD Schematron does not allow in its parent.
func checkTaxDataElements(t *testing.T, data []byte, profile ubl.TaxDataProfile) {
	t.Helper()
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	var path []string
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		switch el := tok.(type) {
		case xml.StartElement:
			prefix := taxDataPrefixes[el.Name.Space]
			if el.Name.Space == profile.Namespace {
				prefix = "pxs"
			}
			require.NotEmpty(t, prefix, "unknown namespace %s", el.Name.Space)
			name := prefix + ":" + el.Name.Local
			if len(path) > 0 {
				parent := strings.Join(path, "/")
				parent = strings.Replace(parent, "pxs:TaxData/pxs:ReportedTransaction/pxs:ReportedDocument", "RD", 1)
				if allowed, ok := taxDataAllowed[parent]; ok {
					assert.Contains(t, strings.Fields(allowed), name, "%s not allowed in %s", name, parent)
				}
			}
			path = append(path, name)
		case xml.EndElement:
			path = path[:len(path)-1]
		}
	}
}

// TestTaxDataSchematron runs the official ViDA TDD Schematron (compiled XSLT
// 2.0 from the ViDA resource bundle) on the golden files with Saxon. It runs
// only when the XSLT path is set and the saxon command exists:
//
//	VIDA_TDD_XSLT=.../Peppol-ViDA-TDD.xslt go test -run TaxDataSchematron
func TestTaxDataSchematron(t *testing.T) {
	saxon, err := exec.LookPath("saxon")
	if err != nil {
		t.Skip("saxon not installed")
	}
	sets := []struct {
		env   string
		files []string
	}{
		{"VIDA_TDD_XSLT", []string{"vida-invoice.xml", "vida-credit-note.xml", "vida-invoice-c3.xml"}},
	}
	for _, s := range sets {
		xsl := os.Getenv(s.env)
		if xsl == "" {
			t.Logf("%s not set, skipped", s.env)
			continue
		}
		for _, f := range s.files {
			t.Run(f, func(t *testing.T) {
				out, err := exec.Command(saxon, "-s:"+filepath.Join(getDataPath(), "taxdata", f), "-xsl:"+xsl).CombinedOutput()
				require.NoError(t, err, string(out))
				assert.NotContains(t, string(out), "failed-assert", string(out))
			})
		}
	}
}
