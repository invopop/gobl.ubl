package ubl_test

import (
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/schema"
	"github.com/invopop/gobl/tax"
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

func TestContextTest(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")

		doc, err := ubl.Export(env, ubl.WithFormat(formatTest))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		assert.Equal(t, formatTest.CustomizationID, ublInv.CustomizationID)
		require.NotNil(t, ublInv.ProfileID)
		assert.Equal(t, formatTest.ProfileID, ublInv.ProfileID.Value)
	})

	t.Run("self-billed conversion", func(t *testing.T) {
		env := loadTestEnvelope(t, "peppol-self-billed/self-billed-invoice.json")

		doc, err := ubl.Export(env, ubl.WithFormat(formatTest))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		assert.Equal(t, specTestSelfBilling.CustomizationID, ublInv.CustomizationID)
		require.NotNil(t, ublInv.ProfileID)
		assert.Equal(t, specTestSelfBilling.ProfileID, ublInv.ProfileID.Value)
	})
}

func TestGetVESID(t *testing.T) {
	t.Run("invoice VESID for standard invoice", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		vesid := formatTest.VESID(inv)
		assert.Equal(t, "test:invoice", vesid)

		// Get VESID for EN16931 context
		vesid = ubl.FormatEN16931.VESID(inv)
		assert.Equal(t, "eu.cen.en16931:ubl:1.3.16", vesid)
	})

	t.Run("credit note VESID for credit note", func(t *testing.T) {
		env := loadTestEnvelope(t, "credit-note.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Verify it's a credit note
		require.True(t, inv.Type.In(bill.InvoiceTypeCreditNote))

		vesid := formatTest.VESID(inv)
		assert.Equal(t, "test:creditnote", vesid)

		// Get VESID for EN16931 context
		vesid = ubl.FormatEN16931.VESID(inv)
		assert.Equal(t, "eu.cen.en16931:ubl-creditnote:1.3.16", vesid)
	})

	t.Run("self-billed invoice VESID", func(t *testing.T) {
		env := loadTestEnvelope(t, "peppol-self-billed/self-billed-invoice.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		vesid := formatTest.VESID(inv)
		assert.Equal(t, "test:invoice-self-billing", vesid)
	})
}

func TestFindContext(t *testing.T) {
	t.Run("find EN16931 by CustomizationID", func(t *testing.T) {
		ctx := ubl.FindFormat("urn:cen.eu:en16931:2017", "")
		require.NotNil(t, ctx)
		assert.Equal(t, ubl.FormatEN16931.CustomizationID, ctx.CustomizationID)
	})

	t.Run("find by CustomizationID and ProfileID", func(t *testing.T) {
		ctx := ubl.FindFormat(formatTest.CustomizationID, formatTest.ProfileID)
		require.NotNil(t, ctx)
		assert.Equal(t, formatTest.Key, ctx.Key)
	})

	t.Run("find by another specification", func(t *testing.T) {
		ctx := ubl.FindFormat(specTestSelfBilling.CustomizationID, specTestSelfBilling.ProfileID)
		require.NotNil(t, ctx)
		assert.Equal(t, formatTest.Key, ctx.Key)
	})

	t.Run("Peppol documents are not claimed", func(t *testing.T) {
		ctx := ubl.FindFormat("urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0", "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0")
		assert.Nil(t, ctx)
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

func TestFormatSpecifications(t *testing.T) {
	selfBilled := func(t *testing.T) *gobl.Envelope {
		t.Helper()
		env := loadTestEnvelope(t, "peppol-self-billed/self-billed-invoice.json")
		inv := env.Extract().(*bill.Invoice)
		require.True(t, inv.HasTags(tax.TagSelfBilled))
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
		assert.Equal(t, formatTest.CustomizationID, formatTest.SpecificationFor(inv).CustomizationID)
		assert.Equal(t, specTestSelfBilling.CustomizationID, formatTest.SpecificationFor(selfBilled(t).Extract()).CustomizationID)
		assert.Equal(t, specTestStatus.CustomizationID, formatTest.SpecificationFor(status(t).Extract()).CustomizationID)
	})

	t.Run("VESIDs", func(t *testing.T) {
		assert.Equal(t, "test:status", formatTest.VESID(status(t).Extract()))
		assert.Empty(t, formatTest.VESID(&bill.Payment{}))
	})

	t.Run("export, detect and import", func(t *testing.T) {
		for name, tc := range map[string]struct {
			env    *gobl.Envelope
			spec   string
			addons bool
		}{
			"self-billed invoice": {selfBilled(t), specTestSelfBilling.CustomizationID, true},
			"status":              {status(t), specTestStatus.CustomizationID, false},
		} {
			t.Run(name, func(t *testing.T) {
				out, err := convert.Export(tc.env, formatTest.Key)
				require.NoError(t, err)
				assert.Contains(t, string(out.Data), tc.spec)

				f, err := convert.Detect(out.Data)
				require.NoError(t, err)
				assert.Equal(t, formatTest.Key, f.Key)

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

// formatTest is a test-only format with a ProfileID and further
// specifications for self-billed invoices and statuses, covering the generic
// format features no format in this package uses.
var formatTest = ubl.Format{
	Key:             "ubl+test-v1",
	Name:            i18n.NewString("UBL Test"),
	Import:          []schema.ID{schema.Lookup(bill.Invoice{}), schema.Lookup(bill.Status{})},
	Export:          []schema.ID{schema.Lookup(bill.Invoice{}), schema.Lookup(bill.Status{})},
	CustomizationID: "urn:cen.eu:en16931:2017#compliant#urn:example:test:billing:1",
	ProfileID:       "urn:example:test:billing:process",
	Addons:          []cbc.Key{en16931.V2017},
	VESIDs: ubl.VESIDMapping{
		Invoice:    "test:invoice",
		CreditNote: "test:creditnote",
	},
	Specifications: []ubl.Specification{specTestSelfBilling, specTestStatus},
}

// specTestSelfBilling is the test format's specification for self-billed
// invoices.
var specTestSelfBilling = ubl.Specification{
	CustomizationID: "urn:cen.eu:en16931:2017#compliant#urn:example:test:selfbilling:1",
	ProfileID:       "urn:example:test:selfbilling:process",
	Addons:          []cbc.Key{en16931.V2017},
	VESIDs: ubl.VESIDMapping{
		Invoice:    "test:invoice-self-billing",
		CreditNote: "test:creditnote-self-billing",
	},
	Applies: func(doc any) bool {
		inv, ok := doc.(*bill.Invoice)
		return ok && inv.HasTags(tax.TagSelfBilled)
	},
}

// specTestStatus is the test format's specification for statuses.
var specTestStatus = ubl.Specification{
	CustomizationID: "urn:example:test:response:1",
	ProfileID:       "urn:example:test:response:process",
	VESIDs: ubl.VESIDMapping{
		Status: "test:status",
	},
	Applies: func(doc any) bool {
		_, ok := doc.(*bill.Status)
		return ok
	},
}

func init() {
	ubl.RegisterFormats(formatTest)
}
