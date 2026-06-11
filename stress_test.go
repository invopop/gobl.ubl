package ubl_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/phive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var stress = flag.Bool("stress", false, "run the OIOUBL stress harness (requires phive on :9090)")

// Text pools exercising the charsets OIOUBL documents meet in the wild.
var stressTexts = []string{
	"Søren & Møller A/S (Æblegården)",
	"Łukasz–Çağrı & <Test> \"Quote's\"",
	"日本語テスト 🚀 invoice παράδειγμα",
	"   padded   with   spaces   ",
	strings.Repeat("Veldokumenteretvarebeskrivelse", 17), // 510 chars
	"a",
}

// Amount and quantity pools stay inside GOBL's exercisable numeric domain:
// stressing beyond ~1e14 total trips silent int64 overflow inside the amount
// arithmetic (negative totals on the wire, caught only by phive F-LIB020) —
// recorded as an upstream gobl finding rather than exercised here.
var stressAmounts = []num.Amount{
	num.MakeAmount(9999999999, 2), // 99,999,999.99
	num.MakeAmount(1, 4),          // 0.0001
	num.MakeAmount(333333, 6),     // 0.333333
	num.MakeAmount(19999, 2),
}

var stressQuantities = []num.Amount{
	num.MakeAmount(1, 0),
	num.MakeAmount(123456, 3), // 123.456
	num.MakeAmount(999, 0),
	num.MakeAmount(1, 3), // 0.001
}

