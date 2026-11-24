package ubl_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/schema"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testCustomizationID = "urn:example:funcs-test"
	testProfileID       = "funcs-test"
)

// testExport records the order it is applied in through the invoice notes,
// and marks every party.
func testExport(name string) ubl.ExportFunc {
	return func(_ *ubl.Format, env *gobl.Envelope, doc ubl.Document) error {
		inv, ok := env.Extract().(*bill.Invoice)
		out, ok2 := doc.(*ubl.Invoice)
		if !ok || !ok2 {
			return nil
		}
		out.Note = append(out.Note, name)
		for _, p := range ubl.InvoiceParties(inv, out) {
			p.UBL.IndustryClassificationCode += name
		}
		return nil
	}
}

// testImport records the order it is applied in through the invoice code,
// and marks every party.
func testImport(name string) ubl.ImportFunc {
	return func(_ *ubl.Format, doc ubl.Document, env *gobl.Envelope) error {
		in, ok := doc.(*ubl.Invoice)
		inv, ok2 := env.Extract().(*bill.Invoice)
		if !ok || !ok2 {
			return nil
		}
		inv.Code = cbc.Code(inv.Code.String() + "-" + name)
		for _, p := range ubl.InvoiceParties(inv, in) {
			p.GOBL.Alias += name
		}
		return nil
	}
}

var formatFuncsTest = ubl.Format{
	Key:                   "ubl+funcs-test",
	Import:                ubl.FormatEN16931.Import,
	Export:                ubl.FormatEN16931.Export,
	CustomizationID:       testCustomizationID,
	OutputCustomizationID: "urn:example:funcs-test-output",
	Addons:                ubl.FormatEN16931.Addons,
	Match: func(_, profileID string) bool {
		return profileID == testProfileID
	},
	Fallback: func(customizationID, _ string) bool {
		return customizationID == "urn:example:funcs-test-fallback"
	},
	ExportFuncs: []ubl.ExportFunc{testExport("a"), testExport("b")},
	ImportFuncs: []ubl.ImportFunc{testImport("a"), testImport("b")},
}

var errFuncs = errors.New("format func failed")

// formatFuncsErrors has functions that fail in both directions.
var formatFuncsErrors = ubl.Format{
	Key:             "ubl+funcs-errors",
	Import:          ubl.FormatEN16931.Import,
	Export:          ubl.FormatEN16931.Export,
	CustomizationID: "urn:example:funcs-errors",
	ExportFuncs: []ubl.ExportFunc{
		func(*ubl.Format, *gobl.Envelope, ubl.Document) error { return errFuncs },
	},
	ImportFuncs: []ubl.ImportFunc{
		func(*ubl.Format, ubl.Document, *gobl.Envelope) error { return errFuncs },
	},
}

// formatExportOnly lists no import schemas, so it is never detected.
var formatExportOnly = ubl.Format{
	Key:             "ubl+export-only-test",
	Export:          ubl.FormatEN16931.Export,
	CustomizationID: "urn:example:export-only",
}

func init() {
	ubl.RegisterFormats(formatFuncsTest, formatFuncsErrors, formatExportOnly)
}

func TestFormatFuncs(t *testing.T) {
	env := loadTestEnvelope(t, "invoice-minimal.json")
	doc, err := ubl.ExportInvoice(env, ubl.WithFormat(formatFuncsTest))
	require.NoError(t, err)

	t.Run("export applies the functions in order", func(t *testing.T) {
		n := len(doc.Note)
		require.GreaterOrEqual(t, n, 2)
		assert.Equal(t, []string{"a", "b"}, doc.Note[n-2:])
		assert.Equal(t, "ab", doc.AccountingSupplierParty.Party.IndustryClassificationCode)
		assert.Equal(t, "ab", doc.AccountingCustomerParty.Party.IndustryClassificationCode)
	})

	t.Run("import applies the functions in order", func(t *testing.T) {
		data, err := ubl.Encode(doc)
		require.NoError(t, err)
		parsed, err := ubl.Decode(data)
		require.NoError(t, err)
		out, err := ubl.Import(parsed)
		require.NoError(t, err)
		inv, ok := out.Extract().(*bill.Invoice)
		require.True(t, ok)
		assert.Equal(t, "SAMPLE-001-a-b", inv.Code.String())
		assert.Equal(t, "ab", inv.Supplier.Alias)
	})

	t.Run("other formats are unaffected", func(t *testing.T) {
		doc, err := ubl.ExportInvoice(env, ubl.WithFormat(ubl.FormatEN16931))
		require.NoError(t, err)
		assert.Empty(t, doc.AccountingSupplierParty.Party.IndustryClassificationCode)
	})
}

