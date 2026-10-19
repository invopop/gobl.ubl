package ubl_test

import (
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextEN16931(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Add EN16931 addon
		inv.SetAddons(en16931.V2017)
		require.NoError(t, inv.Calculate())

		// Convert with EN16931 context
		doc, err := ubl.Export(env, ubl.WithFormat(ubl.FormatEN16931))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// Verify CustomizationID
		assert.Equal(t, "urn:cen.eu:en16931:2017", ublInv.CustomizationID)
		// EN16931 context has no ProfileID
		assert.Empty(t, ublInv.ProfileID)
	})

}

func TestGetVESID(t *testing.T) {
	t.Run("invoice VESID for standard invoice", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		vesid := ubl.FormatEN16931.VESID(inv)
		assert.Equal(t, "eu.cen.en16931:ubl:1.3.16", vesid)
	})

	t.Run("credit note VESID for credit note", func(t *testing.T) {
		env := loadTestEnvelope(t, "credit-note.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Verify it's a credit note
		require.True(t, inv.Type.In(bill.InvoiceTypeCreditNote))

		vesid := ubl.FormatEN16931.VESID(inv)
		assert.Equal(t, "eu.cen.en16931:ubl-creditnote:1.3.16", vesid)
	})
}

func TestFindContext(t *testing.T) {
	t.Run("find EN16931 by CustomizationID", func(t *testing.T) {
		ctx := ubl.FindFormat("urn:cen.eu:en16931:2017", "")
		require.NotNil(t, ctx)
		assert.Equal(t, ubl.FormatEN16931.CustomizationID, ctx.CustomizationID)
	})

	t.Run("find by CustomizationID and ProfileID", func(t *testing.T) {
		ctx := ubl.FindFormat(formatSpecsTest.CustomizationID, formatSpecsTest.ProfileID)
		require.NotNil(t, ctx)
		assert.Equal(t, formatSpecsTest.CustomizationID, ctx.CustomizationID)
		assert.Equal(t, formatSpecsTest.ProfileID, ctx.ProfileID)
	})

	t.Run("find by a specification's CustomizationID and ProfileID", func(t *testing.T) {
		ctx := ubl.FindFormat(specCreditNoteTest.CustomizationID, specCreditNoteTest.ProfileID)
		require.NotNil(t, ctx)
		assert.Equal(t, formatSpecsTest.Key, ctx.Key)
	})

	t.Run("find EN16931 when no ProfileID provided", func(t *testing.T) {
		ctx := ubl.FindFormat("urn:cen.eu:en16931:2017", "")
		require.NotNil(t, ctx)
		assert.Equal(t, ubl.FormatEN16931.CustomizationID, ctx.CustomizationID)
	})

	t.Run("find EN16931 with non-billing-mode ProfileID", func(t *testing.T) {
		// EN16931 documents may have arbitrary ProfileIDs that are not French billing modes
		ctx := ubl.FindFormat("urn:cen.eu:en16931:2017", "Invoicing on purchase order")
		require.NotNil(t, ctx)
		assert.Equal(t, ubl.FormatEN16931.CustomizationID, ctx.CustomizationID)
	})

	t.Run("unknown CustomizationID returns nil", func(t *testing.T) {
		ctx := ubl.FindFormat("unknown:customization:id", "")
		assert.Nil(t, ctx)
	})
}

// specCreditNoteTest is a test specification used for credit notes.
var specCreditNoteTest = ubl.Specification{
	CustomizationID: "urn:example:specs-test:credit-note",
	ProfileID:       "urn:example:specs-test:credit-note:profile",
	Addons:          []cbc.Key{en16931.V2017},
	VESIDs: ubl.VESIDMapping{
		Invoice:    "example:specs-test:credit-note-as-invoice",
		CreditNote: "example:specs-test:credit-note",
	},
	Applies: func(doc any) bool {
		inv, ok := doc.(*bill.Invoice)
		return ok && inv.Type.In(bill.InvoiceTypeCreditNote)
	},
}

