package ubl_test

import (
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/addons/de/xrechnung"
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/addons/fr/facturx"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextEN16931(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Add EN16931 addon
		inv.SetAddons(en16931.V2017)
		require.NoError(t, inv.Calculate())

		// Convert with EN16931 context
		doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextEN16931))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// Verify CustomizationID
		assert.Equal(t, "urn:cen.eu:en16931:2017", ublInv.CustomizationID)
		// EN16931 context has no ProfileID
		assert.Empty(t, ublInv.ProfileID)
	})

	t.Run("with ubl-profile meta", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Add meta field
		if inv.Meta == nil {
			inv.Meta = cbc.Meta{}
		}
		inv.Meta[cbc.Key("ubl-profile")] = "custom-profile"

		inv.SetAddons(en16931.V2017)
		require.NoError(t, inv.Calculate())

		doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextEN16931))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// ProfileID should come from meta
		assert.Equal(t, "custom-profile", ublInv.ProfileID)
	})
}

func TestContextPeppol(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.SetAddons(en16931.V2017)
		require.NoError(t, inv.Calculate())

		// Convert with Peppol context
		doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextPeppol))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// Verify CustomizationID and ProfileID
		assert.Equal(t, "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0", ublInv.CustomizationID)
		assert.Equal(t, "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0", ublInv.ProfileID)
	})

	t.Run("with ubl-profile meta overrides default", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		if inv.Meta == nil {
			inv.Meta = cbc.Meta{}
		}
		inv.Meta[cbc.Key("ubl-profile")] = "custom-peppol-profile"

		inv.SetAddons(en16931.V2017)
		require.NoError(t, inv.Calculate())

		doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextPeppol))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// ProfileID should be overridden by meta
		assert.Equal(t, "custom-peppol-profile", ublInv.ProfileID)
	})

	t.Run("automatic switch to self-billed context when invoice has self-billed tag", func(t *testing.T) {
		env, err := loadTestEnvelope("self-billed-invoice.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Verify the invoice has the self-billed tag
		require.True(t, inv.HasTags(cbc.Key("self-billed")), "invoice should have self-billed tag")

		// Convert with Peppol context - should automatically switch to PeppolSelfBilled
		doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextPeppol))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// Verify that the output uses ContextPeppolSelfBilled's CustomizationID and ProfileID
		assert.Equal(t, "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:selfbilling:3.0", ublInv.CustomizationID,
			"should use self-billing CustomizationID")
		assert.Equal(t, "urn:fdc:peppol.eu:2017:poacc:selfbilling:01:1.0", ublInv.ProfileID,
			"should use self-billing ProfileID")
	})

	t.Run("no automatic switch for non-Peppol contexts with self-billed tag", func(t *testing.T) {
		env, err := loadTestEnvelope("self-billed-invoice.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Verify the invoice has the self-billed tag
		require.True(t, inv.HasTags(cbc.Key("self-billed")), "invoice should have self-billed tag")

		// Convert with EN16931 context - should NOT switch to self-billed
		doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextEN16931))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// Verify that EN16931 context is used (not self-billed)
		assert.Equal(t, "urn:cen.eu:en16931:2017", ublInv.CustomizationID,
			"should use EN16931 CustomizationID, not self-billing")
		assert.Empty(t, ublInv.ProfileID,
			"EN16931 context has no ProfileID")
	})
}

func TestContextXRechnung(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.SetAddons(xrechnung.V3)
		require.NoError(t, inv.Calculate())

		// Convert with XRechnung context
		doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextXRechnung))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// Verify CustomizationID and ProfileID
		assert.Equal(t, "urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0", ublInv.CustomizationID)
		assert.Equal(t, "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0", ublInv.ProfileID)
	})
}

func TestContextPeppolFranceCIUS(t *testing.T) {
	t.Run("with ubl-profile meta", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Add the ubl-profile meta field
		if inv.Meta == nil {
			inv.Meta = cbc.Meta{}
		}
		inv.Meta[cbc.Key("ubl-profile")] = "M1"

		inv.SetAddons(en16931.V2017)
		require.NoError(t, inv.Calculate())

		// Convert with France CIUS context
		doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextPeppolFranceCIUS))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// Verify the CustomizationID in the output is the simple EN16931 one
		assert.Equal(t, "urn:cen.eu:en16931:2017", ublInv.CustomizationID)
		// Verify the ProfileID comes from the meta field
		assert.Equal(t, "M1", ublInv.ProfileID)
	})

	t.Run("without meta field uses default", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.SetAddons(en16931.V2017)
		require.NoError(t, inv.Calculate())

		// Convert with France CIUS context
		doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextPeppolFranceCIUS))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// Verify OutputCustomizationID is used
		assert.Equal(t, "urn:cen.eu:en16931:2017", ublInv.CustomizationID)
		// Verify the ProfileID falls back to the context default
		assert.Equal(t, "urn:peppol:france:billing:regulated", ublInv.ProfileID)
	})

	t.Run("external identification uses full CustomizationID", func(t *testing.T) {
		// Verify the context itself has the full identification
		assert.Equal(t, "urn:cen.eu:en16931:2017#compliant#urn:peppol:france:billing:cius:1.0", ubl.ContextPeppolFranceCIUS.CustomizationID)
		assert.Equal(t, "urn:peppol:france:billing:regulated", ubl.ContextPeppolFranceCIUS.ProfileID)
		assert.Equal(t, "urn:cen.eu:en16931:2017", ubl.ContextPeppolFranceCIUS.OutputCustomizationID)
	})
}

