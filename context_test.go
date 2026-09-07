package ubl_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/invopop/gobl.fr.ctc/addon/flow2"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/addons/de/xrechnung"
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/bill"
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
		doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextEN16931))
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
		doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextPeppol))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// Verify CustomizationID and ProfileID
		assert.Equal(t, "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0", ublInv.CustomizationID)
		assert.Equal(t, "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0", ublInv.ProfileID.Value)
	})

}

func TestContextXRechnung(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")

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
		assert.Equal(t, "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0", ublInv.ProfileID.Value)
	})
}

func TestContextPeppolFranceCIUS(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		env := loadTestEnvelope(t, "france-cius/invoice-fr-cius.json")

		// Convert with France CIUS context
		doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextPeppolFranceCIUS))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// Verify OutputCustomizationID is used in the output
		assert.Equal(t, "urn:cen.eu:en16931:2017", ublInv.CustomizationID)
		// Verify ProfileID comes from the fr-ctc-billing-mode extension
		assert.Equal(t, "S1", ublInv.ProfileID.Value)
	})

	t.Run("external identification uses full CustomizationID", func(t *testing.T) {
		// Verify the context itself has the full identification
		assert.Equal(t, "urn:cen.eu:en16931:2017#compliant#urn:peppol:france:billing:cius:1.0", ubl.ContextPeppolFranceCIUS.CustomizationID)
		assert.Equal(t, "urn:peppol:france:billing:regulated", ubl.ContextPeppolFranceCIUS.ProfileID)
		assert.Equal(t, "urn:cen.eu:en16931:2017", ubl.ContextPeppolFranceCIUS.OutputCustomizationID)
	})
}

func TestContextPeppolFranceExtended(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		env := loadTestEnvelope(t, "france-extended/invoice-standard.json")

		// Convert with France Extended context
		doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextPeppolFranceExtended))
		require.NoError(t, err)

		ublInv, ok := doc.(*ubl.Invoice)
		require.True(t, ok)

		// Verify OutputCustomizationID is used
		assert.Equal(t, "urn:cen.eu:en16931:2017#conformant#urn.cpro.gouv.fr:1p0:extended-ctc-fr", ublInv.CustomizationID)
		// Verify ProfileID comes from the fr-ctc-billing-mode extension
		assert.Equal(t, "S1", ublInv.ProfileID.Value)
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
		env := loadTestEnvelope(t, "peppol-self-billed/self-billed-invoice.json")

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
		assert.Equal(t, "urn:fdc:peppol.eu:2017:poacc:selfbilling:01:1.0", ublInv.ProfileID.Value)
	})
}

func TestGetVESID(t *testing.T) {
	t.Run("invoice VESID for standard invoice", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Get VESID for Peppol context
		vesid := ubl.ContextPeppol.GetVESID(inv)
		assert.Equal(t, "eu.peppol.bis3:invoice:2026.5", vesid)

		// Get VESID for EN16931 context
		vesid = ubl.ContextEN16931.GetVESID(inv)
		assert.Equal(t, "eu.cen.en16931:ubl:1.3.16", vesid)

		// Get VESID for XRechnung context
		vesid = ubl.ContextXRechnung.GetVESID(inv)
		assert.Equal(t, "de.xrechnung:ubl-invoice:3.0.2", vesid)
	})

	t.Run("credit note VESID for credit note", func(t *testing.T) {
		env := loadTestEnvelope(t, "credit-note.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Verify it's a credit note
		require.True(t, inv.Type.In(bill.InvoiceTypeCreditNote))

		// Get VESID for Peppol context
		vesid := ubl.ContextPeppol.GetVESID(inv)
		assert.Equal(t, "eu.peppol.bis3:creditnote:2026.5", vesid)

		// Get VESID for EN16931 context
		vesid = ubl.ContextEN16931.GetVESID(inv)
		assert.Equal(t, "eu.cen.en16931:ubl-creditnote:1.3.16", vesid)
	})

	t.Run("self-billed invoice VESID", func(t *testing.T) {
		env := loadTestEnvelope(t, "peppol-self-billed/self-billed-invoice.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Get VESID for PeppolSelfBilled context
		vesid := ubl.ContextPeppolSelfBilled.GetVESID(inv)
		assert.Equal(t, "eu.peppol.bis3:invoice-self-billing:2026.5", vesid)
	})

	t.Run("France CIUS VESID", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Get VESID for France CIUS context
		vesid := ubl.ContextPeppolFranceCIUS.GetVESID(inv)
		assert.Equal(t, "fr.ctc:ubl-invoice:1.4.0-03", vesid)
	})

	t.Run("France Extended VESID", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		// Get VESID for France Extended context
		vesid := ubl.ContextPeppolFranceExtended.GetVESID(inv)
		assert.Equal(t, "fr.ctc:extended-ubl-invoice:1.4.0-03", vesid)
	})
}

