package ubl_test

import (
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
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

func TestContextPeppol(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.SetAddons(en16931.V2017)
		require.NoError(t, inv.Calculate())

		// Convert with Peppol context
		doc, err := ubl.Export(env, ubl.WithFormat(ubl.FormatPeppol))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// Verify CustomizationID and ProfileID
		assert.Equal(t, "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0", ublInv.CustomizationID)
		assert.Equal(t, "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0", ublInv.ProfileID.Value)
	})

}

func TestContextPeppolSelfBilled(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		env := loadTestEnvelope(t, "peppol-self-billed/self-billed-invoice.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.SetAddons(en16931.V2017)
		require.NoError(t, inv.Calculate())

		// Convert directly with PeppolSelfBilled context
		doc, err := ubl.Export(env, ubl.WithFormat(ubl.FormatPeppol))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// Verify CustomizationID and ProfileID
		assert.Equal(t, "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:selfbilling:3.0", ublInv.CustomizationID)
		assert.Equal(t, "urn:fdc:peppol.eu:2017:poacc:selfbilling:01:1.0", ublInv.ProfileID.Value)
	})
}

func TestGetVESID(t *testing.T) {
	t.Run("invoice VESID for standard invoice", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Get VESID for Peppol context
		vesid := ubl.FormatPeppol.VESID(inv)
		assert.Equal(t, "eu.peppol.bis3:invoice:2026.5", vesid)

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

		// Get VESID for Peppol context
		vesid := ubl.FormatPeppol.VESID(inv)
		assert.Equal(t, "eu.peppol.bis3:creditnote:2026.5", vesid)

		// Get VESID for EN16931 context
		vesid = ubl.FormatEN16931.VESID(inv)
		assert.Equal(t, "eu.cen.en16931:ubl-creditnote:1.3.16", vesid)
	})

	t.Run("self-billed invoice VESID", func(t *testing.T) {
		env := loadTestEnvelope(t, "peppol-self-billed/self-billed-invoice.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Get VESID for PeppolSelfBilled context
		vesid := ubl.FormatPeppol.VESID(inv)
		assert.Equal(t, "eu.peppol.bis3:invoice-self-billing:2026.5", vesid)
	})
}

func TestFindContext(t *testing.T) {
	t.Run("find EN16931 by CustomizationID", func(t *testing.T) {
		ctx := ubl.FindFormat("urn:cen.eu:en16931:2017", "")
		require.NotNil(t, ctx)
		assert.Equal(t, ubl.FormatEN16931.CustomizationID, ctx.CustomizationID)
	})

	t.Run("find Peppol by CustomizationID and ProfileID", func(t *testing.T) {
		ctx := ubl.FindFormat("urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0", "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0")
		require.NotNil(t, ctx)
		assert.Equal(t, ubl.FormatPeppol.CustomizationID, ctx.CustomizationID)
		assert.Equal(t, ubl.FormatPeppol.ProfileID, ctx.ProfileID)
	})

	t.Run("find PeppolSelfBilled by CustomizationID and ProfileID", func(t *testing.T) {
		ctx := ubl.FindFormat("urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:selfbilling:3.0", "urn:fdc:peppol.eu:2017:poacc:selfbilling:01:1.0")
		require.NotNil(t, ctx)
		assert.Equal(t, ubl.FormatPeppol.Key, ctx.Key)
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

func TestFormatPeppolSpecifications(t *testing.T) {
	selfBilled := func(t *testing.T) *gobl.Envelope {
		t.Helper()
		env := loadTestEnvelope(t, "peppol-self-billed/self-billed-invoice.json")
		inv := env.Extract().(*bill.Invoice)
		require.True(t, inv.HasTags(tax.TagSelfBilled))
		return env
	}
	status := func(t *testing.T) *gobl.Envelope {
		t.Helper()
		env, err := gobl.Envelop(basePeppolStatus())
		require.NoError(t, err)
		return env
	}

	t.Run("specification for each document", func(t *testing.T) {
		inv := loadTestEnvelope(t, "peppol/invoice-minimal.json").Extract()
		assert.Equal(t, ubl.FormatPeppol.CustomizationID, ubl.FormatPeppol.SpecificationFor(inv).CustomizationID)
		assert.Equal(t, ubl.SpecificationPeppolSelfBilling.CustomizationID, ubl.FormatPeppol.SpecificationFor(selfBilled(t).Extract()).CustomizationID)
		assert.Equal(t, ubl.SpecificationPeppolInvoiceResponse.CustomizationID, ubl.FormatPeppol.SpecificationFor(status(t).Extract()).CustomizationID)
	})

	t.Run("VESIDs", func(t *testing.T) {
		assert.Equal(t, "eu.peppol.bis3:invoice-message-response:2026.5", ubl.FormatPeppol.VESID(status(t).Extract()))
		assert.Empty(t, ubl.FormatPeppol.VESID(&bill.Payment{}))
	})

	t.Run("export, detect and import", func(t *testing.T) {
		for name, tc := range map[string]struct {
			env    *gobl.Envelope
			spec   string
			addons bool
		}{
			"self-billed invoice": {selfBilled(t), ubl.SpecificationPeppolSelfBilling.CustomizationID, true},
			"status":              {status(t), ubl.SpecificationPeppolInvoiceResponse.CustomizationID, false},
		} {
			t.Run(name, func(t *testing.T) {
				out, err := convert.Export(tc.env, ubl.FormatPeppol.Key)
				require.NoError(t, err)
				assert.Contains(t, string(out.Data), tc.spec)

				f, err := convert.Detect(out.Data)
				require.NoError(t, err)
				assert.Equal(t, ubl.FormatPeppol.Key, f.Key)

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
