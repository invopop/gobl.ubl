package ubl

import (
	"strings"

	"github.com/invopop/gobl/uuid"
)

// TaxDataInvoiceUUIDName joins the values that identify an invoice into the
// name hashed into the invoice UUID: each value is trimmed of leading and
// trailing whitespace (space, tab, CR, LF) and kept as it is otherwise, and the
// values are joined with one space. Any empty value makes the invoice not
// reportable.
func TaxDataInvoiceUUIDName(values ...string) (string, error) {
	out := make([]string, len(values))
	for i, v := range values {
		v = strings.Trim(v, " \t\r\n")
		if v == "" {
			return "", ErrTaxDataNotReportable
		}
		out[i] = v
	}
	return strings.Join(out, " "), nil
}

// TaxDataInvoiceUUID calculates the invoice UUID (TDT-017) from the name.
func TaxDataInvoiceUUID(name string) string {
	return uuid.V5(uuid.MustParse(TaxDataInvoiceUUIDNamespace), []byte(name)).String()
}

// vidaInvoiceUUIDName uses the seller VAT identifier (BT-31), the type code
// (BT-3), the invoice number (BT-1) and the issue date (BT-2). An invoice
// without a seller VAT identifier is not reportable (ViDA TDD BIS 2.2.6).
func vidaInvoiceUUIDName(inv *Invoice) (string, error) {
	vat := ""
	if pts := vatTaxScheme(inv.AccountingSupplierParty.Party); pts != nil {
		vat = pts.CompanyID.Value
	}
	return TaxDataInvoiceUUIDName(vat, typeCodeOf(inv), inv.ID, inv.IssueDate)
}