func TestFrenchBillingModeResolution(t *testing.T) {
	// A French document is recognised by its billing mode in cbc:ProfileID; the
	// CustomizationID then only picks CIUS or Extended. Senders differ on which
	// identifier they put in the document: the one the profile emits, or the
	// spec-level one that identifies the profile on the network. Both resolve.
	const (
		specCIUS     = "urn:cen.eu:en16931:2017#compliant#urn:peppol:france:billing:cius:1.0"
		specExtended = "urn:cen.eu:en16931:2017#conformant#urn:peppol:france:billing:extended:1.0"
		docCIUS      = "urn:cen.eu:en16931:2017"
		docExtended  = "urn:cen.eu:en16931:2017#conformant#urn.cpro.gouv.fr:1p0:extended-ctc-fr"
	)

	for _, tt := range []struct {
		name            string
		customizationID string
		want            ubl.Context
	}{
		{"in-document CIUS", docCIUS, ubl.ContextPeppolFranceCIUS},
		{"in-document Extended", docExtended, ubl.ContextPeppolFranceExtended},
		{"spec-level CIUS", specCIUS, ubl.ContextPeppolFranceCIUS},
		{"spec-level Extended", specExtended, ubl.ContextPeppolFranceExtended},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, mode := range []string{"B1", "S1", "M4"} {
				ctx := ubl.FindContext(tt.customizationID, mode)
				require.NotNil(t, ctx, "mode %s", mode)
				assert.Equal(t, tt.want.CustomizationID, ctx.CustomizationID, "mode %s", mode)
				assert.Equal(t, tt.want.VESIDs.Invoice, ctx.VESIDs.Invoice, "mode %s", mode)
			}
		})
	}

	t.Run("spec-level CustomizationID still carries the addon through Convert", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(getParsePath(), "france-cius", "b2b-reg.xml"))
		require.NoError(t, err)
		old := []byte("<cbc:CustomizationID>" + docCIUS + "</cbc:CustomizationID>")
		require.Contains(t, string(data), string(old))
		data = bytes.Replace(data, old, []byte("<cbc:CustomizationID>"+specCIUS+"</cbc:CustomizationID>"), 1)

		doc, err := ubl.Parse(data)
		require.NoError(t, err)
		env, err := doc.(*ubl.Invoice).Convert()
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		assert.Contains(t, inv.GetAddons(), flow2.V1)
	})

	t.Run("unmodelled CustomizationID still parses best-effort", func(t *testing.T) {
		ctx := ubl.FindContext("urn:peppol:pint:billing-1@sg-1", "B1")
		assert.Nil(t, ctx)
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

	t.Run("find EN16931 when no ProfileID provided", func(t *testing.T) {
		ctx := ubl.FindContext("urn:cen.eu:en16931:2017", "")
		require.NotNil(t, ctx)
		assert.Equal(t, ubl.ContextEN16931.CustomizationID, ctx.CustomizationID)
	})

	t.Run("find EN16931 with non-billing-mode ProfileID", func(t *testing.T) {
		// EN16931 documents may have arbitrary ProfileIDs that are not French billing modes
		ctx := ubl.FindContext("urn:cen.eu:en16931:2017", "Invoicing on purchase order")
		require.NotNil(t, ctx)
		assert.Equal(t, ubl.ContextEN16931.CustomizationID, ctx.CustomizationID)
	})

	t.Run("find France CIUS by billing mode ProfileID", func(t *testing.T) {
		// France CIUS documents use EN16931 CustomizationID but have a billing mode as ProfileID
		ctx := ubl.FindContext("urn:cen.eu:en16931:2017", "B1")
		require.NotNil(t, ctx)
		assert.Equal(t, ubl.ContextPeppolFranceCIUS.CustomizationID, ctx.CustomizationID)

		ctx = ubl.FindContext("urn:cen.eu:en16931:2017", "S1")
		require.NotNil(t, ctx)
		assert.Equal(t, ubl.ContextPeppolFranceCIUS.CustomizationID, ctx.CustomizationID)

		ctx = ubl.FindContext("urn:cen.eu:en16931:2017", "M4")
		require.NotNil(t, ctx)
		assert.Equal(t, ubl.ContextPeppolFranceCIUS.CustomizationID, ctx.CustomizationID)
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