func TestFormatFuncErrors(t *testing.T) {
	f := formatFuncsErrors

	t.Run("export invoice", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")
		_, err := ubl.ExportInvoice(env, ubl.WithFormat(f))
		assert.ErrorIs(t, err, errFuncs)
	})
	t.Run("export status", func(t *testing.T) {
		env, err := gobl.Envelop(basePeppolStatus())
		require.NoError(t, err)
		_, err = ubl.Export(env, ubl.WithFormat(f))
		assert.ErrorIs(t, err, errFuncs)
	})
	t.Run("import invoice", func(t *testing.T) {
		in := parsedFixture(t, "peppol/base-example.xml")
		_, err := ubl.Import(in, ubl.WithFormat(f))
		assert.ErrorIs(t, err, errFuncs)
	})
	t.Run("import status", func(t *testing.T) {
		env, err := gobl.Envelop(basePeppolStatus())
		require.NoError(t, err)
		doc, err := ubl.Export(env, ubl.WithFormat(ubl.FormatPeppol))
		require.NoError(t, err)
		ar := doc.(*ubl.ApplicationResponse)
		ar.CustomizationID = f.CustomizationID
		ar.ProfileID = nil
		_, err = ubl.Import(ar)
		assert.ErrorIs(t, err, errFuncs)
	})
}

// importCall records what an import function saw: whether the invoice had
// been reconciled, and whether the envelope had been calculated.
type importCall struct {
	name       string
	reconciled bool
	calculated bool
}

// importRecorder collects the calls of the import functions it builds, in
// the order they run.
type importRecorder []importCall

func (r *importRecorder) fn(name string) ubl.ImportFunc {
	return func(_ *ubl.Format, _ ubl.Document, env *gobl.Envelope) error {
		inv, ok := env.Extract().(*bill.Invoice)
		if !ok {
			return nil
		}
		*r = append(*r, importCall{
			name:       name,
			reconciled: inv.Totals != nil,
			calculated: env.Head.Digest != nil,
		})
		return nil
	}
}

// formatWithReconciledFuncs is EN 16931 with the reconciled import functions.
func formatWithReconciledFuncs(fns ...ubl.ImportFunc) ubl.Format {
	f := ubl.FormatEN16931
	f.ReconciledImportFuncs = fns
	return f
}

// addCharge adds an untaxed document charge of 10.00.
func addCharge(_ *ubl.Format, _ ubl.Document, env *gobl.Envelope) error {
	inv := env.Extract().(*bill.Invoice)
	inv.Charges = append(inv.Charges, &bill.Charge{
		Reason: "Restated",
		Amount: num.MakeAmount(1000, 2),
	})
	return nil
}

// replaceWithCharge replaces the envelope's invoice with a copy, and adds the
// charge to the copy.
func replaceWithCharge(f *ubl.Format, doc ubl.Document, env *gobl.Envelope) error {
	data, err := json.Marshal(env.Extract())
	if err != nil {
		return err
	}
	inv := new(bill.Invoice)
	if err := json.Unmarshal(data, inv); err != nil {
		return err
	}
	if env.Document, err = schema.NewObject(inv); err != nil {
		return err
	}
	return addCharge(f, doc, env)
}

// roundDown rounds the amount payable down by 0.01.
func roundDown(_ *ubl.Format, _ ubl.Document, env *gobl.Envelope) error {
	inv := env.Extract().(*bill.Invoice)
	rounding := num.MakeAmount(-1, 2)
	inv.Totals.Rounding = &rounding
	return nil
}

// addRoundedCharge adds an untaxed document charge of 0.01 and rounds the
// amount payable down by as much.
func addRoundedCharge(f *ubl.Format, doc ubl.Document, env *gobl.Envelope) error {
	inv := env.Extract().(*bill.Invoice)
	inv.Charges = append(inv.Charges, &bill.Charge{
		Reason: "Restated",
		Amount: num.MakeAmount(1, 2),
	})
	return roundDown(f, doc, env)
}

// addAdvance adds an advance payment of 100.00.
func addAdvance(_ *ubl.Format, _ ubl.Document, env *gobl.Envelope) error {
	inv := env.Extract().(*bill.Invoice)
	inv.Payment.Advances = append(inv.Payment.Advances, &pay.Record{
		Description: "Restated",
		Amount:      num.MakeAmount(10000, 2),
	})
	return nil
}