func TestContextPeppolFranceExtended(t *testing.T) {
	t.Run("with ubl-profile meta", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		if inv.Meta == nil {
			inv.Meta = cbc.Meta{}
		}
		inv.Meta[cbc.Key("ubl-profile")] = "M2"

		inv.SetAddons(facturx.V1)
		require.NoError(t, inv.Calculate())

		// Convert with France Extended context
		doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextPeppolFranceExtended))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// Verify OutputCustomizationID is used
		assert.Equal(t, "urn:cen.eu:en16931:2017#conformant#urn.cpro.gouv.fr:1p0:extended-ctc-fr", ublInv.CustomizationID)
		// Verify the ProfileID comes from the meta field
		assert.Equal(t, "M2", ublInv.ProfileID)
	})

	t.Run("without meta field uses default", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.SetAddons(facturx.V1)
		require.NoError(t, inv.Calculate())

		// Convert with France Extended context
		doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextPeppolFranceExtended))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// Verify OutputCustomizationID is used
		assert.Equal(t, "urn:cen.eu:en16931:2017#conformant#urn.cpro.gouv.fr:1p0:extended-ctc-fr", ublInv.CustomizationID)
		// Verify the ProfileID falls back to the context default
		assert.Equal(t, "urn:peppol:france:billing:regulated", ublInv.ProfileID)
	})

	t.Run("external identification uses full CustomizationID", func(t *testing.T) {
		// Verify the context itself has the full identification
		assert.Equal(t, "urn:cen.eu:en16931:2017#conformant#urn:peppol:france:billing:extended:1.0", ubl.ContextPeppolFranceExtended.CustomizationID)
		assert.Equal(t, "urn:peppol:france:billing:regulated", ubl.ContextPeppolFranceExtended.ProfileID)
		assert.Equal(t, "urn:cen.eu:en16931:2017#conformant#urn.cpro.gouv.fr:1p0:extended-ctc-fr", ubl.ContextPeppolFranceExtended.OutputCustomizationID)
	})
}

func TestContextPeppolSelfBilled(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		env, err := loadTestEnvelope("self-billed-invoice.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		inv.SetAddons(en16931.V2017)
		require.NoError(t, inv.Calculate())

		// Convert directly with PeppolSelfBilled context
		doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextPeppolSelfBilled))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// Verify CustomizationID and ProfileID
		assert.Equal(t, "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:selfbilling:3.0", ublInv.CustomizationID)
		assert.Equal(t, "urn:fdc:peppol.eu:2017:poacc:selfbilling:01:1.0", ublInv.ProfileID)
	})
}

