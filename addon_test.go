package ubl_test

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
	"github.com/invopop/gobl/tax"
)

// testAddon requires invoices to include ordering details, so tests can check
// that a format applies its addons without depending on a real addon.
const testAddon cbc.Key = "test-ubl-v1"

func init() {
	tax.RegisterAddonDef(&tax.AddonDef{
		Key:  testAddon,
		Name: i18n.String{i18n.EN: "Test UBL"},
	})
	rules.RegisterWithGuard(
		testAddon.String(),
		rules.GOBL.Add("TEST-UBL"),
		is.InContext(tax.AddonIn(testAddon)),
		rules.For(new(bill.Invoice),
			rules.Field("ordering",
				rules.Assert("01", "ordering is required", is.Present),
			),
		),
	)
}