// addDiscount adds an untaxed document discount of 10.00.
func addDiscount(_ *ubl.Format, _ ubl.Document, env *gobl.Envelope) error {
	inv := env.Extract().(*bill.Invoice)
	inv.Discounts = append(inv.Discounts, &bill.Discount{
		Reason: "Restated",
		Amount: num.MakeAmount(1000, 2),
	})
	return nil
}

func TestReconciledImportFuncs(t *testing.T) {
	const fixture = "peppol/base-example.xml"
	ten := num.MakeAmount(1000, 2)
	env, err := ubl.Import(parsedFixture(t, fixture), ubl.WithFormat(ubl.FormatEN16931))
	require.NoError(t, err)
	before := env.Extract().(*bill.Invoice)
	require.NotNil(t, before.Totals.Charge)

	for _, tc := range []struct{ name, path string }{
		{"invoice", fixture},
		{"credit note", "peppol/base-creditnote-correction.xml"},
	} {
		t.Run("run in order after reconciliation: "+tc.name, func(t *testing.T) {
			var r importRecorder
			f := formatWithReconciledFuncs(r.fn("reconciled-a"), r.fn("reconciled-b"))
			f.ImportFuncs = []ubl.ImportFunc{r.fn("import-a"), r.fn("import-b")}

			_, err := ubl.Import(parsedFixture(t, tc.path), ubl.WithFormat(f))
			require.NoError(t, err)
			assert.Equal(t, importRecorder{
				{name: "import-a"},
				{name: "import-b"},
				{name: "reconciled-a", reconciled: true, calculated: true},
				{name: "reconciled-b", reconciled: true, calculated: true},
			}, r)
		})
	}

	t.Run("bypassed invoices do not run them", func(t *testing.T) {
		var r importRecorder
		f := formatWithReconciledFuncs(r.fn("reconciled"), addCharge)
		f.ImportFuncs = []ubl.ImportFunc{r.fn("import")}

		env, err := ubl.Import(parsedFixture(t, "en16931/line-totals-mismatch.xml"), ubl.WithFormat(f))
		require.NoError(t, err)
		inv := env.Extract().(*bill.Invoice)
		require.True(t, inv.HasTags(tax.TagBypass))
		assert.Equal(t, importRecorder{{name: "import"}}, r)
		assert.Empty(t, inv.Charges)
	})

	t.Run("totals are calculated again", func(t *testing.T) {
		f := formatWithReconciledFuncs(addCharge, addDiscount)
		env, err := ubl.Import(parsedFixture(t, fixture), ubl.WithFormat(f))
		require.NoError(t, err)
		inv := env.Extract().(*bill.Invoice)
		assert.Equal(t, before.Totals.Charge.Add(ten).String(), inv.Totals.Charge.String())
		require.NotNil(t, inv.Totals.Discount)
		assert.Equal(t, "10.00", inv.Totals.Discount.String())
		assert.Equal(t, before.Totals.Payable.String(), inv.Totals.Payable.String())
	})

	for _, tc := range []struct {
		name string
		fn   ubl.ImportFunc
		now  string
	}{
		{"in place", addCharge, "total with tax 1666.25, payable 1666.25, due none"},
		{"in a replacement", replaceWithCharge, "total with tax 1666.25, payable 1666.25, due none"},
		{"total with tax only", addRoundedCharge, "total with tax 1656.26, payable 1656.25, due none"},
		{"amount payable only", roundDown, "total with tax 1656.25, payable 1656.24, due none"},
		{"amount due only", addAdvance, "total with tax 1656.25, payable 1656.25, due 1556.25"},
	} {
		t.Run("a change to the amounts owed is an error: "+tc.name, func(t *testing.T) {
			f := formatWithReconciledFuncs(tc.fn)
			_, err := ubl.Import(parsedFixture(t, fixture), ubl.WithFormat(f))
			require.Error(t, err)
			assert.EqualError(t, err, "format "+f.Key.String()+": reconciled import changed the amounts owed: "+
				"was total with tax 1656.25, payable 1656.25, due none; now "+tc.now)
		})
	}

	t.Run("a document that is no longer an invoice is an error", func(t *testing.T) {
		f := formatWithReconciledFuncs(func(_ *ubl.Format, _ ubl.Document, env *gobl.Envelope) error {
			var err error
			env.Document, err = schema.NewObject(basePeppolStatus())
			return err
		})
		_, err := ubl.Import(parsedFixture(t, fixture), ubl.WithFormat(f))
		require.Error(t, err)
		assert.ErrorContains(t, err, f.Key.String())
	})

	t.Run("an error stops the import", func(t *testing.T) {
		var r importRecorder
		f := formatWithReconciledFuncs(
			func(*ubl.Format, ubl.Document, *gobl.Envelope) error { return errFuncs },
			r.fn("after"),
		)
		_, err := ubl.Import(parsedFixture(t, "peppol/base-creditnote-correction.xml"), ubl.WithFormat(f))
		assert.ErrorIs(t, err, errFuncs)
		assert.Empty(t, r)
	})

	t.Run("application responses do not run them", func(t *testing.T) {
		env, err := gobl.Envelop(basePeppolStatus())
		require.NoError(t, err)
		doc, err := ubl.Export(env, ubl.WithFormat(ubl.FormatPeppol))
		require.NoError(t, err)
		f := ubl.FormatPeppol
		f.ReconciledImportFuncs = []ubl.ImportFunc{
			func(*ubl.Format, ubl.Document, *gobl.Envelope) error { return errFuncs },
		}
		_, err = ubl.Import(doc, ubl.WithFormat(f))
		assert.NoError(t, err)
	})
}