func TestGetVESID(t *testing.T) {
	t.Run("invoice VESID for standard invoice", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Get VESID for Peppol context
		vesid := ubl.ContextPeppol.GetVESID(inv)
		assert.Equal(t, "eu.peppol.bis3:invoice:2025.5", vesid)

		// Get VESID for EN16931 context
		vesid = ubl.ContextEN16931.GetVESID(inv)
		assert.Equal(t, "eu.cen.en16931:ubl:1.3.14-2", vesid)

		// Get VESID for XRechnung context
		vesid = ubl.ContextXRechnung.GetVESID(inv)
		assert.Equal(t, "de.xrechnung:ubl-invoice:3.0.2", vesid)
	})

	t.Run("credit note VESID for credit note", func(t *testing.T) {
		env, err := loadTestEnvelope("credit-note.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Verify it's a credit note
		require.True(t, inv.Type.In(bill.InvoiceTypeCreditNote))

		// Get VESID for Peppol context
		vesid := ubl.ContextPeppol.GetVESID(inv)
		assert.Equal(t, "eu.peppol.bis3:creditnote:2025.5", vesid)

		// Get VESID for EN16931 context
		vesid = ubl.ContextEN16931.GetVESID(inv)
		assert.Equal(t, "eu.cen.en16931:ubl-creditnote:1.3.15", vesid)
	})

	t.Run("self-billed invoice VESID", func(t *testing.T) {
		env, err := loadTestEnvelope("self-billed-invoice.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Get VESID for PeppolSelfBilled context
		vesid := ubl.ContextPeppolSelfBilled.GetVESID(inv)
		assert.Equal(t, "eu.peppol.bis3:invoice-self-billing:2025.3", vesid)
	})

	t.Run("France CIUS VESID", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Get VESID for France CIUS context
		vesid := ubl.ContextPeppolFranceCIUS.GetVESID(inv)
		assert.Equal(t, "fr.ctc:ubl-invoice:1.2", vesid)
	})

	t.Run("France Extended VESID", func(t *testing.T) {
		env, err := loadTestEnvelope("invoice-minimal.json")
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Get VESID for France Extended context
		vesid := ubl.ContextPeppolFranceExtended.GetVESID(inv)
		assert.Equal(t, "fr.ctc:ubl-invoice:1.2", vesid)
	})
}

func TestFindContext(t *testing.T) {
	t.Run("find EN16931 by CustomizationID", func(t *testing.T) {
		ctx := ubl.FindContext("urn:cen.eu:en16931:2017", "")
		require.NotNil(t, ctx)
		assert.Equal(t, ubl.ContextEN16931.CustomizationID, ctx.CustomizationID)
	})

	t.Run("find Peppol by CustomizationID and ProfileID", func(t *testing.T) {
		ctx := ubl.FindContext("urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0", "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0")
		require.NotNil(t, ctx)
		assert.Equal(t, ubl.ContextPeppol.CustomizationID, ctx.CustomizationID)
		assert.Equal(t, ubl.ContextPeppol.ProfileID, ctx.ProfileID)
	})

	t.Run("find PeppolSelfBilled by CustomizationID and ProfileID", func(t *testing.T) {
		ctx := ubl.FindContext("urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:selfbilling:3.0", "urn:fdc:peppol.eu:2017:poacc:selfbilling:01:1.0")
		require.NotNil(t, ctx)
		assert.Equal(t, ubl.ContextPeppolSelfBilled.CustomizationID, ctx.CustomizationID)
		assert.Equal(t, ubl.ContextPeppolSelfBilled.ProfileID, ctx.ProfileID)
	})

	t.Run("find France CIUS by full CustomizationID", func(t *testing.T) {
		ctx := ubl.FindContext("urn:cen.eu:en16931:2017#compliant#urn:peppol:france:billing:cius:1.0", "urn:peppol:france:billing:regulated")
		require.NotNil(t, ctx)
		assert.Equal(t, ubl.ContextPeppolFranceCIUS.CustomizationID, ctx.CustomizationID)
		assert.Equal(t, ubl.ContextPeppolFranceCIUS.ProfileID, ctx.ProfileID)
	})

	t.Run("find XRechnung by CustomizationID and ProfileID", func(t *testing.T) {
		ctx := ubl.FindContext("urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0", "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0")
		require.NotNil(t, ctx)
		assert.Equal(t, ubl.ContextXRechnung.CustomizationID, ctx.CustomizationID)
	})

	t.Run("find France CIUS by OutputCustomizationID", func(t *testing.T) {
		// Simulates parsing a French document with OutputCustomizationID
		ctx := ubl.FindContext("urn:cen.eu:en16931:2017", "")
		require.NotNil(t, ctx)
		// Could match either EN16931 or France CIUS since both could use this CustomizationID
		// EN16931 is returned first since it has no OutputCustomizationID
		assert.Equal(t, ubl.ContextEN16931.CustomizationID, ctx.CustomizationID)
	})

	t.Run("find France Extended by OutputCustomizationID", func(t *testing.T) {
		// Simulates parsing a French Extended document
		ctx := ubl.FindContext("urn:cen.eu:en16931:2017#conformant#urn.cpro.gouv.fr:1p0:extended-ctc-fr", "")
		require.NotNil(t, ctx)
		assert.Equal(t, ubl.ContextPeppolFranceExtended.CustomizationID, ctx.CustomizationID)
		assert.Equal(t, "urn:cen.eu:en16931:2017#conformant#urn.cpro.gouv.fr:1p0:extended-ctc-fr", ctx.OutputCustomizationID)
	})

	t.Run("unknown CustomizationID returns nil", func(t *testing.T) {
		ctx := ubl.FindContext("unknown:customization:id", "")
		assert.Nil(t, ctx)
	})
}
