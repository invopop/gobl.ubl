package ubl_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	oioubl "github.com/invopop/gobl.dk.oioubl/addon"
	ubl "github.com/invopop/gobl.ubl"
	en16931 "github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOIOUBL21AddonIntegration exercises the full stack against the pushed
// gobl addon branch: en16931 normalizes the UNTDID codes, dk-oioubl-v2-1
// validates the OIOUBL rules, and the OIOUBL Context converts to XML.
func TestOIOUBL21AddonIntegration(t *testing.T) {
	load := func(t *testing.T) (*bill.Invoice, *gobl.Envelope) {
		t.Helper()
		env, err := loadTestEnvelopeFromPath(filepath.Join(getConvertPath(), "oioubl21", "invoice-minimal.json"))
		require.NoError(t, err)
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		// Stack both addons: en16931 for UNTDID normalization, ours for OIOUBL.
		inv.Addons = tax.WithAddons(en16931.V2017, oioubl.V2_1)
		return inv, env
	}

	contact := func() []*org.Person {
		// F-INV051 (addon rule 20) requires the customer contact to carry an
		// identity code, which the converter emits as cac:Contact/cbc:ID.
		return []*org.Person{{
			Name:       &org.Name{Given: "Anders", Surname: "Jensen"},
			Identities: []*org.Identity{{Code: "EMP-7781"}},
		}}
	}

	t.Run("credit note with value_date emits TaxPointDate before CreditNoteTypeCode (XSD order)", func(t *testing.T) {
		inv, env := load(t)
		inv.Customer.People = contact()
		inv.Type = bill.InvoiceTypeCreditNote
		d := cal.MakeDate(2024, 6, 15)
		inv.ValueDate = &d

		require.NoError(t, env.Calculate())
		require.NoError(t, env.Validate())

		doc, err := ubl.ConvertInvoice(env, ubl.WithContext(ubl.ContextOIOUBL21))
		require.NoError(t, err)
		out, err := ubl.Bytes(doc)
		require.NoError(t, err)

		xml := string(out)
		tpd := strings.Index(xml, "<cbc:TaxPointDate>")
		tc := strings.Index(xml, "<cbc:CreditNoteTypeCode")
		require.Greater(t, tpd, -1, "TaxPointDate must be present")
		require.Greater(t, tc, -1, "CreditNoteTypeCode must be present")
		assert.Less(t, tpd, tc, "the UBL CreditNote XSD sequences TaxPointDate before CreditNoteTypeCode")
	})

	t.Run("valid OIOUBL invoice passes en16931 + dk-oioubl-v2-1 and converts", func(t *testing.T) {
		inv, env := load(t)
		// The convert fixture lacks a customer contact, which OIOUBL requires
		// (F-INV046); supply one so the document is genuinely OIOUBL-valid.
		inv.Customer.People = contact()

		require.NoError(t, env.Calculate())
		require.NoError(t, env.Validate(), "should pass both addons")

		doc, err := ubl.ConvertInvoice(env, ubl.WithContext(ubl.ContextOIOUBL21))
		require.NoError(t, err)
		require.NotNil(t, doc.ProfileID)
		assert.Equal(t, "OIOUBL-2.1", doc.CustomizationID)
	})

	t.Run("our addon fires: missing customer contact -> F-INV046", func(t *testing.T) {
		inv, env := load(t)
		inv.Customer.People = nil
		require.NoError(t, env.Calculate())
		err := env.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "F-INV046")
	})

	t.Run("addon derives the participant for a bare DK supplier", func(t *testing.T) {
		inv, env := load(t)
		inv.Customer.People = contact()
		inv.Supplier.Inboxes = nil
		require.NoError(t, env.Calculate())
		require.NoError(t, env.Validate())
		require.NotEmpty(t, inv.Supplier.Endpoints)
	})

	t.Run("our addon fires: no participant or tax ID code -> F-INV031", func(t *testing.T) {
		inv, env := load(t)
		inv.Customer.People = contact()
		inv.Supplier.Inboxes = nil
		inv.Supplier.TaxID = &tax.Identity{Country: "DK"}
		require.NoError(t, env.Calculate())
		err := env.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "F-INV031")
	})
}
