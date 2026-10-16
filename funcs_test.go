package ubl_test

import (
	"errors"
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
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
}

// TestFoldToCore pins what happens to a document that repeats what the EN
// 16931 core model allows once, such as one imported from a conformant
// extension: formats listing FoldToCore keep one buyer identifier
// (UBL-SR-16), one payee identifier besides SEPA (UBL-SR-20) and one note
// per line (UBL-SR-34), while the others keep them all.
func TestFoldToCore(t *testing.T) {
	extended := func(t *testing.T) *gobl.Envelope {
		t.Helper()
		env := loadTestEnvelope(t, "invoice-complete.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		inv.Customer.Identities = []*org.Identity{
			{Code: "726697736", Ext: tax.ExtensionsOf(cbc.CodeMap{iso.ExtKeySchemeID: "SE"})},
			{Code: "73269773600017", Ext: tax.ExtensionsOf(cbc.CodeMap{iso.ExtKeySchemeID: "0009"})},
			{Code: "CUST-1"},
		}
		inv.Payment.Payee.Identities = []*org.Identity{
			{Code: "PAYEE-1"},
			{Code: "2298740918237", Ext: tax.ExtensionsOf(cbc.CodeMap{iso.ExtKeySchemeID: "0088"})},
		}
		inv.Payment.Instructions = &pay.Instructions{
			Key: pay.MeansKeyDirectDebit,
			DirectDebit: &pay.DirectDebit{
				Ref:      "MANDATE-1",
				Creditor: "NO98ZZZ09999999999",
			},
			Ext: tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyPaymentMeans: "59"}),
		}
		inv.Lines[0].Notes = []*org.Note{
			{Text: "Delivered on Monday"},
			{Ext: tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyTextSubject: "AAI"}), Text: "Handle with care"},
		}
		require.NoError(t, env.Calculate())
		return env
	}

	t.Run("core formats keep one of each", func(t *testing.T) {
		env := extended(t)
		doc, err := ubl.ExportInvoice(env)
		require.NoError(t, err)

		ids := doc.AccountingCustomerParty.Party.PartyIdentification
		require.Len(t, ids, 1)
		assert.Equal(t, "73269773600017", ids[0].ID.Value, "the buyer identifier with an ISO 6523 ICD scheme wins")
		assert.Equal(t, "0009", *ids[0].ID.SchemeID)

		ids = doc.PayeeParty.PartyIdentification
		require.Len(t, ids, 2)
		assert.Equal(t, "2298740918237", ids[0].ID.Value)
		assert.Equal(t, "0088", *ids[0].ID.SchemeID)
		assert.Equal(t, "SEPA", *ids[1].ID.SchemeID, "the SEPA creditor identifier is kept alongside")

		assert.Equal(t, []string{"Delivered on Monday\n\n#AAI#Handle with care"}, doc.InvoiceLines[0].Note)

		if *validate {
			inv, ok := env.Extract().(*bill.Invoice)
			require.True(t, ok)
			data, err := ubl.Encode(doc)
			require.NoError(t, err)
			validateXML(t, phormClient(t), ubl.FormatEN16931.VESID(inv), data)
		}
	})

	t.Run("credit note lines fold too", func(t *testing.T) {
		env := loadTestEnvelope(t, "credit-note.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		inv.Lines[0].Notes = []*org.Note{{Text: "first"}, {Text: "second"}}
		require.NoError(t, env.Calculate())

		doc, err := ubl.ExportInvoice(env)
		require.NoError(t, err)
		assert.Equal(t, []string{"first\n\nsecond"}, doc.CreditNoteLines[0].Note)
	})

	t.Run("formats without the fold keep them all", func(t *testing.T) {
		f := ubl.FormatEN16931
		f.ExportFuncs = nil
		doc, err := ubl.ExportInvoice(extended(t), ubl.WithFormat(f))
		require.NoError(t, err)

		assert.Len(t, doc.AccountingCustomerParty.Party.PartyIdentification, 3)
		assert.Len(t, doc.PayeeParty.PartyIdentification, 3)
		assert.Len(t, doc.InvoiceLines[0].Note, 2)
	})
}
