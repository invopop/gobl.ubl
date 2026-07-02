package ubl_test

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreditNoteMarshalOrdering verifies that a credit note is marshalled in the
// element sequence mandated by the UBL-CreditNote-2.1 XSD, which diverges from
// the Invoice XSD in several places, and that invoice-only elements never leak
// into a credit note. Credit notes are built as an Invoice and mapped onto the
// CreditNote layout at the marshalling boundary, so this exercises that mapping
// end to end — no post-marshal byte surgery involved.
func TestCreditNoteMarshalOrdering(t *testing.T) {
	inv := &ubl.Invoice{XMLName: xml.Name{Local: "CreditNote"}}
	inv.CACNamespace = ubl.NamespaceCAC
	inv.CBCNamespace = ubl.NamespaceCBC
	inv.QDTNamespace = ubl.NamespaceQDT
	inv.UDTNamespace = ubl.NamespaceUDT
	inv.CCTSNamespace = ubl.NamespaceCCTS
	inv.UBLNamespace = ubl.NamespaceUBLCreditNote
	inv.XSINamespace = ubl.NamespaceXSI
	inv.EXTNamespace = ubl.NamespaceEXT
	inv.ID = "CN-1"
	inv.IssueDate = "2024-02-14"

	// Head divergence: TaxPointDate precedes the type code in a credit note.
	inv.TaxPointDate = "2024-02-10"
	inv.CreditNoteTypeCode = &ubl.IDType{Value: "381"}

	// Invoice-only head fields that must be dropped.
	inv.DueDate = "2024-03-15"
	inv.InvoiceTypeCode = &ubl.IDType{Value: "380"}

	// Reference block: the credit note orders Contract/Additional ahead of
	// Statement/Originator (the invoice does the reverse).
	inv.ContractDocumentReference = []ubl.Reference{{ID: ubl.IDType{Value: "CONTRACT-1"}}}
	inv.StatementDocumentReference = []ubl.Reference{{ID: ubl.IDType{Value: "STATEMENT-1"}}}
	inv.OriginatorDocumentReference = []ubl.Reference{{ID: ubl.IDType{Value: "ORIGINATOR-1"}}}

	// Invoice-only reference that must be dropped.
	inv.ProjectReference = []ubl.ProjectReference{{ID: "PROJECT-1"}}

	// Tail divergence: AllowanceCharge follows the exchange rates in a credit
	// note (it precedes them in an invoice).
	inv.TaxExchangeRate = &ubl.ExchangeRate{}
	inv.AllowanceCharge = []ubl.AllowanceCharge{{ChargeIndicator: true, Amount: ubl.Amount{Value: "10.00"}}}
	inv.TaxTotal = []ubl.TaxTotal{{TaxAmount: ubl.Amount{Value: "5.00"}}}
	inv.LegalMonetaryTotal = ubl.MonetaryTotal{LineExtensionAmount: ubl.Amount{Value: "100.00"}}
	inv.CreditNoteLines = []ubl.InvoiceLine{{ID: "1", LineExtensionAmount: ubl.Amount{Value: "100.00"}}}

	// Invoice-only tail fields that must be dropped.
	inv.PrepaidPayment = []ubl.PrepaidPayment{{ID: "PREPAID-1"}}
	inv.WithholdingTaxTotal = []ubl.TaxTotal{{TaxAmount: ubl.Amount{Value: "1.00"}}}

	data, err := ubl.Bytes(inv)
	require.NoError(t, err)
	out := string(data)

	// The output must be well-formed XML with all prefixes bound.
	dec := xml.NewDecoder(strings.NewReader(out))
	for {
		_, err := dec.Token()
		if err == io.EOF {
			break
		}
		require.NoError(t, err, "marshalled credit note must be well-formed XML")
	}

	// Root element is a CreditNote.
	assert.Contains(t, out, "<CreditNote")

	// Element order must follow the CreditNote XSD sequence.
	assertOrder(t, out,
		"<cbc:TaxPointDate>",
		"<cbc:CreditNoteTypeCode>",
		"<cac:ContractDocumentReference>",
		"<cac:StatementDocumentReference>",
		"<cac:OriginatorDocumentReference>",
		"<cac:TaxExchangeRate>",
		"<cac:AllowanceCharge>",
		"<cac:TaxTotal>",
		"<cac:LegalMonetaryTotal>",
		"<cac:CreditNoteLine>",
	)

	// Invoice-only elements must never appear in a credit note.
	for _, forbidden := range []string{
		"<cbc:DueDate>",
		"<cbc:InvoiceTypeCode>",
		"<cac:ProjectReference>",
		"<cac:PrepaidPayment>",
		"<cac:WithholdingTaxTotal>",
		"<cac:InvoiceLine>",
	} {
		assert.NotContainsf(t, out, forbidden, "credit note must not contain %s", forbidden)
	}
}

// assertOrder asserts that each needle appears in s, in the given order.
func assertOrder(t *testing.T, s string, needles ...string) {
	t.Helper()
	prev := -1
	prevNeedle := ""
	for _, n := range needles {
		idx := strings.Index(s, n)
		require.GreaterOrEqualf(t, idx, 0, "expected output to contain %s", n)
		assert.Greaterf(t, idx, prev, "%s must appear after %s", n, prevNeedle)
		prev = idx
		prevNeedle = n
	}
}