func stressPhive(t *testing.T) phive.ValidationServiceClient {
	t.Helper()
	conn, err := grpc.NewClient("127.0.0.1:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	return phive.NewValidationServiceClient(conn)
}

func stressValidateXML(t *testing.T, pc phive.ValidationServiceClient, vesid string, data []byte) {
	t.Helper()
	resp, err := pc.ValidateXml(context.Background(), &phive.ValidateXmlRequest{
		Vesid:      vesid,
		XmlContent: data,
	})
	require.NoError(t, err)
	if !resp.Success {
		out, _ := json.MarshalIndent(resp.Results, "", "  ")
		t.Fatalf("phive rejected the document (%s): %s", vesid, out)
	}
}

// canonicalParticipant reduces either routing form to a comparable (icd, code)
// pair so endpoint- and inbox-modelled parties compare equal across the trip.
var stressSymbolicICDs = map[string]string{"GLN": "0088", "DK:CVR": "0184", "DK:SE": "0198"}

func canonicalParticipant(p *org.Party) string {
	if p == nil {
		return ""
	}
	if ep := p.Endpoint("iso6523-actorid-upis"); ep != nil {
		rest := strings.TrimPrefix(ep.URI.Opaque(), ":")
		if icd, code, ok := strings.Cut(rest, ":"); ok {
			return icd + ":" + strings.TrimPrefix(code, "DK")
		}
		return rest
	}
	if len(p.Inboxes) > 0 {
		ib := p.Inboxes[0]
		s := ib.Scheme.String()
		if icd, ok := stressSymbolicICDs[s]; ok {
			s = icd
		}
		return s + ":" + strings.TrimPrefix(ib.Code.String(), "DK")
	}
	return ""
}

// flipParticipantForm swaps a party between the endpoint and inbox models,
// exercising both serializer paths with identical wire expectations.
func flipParticipantForm(p *org.Party) {
	if p == nil {
		return
	}
	if ep := p.Endpoint("iso6523-actorid-upis"); ep != nil {
		rest := strings.TrimPrefix(ep.URI.Opaque(), ":")
		if icd, code, ok := strings.Cut(rest, ":"); ok {
			p.Endpoints = nil
			p.Inboxes = []*org.Inbox{{Scheme: cbc.Code(icd), Code: cbc.Code(code)}}
		}
		return
	}
	if len(p.Inboxes) > 0 && p.Inboxes[0].Scheme != "" {
		ib := p.Inboxes[0]
		icd := ib.Scheme.String()
		if mapped, ok := stressSymbolicICDs[icd]; ok {
			icd = mapped
		}
		p.Endpoints = []*org.Endpoint{{URI: cbc.URI("iso6523-actorid-upis::" + icd + ":" + ib.Code.String())}}
		p.Inboxes = p.Inboxes[1:]
	}
}

type stressMutation struct {
	name  string
	apply func(rng *rand.Rand, inv *bill.Invoice)
}

var stressMutations = []stressMutation{
	{"unicode-text", func(rng *rand.Rand, inv *bill.Invoice) {
		inv.Supplier.Name = stressTexts[rng.Intn(len(stressTexts))]
		if inv.Customer != nil {
			inv.Customer.Name = stressTexts[rng.Intn(len(stressTexts))]
		}
		for _, l := range inv.Lines {
			l.Item.Name = stressTexts[rng.Intn(len(stressTexts))]
		}
	}},
	{"extreme-amounts", func(rng *rand.Rand, inv *bill.Invoice) {
		for _, l := range inv.Lines {
			a := stressAmounts[rng.Intn(len(stressAmounts))]
			l.Item.Price = &a
			l.Quantity = stressQuantities[rng.Intn(len(stressQuantities))]
		}
	}},
	{"many-lines", func(rng *rand.Rand, inv *bill.Invoice) {
		base := inv.Lines[0]
		for i := 0; i < 30; i++ {
			cp := *base
			item := *base.Item
			a := stressAmounts[rng.Intn(len(stressAmounts))]
			item.Price = &a
			item.Name = fmt.Sprintf("%s #%d", stressTexts[rng.Intn(len(stressTexts))], i)
			cp.Item = &item
			cp.Quantity = stressQuantities[rng.Intn(len(stressQuantities))]
			inv.Lines = append(inv.Lines, &cp)
		}
	}},
	{"participant-flip", func(_ *rand.Rand, inv *bill.Invoice) {
		flipParticipantForm(inv.Supplier)
		flipParticipantForm(inv.Customer)
	}},
	{"series-code", func(rng *rand.Rand, inv *bill.Invoice) {
		inv.Series = cbc.Code(fmt.Sprintf("STRESS-%d", rng.Intn(1000)))
		inv.Code = cbc.Code(fmt.Sprintf("%08d", rng.Intn(100000000)))
	}},
}

// TestStressOIOUBLInvoices mutates every stored convert fixture and pushes the
// result through the full pipeline: Calculate -> Validate -> Convert -> phive
// schematron -> Parse -> Recalculate -> invariant comparison, plus determinism
// and idempotency checks. Reproducible: each case derives its seed from the
// fixture name and mutation index.
func TestStressOIOUBLInvoices(t *testing.T) {
	if !*stress {
		t.Skip("run with -stress")
	}
	pc := stressPhive(t)

	fixtures, err := filepath.Glob("test/data/convert/oioubl21/*.json")
	require.NoError(t, err)
	require.NotEmpty(t, fixtures)

	for _, fx := range fixtures {
		for mi, mut := range stressMutations {
			name := fmt.Sprintf("%s/%s", filepath.Base(fx), mut.name)
			t.Run(name, func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("PANIC: %v\n%s", r, debug.Stack())
					}
				}()
				seed := int64(len(fx)*1000 + mi)
				rng := rand.New(rand.NewSource(seed))

				data, err := os.ReadFile(fx)
				require.NoError(t, err)
				env := new(gobl.Envelope)
				require.NoError(t, json.Unmarshal(data, env))
				inv, ok := env.Extract().(*bill.Invoice)
				require.True(t, ok)

				mut.apply(rng, inv)

				require.NoError(t, env.Calculate(), "Calculate after mutation")
				if inv.Totals.Payable.IsNegative() || (inv.Totals.Due != nil && inv.Totals.Due.IsNegative()) {
					// Over-discounted mutants must be stopped by the addon's
					// F-LIB016/F-LIB020 rule before they can reach the wire.
					err := env.Validate()
					require.Error(t, err, "negative totals must not validate")
					assert.Contains(t, err.Error(), "F-LIB016", "the totals rule should reject it")
					return
				}
				require.NoError(t, env.Validate(), "Validate after mutation")
				payable := inv.Totals.Payable.String()

				// Idempotency: a second Calculate must not change the result.
				require.NoError(t, env.Calculate(), "second Calculate")
				assert.Equal(t, payable, inv.Totals.Payable.String(), "Calculate must be idempotent")

				doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextOIOUBL21))
				require.NoError(t, err, "Convert")
				xml1, err := ubl.Bytes(doc)
				require.NoError(t, err)

				// Determinism: converting again must yield identical bytes.
				doc2, err := ubl.Convert(env, ubl.WithContext(ubl.ContextOIOUBL21))
				require.NoError(t, err)
				xml2, err := ubl.Bytes(doc2)
				require.NoError(t, err)
				assert.True(t, bytes.Equal(xml1, xml2), "Convert must be deterministic")

				vesid := ubl.ContextOIOUBL21.VESIDs.Invoice
				if inv.Type == bill.InvoiceTypeCreditNote {
					vesid = ubl.ContextOIOUBL21.VESIDs.CreditNote
				}
				stressValidateXML(t, pc, vesid, xml1)

				parsed, err := ubl.Parse(xml1)
				require.NoError(t, err, "Parse")
				wire, ok := parsed.(*ubl.Invoice)
				require.True(t, ok, "parsed document should be a UBL invoice")
				env2, err := wire.Convert()
				require.NoError(t, err, "Convert parsed invoice to GOBL")
				inv2, ok := env2.Extract().(*bill.Invoice)
				require.True(t, ok, "reconstructed document should be an invoice")
				require.NoError(t, env2.Calculate(), "Recalculate parsed invoice")

				// The wire carries 2-decimal amounts while GOBL recomputes
				// percentage allowances and fractional quantities at full
				// precision, so a reconstructed document may legitimately land
				// one cent away (the schematron itself allows +/-1.00,
				// F-LIB401/402). Anything beyond a cent is a mapping bug.
				drift := inv.Totals.Payable.Subtract(inv2.Totals.Payable).Abs()
				tolerance := num.MakeAmount(int64(len(inv.Lines)), 2) // one cent per line
				assert.LessOrEqual(t, drift.Compare(tolerance), 0,
					"payable must survive the round trip within a cent per line (was %s, got %s)",
					payable, inv2.Totals.Payable.String())
				assert.Len(t, inv2.Lines, len(inv.Lines), "line count must survive")
				for i := range inv.Lines {
					assert.Zero(t, inv.Lines[i].Quantity.Compare(inv2.Lines[i].Quantity),
						"line %d quantity must survive", i)
				}
				assert.Equal(t, canonicalParticipant(inv.Supplier), canonicalParticipant(inv2.Supplier),
					"supplier participant must survive")
				assert.Equal(t, canonicalParticipant(inv.Customer), canonicalParticipant(inv2.Customer),
					"customer participant must survive")
				assert.Equal(t, inv.Currency, inv2.Currency, "currency must survive")
			})
		}
	}
}