func TestFindFormatMatching(t *testing.T) {
	t.Run("match before customization", func(t *testing.T) {
		f := ubl.FindFormat(ubl.FormatEN16931.CustomizationID, testProfileID)
		require.NotNil(t, f)
		assert.Equal(t, formatFuncsTest.Key, f.Key)
	})
	t.Run("output customization", func(t *testing.T) {
		f := ubl.FindFormat("urn:example:funcs-test-output", "")
		require.NotNil(t, f)
		assert.Equal(t, formatFuncsTest.Key, f.Key)
	})
	t.Run("fallback after customization", func(t *testing.T) {
		f := ubl.FindFormat("urn:example:funcs-test-fallback", "")
		require.NotNil(t, f)
		assert.Equal(t, formatFuncsTest.Key, f.Key)
	})
	t.Run("export only", func(t *testing.T) {
		assert.Nil(t, ubl.FindFormat("urn:example:export-only", ""))
		f := convert.FormatFor(formatExportOnly.Key)
		require.NotNil(t, f)
		assert.Empty(t, f.Import)
		assert.Equal(t, formatExportOnly.Export, f.Export)
	})
	t.Run("registered converter detects its own format", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")
		out, err := convert.Export(env, formatFuncsTest.Key)
		require.NoError(t, err)
		f, err := convert.Detect(out.Data)
		require.NoError(t, err)
		assert.Equal(t, formatFuncsTest.Key, f.Key)
	})
}

func TestInvoiceParties(t *testing.T) {
	env := loadTestEnvelope(t, "invoice-minimal.json")
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)
	agent := &org.Party{Name: "Agent"}
	inv.Supplier.Agent = agent
	inv.Ordering = &bill.Ordering{
		Seller: &org.Party{Name: "Representative"},
		Issuer: &org.Party{Name: "Issuer"},
	}
	inv.Payment.Payee = &org.Party{Name: "Payee"}
	inv.Lines[0].Seller = &org.Party{Name: "Manufacturer"}
	require.NoError(t, env.Calculate())

	doc, err := ubl.ExportInvoice(env)
	require.NoError(t, err)
	doc.AccountingSupplierParty.Party.AgentParty = ubl.NewParty(agent)

	names := make(map[string]string)
	for _, p := range ubl.InvoiceParties(inv, doc) {
		names[p.GOBL.Name] = p.UBL.PartyName.Name
	}
	for _, name := range []string{"Agent", "Representative", "Issuer", "Payee", "Manufacturer"} {
		assert.Equal(t, name, names[name], name)
	}
	assert.Contains(t, names, inv.Supplier.Name)
	assert.Contains(t, names, inv.Customer.Name)

	t.Run("parse", func(t *testing.T) {
		p := ubl.ParseParty(doc.AccountingSupplierParty.Party.AgentParty)
		assert.Equal(t, "Agent", p.Name)
	})
}

func TestFormatHelpers(t *testing.T) {
	assert.Equal(t, "2024-01-02", ubl.FormatDate(cal.MakeDate(2024, 1, 2)))
	assert.Equal(t, "10.00", ubl.NewAmount(num.MakeAmount(10, 0), "EUR").Value)
	code, ok := ubl.TaxPointCode(tax.PointDelivery)
	assert.True(t, ok)
	assert.Equal(t, "35", code)
	_, ok = ubl.TaxPointCode("unknown")
	assert.False(t, ok)
	assert.Equal(t, "ab", ubl.CleanString("a�b"))
	assert.Equal(t, "Ana Horvat", ubl.ContactName(&org.Name{Given: "Ana", Surname: "Horvat"}))
}
