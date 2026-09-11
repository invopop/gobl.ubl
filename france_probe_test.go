package ubl_test

import (
	"path/filepath"
	"strings"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
)

// franceProbe pins one Flow 2 invoice/credit-note fixture to the context
// whose dedicated 1.3.1 French CTC schematron the generated UBL must
// satisfy with zero errors AND zero warnings (warnings become errors in
// future schematron releases). Mirrors the gobl.cii probe harness.
//
// NOTE: the pre-existing france-extended/invoice-fr-extended*.json fixtures
// are intentionally NOT listed here. They are stale facturx-addon documents
// that predate the Flow 2 BR-FR rules (no SIREN identities, no PMT/PMD/AAB
// notes, no fr-ctc-billing-mode), so they raise French-schematron warnings.
// The gap is in the `extended` UBL context declaring the `facturx.V1` addon
// rather than `flow2` (CIUS uses flow2 and normalises these in); migrating
// the extended context — or upgrading those fixtures — is left to follow-up
// (the issue scopes extended as best-effort). They still pass the
// errors-only TestConvertToInvoice gate.
type franceProbe struct {
	name    string
	dir     string
	file    string
	context ubl.Context
}

var franceProbes = []franceProbe{
	{"CIUS/standard", "france-cius", "invoice-standard.json", ubl.ContextPeppolFranceCIUS},
	{"CIUS/credit-note", "france-cius", "credit-note.json", ubl.ContextPeppolFranceCIUS},
	{"CIUS/existing-invoice", "france-cius", "invoice-fr-cius.json", ubl.ContextPeppolFranceCIUS},
	{"CIUS/existing-credit-note", "france-cius", "credit-note-fr.json", ubl.ContextPeppolFranceCIUS},
	{"Extended/standard", "france-extended", "invoice-standard.json", ubl.ContextPeppolFranceExtended},
	{"Extended/credit-note", "france-extended", "credit-note.json", ubl.ContextPeppolFranceExtended},
	{"Extended/payer", "france-extended", "invoice-payer.json", ubl.ContextPeppolFranceExtended},
}

// TestProbeFranceInvoices converts each Flow 2 fixture and pushes the
// generated UBL through phorm against the per-document-type French VESID,
// failing on any error or warning.
func TestProbeFranceInvoices(t *testing.T) {
	pc := phormClient(t)

	for _, p := range franceProbes {
		t.Run(p.name, func(t *testing.T) {
			env, err := loadTestEnvelopeFromPath(filepath.Join(getConvertPath(), p.dir, p.file))
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			inv, ok := env.Extract().(*bill.Invoice)
			if !ok {
				t.Fatalf("fixture is not an invoice")
			}
			vesid := p.context.GetVESID(inv)

			doc, err := ubl.ConvertInvoice(env, ubl.WithContext(p.context))
			if err != nil {
				t.Fatalf("ConvertInvoice: %v", err)
			}
			data, err := ubl.Bytes(doc)
			if err != nil {
				t.Fatalf("Bytes: %v", err)
			}
			problems := phormValidate(t, pc, vesid, data)
			lines := make([]string, 0, len(problems))
			for _, f := range problems {
				lines = append(lines, f.String())
			}
			if len(problems) > 0 {
				t.Errorf("[%s] %s: %d problem(s) (warnings are treated as errors):\n%s",
					p.name, vesid, len(lines), strings.Join(lines, "\n\n"))
			}
		})
	}
}