// TestStressOIOUBLResponses runs the same gauntlet over the ApplicationResponse
// fixtures across every supported status event.
func TestStressOIOUBLResponses(t *testing.T) {
	if !*stress {
		t.Skip("run with -stress")
	}
	pc := stressPhive(t)

	fixtures, err := filepath.Glob("test/data/convert/oioubl21-response/*.json")
	require.NoError(t, err)
	require.NotEmpty(t, fixtures)

	events := []cbc.Key{bill.StatusLineAccepted, bill.StatusLineRejected, bill.StatusLineAcknowledged, bill.StatusLineError}

	for _, fx := range fixtures {
		for ei, event := range events {
			for _, flip := range []bool{false, true} {
				name := fmt.Sprintf("%s/%s/flip=%v", filepath.Base(fx), event, flip)
				t.Run(name, func(t *testing.T) {
					defer func() {
						if r := recover(); r != nil {
							t.Fatalf("PANIC: %v\n%s", r, debug.Stack())
						}
					}()
					rng := rand.New(rand.NewSource(int64(len(fx)*100 + ei)))

					data, err := os.ReadFile(fx)
					require.NoError(t, err)
					env := new(gobl.Envelope)
					require.NoError(t, json.Unmarshal(data, env))
					st, ok := env.Extract().(*bill.Status)
					require.True(t, ok)

					st.Lines[0].Key = event
					st.Lines[0].Ext = tax.Extensions{} // force the normalizer to derive the wire code
					st.Lines[0].Description = stressTexts[rng.Intn(len(stressTexts))]
					if flip {
						flipParticipantForm(st.Supplier)
						flipParticipantForm(st.Customer)
					}

					require.NoError(t, env.Calculate(), "Calculate after mutation")
					require.NoError(t, env.Validate(), "Validate after mutation")

					doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextOIOUBL21))
					require.NoError(t, err, "Convert")
					xml1, err := ubl.Bytes(doc)
					require.NoError(t, err)

					stressValidateXML(t, pc, ubl.ContextOIOUBL21.VESIDs.ApplicationResponse, xml1)

					parsed, err := ubl.Parse(xml1)
					require.NoError(t, err, "Parse")
					wire, ok := parsed.(*ubl.ApplicationResponse)
					require.True(t, ok, "parsed document should be a UBL application response")
					env2, err := wire.Convert()
					require.NoError(t, err, "Convert parsed response to GOBL")
					st2, ok := env2.Extract().(*bill.Status)
					require.True(t, ok, "reconstructed document should be a status")
					require.NoError(t, env2.Calculate(), "Recalculate parsed status")

					assert.Equal(t, event, st2.Lines[0].Key, "status event must survive the round trip")
					assert.Equal(t, canonicalParticipant(st.Supplier), canonicalParticipant(st2.Supplier),
						"supplier participant must survive")
					assert.Equal(t, canonicalParticipant(st.Customer), canonicalParticipant(st2.Customer),
						"customer participant must survive")
				})
			}
		}
	}
}

// FuzzParseOIOUBL feeds the parser arbitrary bytes, seeded with every stored
// OIOUBL document: Parse must return an error, never panic or hang.
func FuzzParseOIOUBL(f *testing.F) {
	for _, dir := range []string{
		"test/data/convert/oioubl21/out",
		"test/data/convert/oioubl21-response/out",
		"test/data/parse/oioubl21",
		"test/data/parse/oioubl21-response",
	} {
		files, err := filepath.Glob(filepath.Join(dir, "*.xml"))
		if err != nil {
			f.Fatal(err)
		}
		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				f.Fatal(err)
			}
			f.Add(data)
		}
	}
	f.Fuzz(func(_ *testing.T, data []byte) {
		_, _ = ubl.Parse(data) //nolint:errcheck // errors are expected; panics are the failure mode
	})
}