// specStatusTest is a test specification used for statuses.
var specStatusTest = ubl.Specification{
	CustomizationID: "urn:example:specs-test:status",
	ProfileID:       "urn:example:specs-test:status:profile",
	VESIDs: ubl.VESIDMapping{
		Status: "example:specs-test:status",
	},
	Applies: func(doc any) bool {
		_, ok := doc.(*bill.Status)
		return ok
	},
}

// formatSpecsTest is a test format that writes invoices and statuses, with a
// specification of its own for credit notes and another for statuses.
var formatSpecsTest = ubl.Format{
	Key:             "ubl+specs-test",
	Import:          []schema.ID{schema.Lookup(bill.Invoice{}), schema.Lookup(bill.Status{})},
	Export:          []schema.ID{schema.Lookup(bill.Invoice{}), schema.Lookup(bill.Status{})},
	CustomizationID: "urn:example:specs-test",
	ProfileID:       "urn:example:specs-test:profile",
	Addons:          []cbc.Key{en16931.V2017},
	VESIDs: ubl.VESIDMapping{
		Invoice:    "example:specs-test:invoice",
		CreditNote: "example:specs-test:creditnote",
	},
	Specifications: []ubl.Specification{specCreditNoteTest, specStatusTest},
}

func init() {
	ubl.RegisterFormats(formatSpecsTest)
}

func TestFormatSpecifications(t *testing.T) {
	creditNote := func(t *testing.T) *gobl.Envelope {
		t.Helper()
		env := loadTestEnvelope(t, "credit-note.json")
		require.True(t, env.Extract().(*bill.Invoice).Type.In(bill.InvoiceTypeCreditNote))
		return env
	}
	status := func(t *testing.T) *gobl.Envelope {
		t.Helper()
		env, err := gobl.Envelop(baseStatus())
		require.NoError(t, err)
		return env
	}

	t.Run("specification for each document", func(t *testing.T) {
		inv := loadTestEnvelope(t, "invoice-minimal.json").Extract()
		assert.Equal(t, formatSpecsTest.CustomizationID, formatSpecsTest.SpecificationFor(inv).CustomizationID)
		assert.Equal(t, specCreditNoteTest.CustomizationID, formatSpecsTest.SpecificationFor(creditNote(t).Extract()).CustomizationID)
		assert.Equal(t, specStatusTest.CustomizationID, formatSpecsTest.SpecificationFor(status(t).Extract()).CustomizationID)
	})

	t.Run("VESIDs", func(t *testing.T) {
		inv := loadTestEnvelope(t, "invoice-minimal.json").Extract()
		assert.Equal(t, "example:specs-test:invoice", formatSpecsTest.VESID(inv))
		assert.Equal(t, "example:specs-test:credit-note", formatSpecsTest.VESID(creditNote(t).Extract()))
		assert.Equal(t, "example:specs-test:status", formatSpecsTest.VESID(status(t).Extract()))
		assert.Empty(t, formatSpecsTest.VESID(&bill.Payment{}))
	})

	t.Run("export, detect and import", func(t *testing.T) {
		for name, tc := range map[string]struct {
			env    *gobl.Envelope
			spec   string
			addons bool
		}{
			"credit note": {creditNote(t), specCreditNoteTest.CustomizationID, true},
			"status":      {status(t), specStatusTest.CustomizationID, false},
		} {
			t.Run(name, func(t *testing.T) {
				out, err := convert.Export(tc.env, formatSpecsTest.Key)
				require.NoError(t, err)
				assert.Contains(t, string(out.Data), tc.spec)

				f, err := convert.Detect(out.Data)
				require.NoError(t, err)
				assert.Equal(t, formatSpecsTest.Key, f.Key)

				res, err := convert.Import(out.Data)
				require.NoError(t, err)
				doc := res.Envelope.Extract().(interface{ GetAddons() []cbc.Key })
				if tc.addons {
					assert.Contains(t, doc.GetAddons(), en16931.V2017)
				} else {
					assert.Empty(t, doc.GetAddons())
				}
			})
		}
	})
}
