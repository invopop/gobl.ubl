package ubl

import (
	"fmt"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

// ExportFunc adjusts the UBL document exported from the GOBL envelope for a
// format.
type ExportFunc func(f *Format, env *gobl.Envelope, doc Document) error

// ImportFunc adjusts the GOBL envelope imported from the UBL document for a
// format. Format.ImportFuncs and Format.ReconciledImportFuncs say when it
// runs.
type ImportFunc func(f *Format, doc Document, env *gobl.Envelope) error

func (f *Format) runExportFuncs(env *gobl.Envelope, doc Document) error {
	for _, fn := range f.ExportFuncs {
		if err := fn(f, env, doc); err != nil {
			return err
		}
	}
	return nil
}

func (f *Format) runImportFuncs(doc Document, env *gobl.Envelope) error {
	for _, fn := range f.ImportFuncs {
		if err := fn(f, doc, env); err != nil {
			return err
		}
	}
	return nil
}

// runReconciledImportFuncs runs the format's reconciled import functions on an
// invoice not tagged for bypass, calculates the envelope again, and checks
// that the amounts the buyer owes did not change.
func (f *Format) runReconciledImportFuncs(doc Document, env *gobl.Envelope) error {
	if len(f.ReconciledImportFuncs) == 0 {
		return nil
	}
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok || inv.HasTags(tax.TagBypass) {
		return nil
	}
	before := amountsOwedBy(inv.Totals)
	for _, fn := range f.ReconciledImportFuncs {
		if err := fn(f, doc, env); err != nil {
			return err
		}
	}
	if err := env.Calculate(); err != nil {
		return err
	}
	// A function may have replaced the document, so check the one there now.
	after, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return fmt.Errorf("format %s: reconciled import replaced the invoice with %T", f.Key, env.Extract())
	}
	// Only what the buyer owes is compared, not every declared total: a
	// restatement can rightly move an amount from the tax-exclusive total
	// (BT-109) into the taxes, which the reconciliation check would report
	// as a mismatch.
	if now := amountsOwedBy(after.Totals); !now.equals(before) {
		return fmt.Errorf("format %s: reconciled import changed the amounts owed: was %s; now %s", f.Key, before, now)
	}
	return nil
}

// amountsOwed holds the totals a reconciled import must leave as they were.
type amountsOwed struct {
	totalWithTax num.Amount
	payable      num.Amount
	due          *num.Amount
}

func amountsOwedBy(t *bill.Totals) amountsOwed {
	o := amountsOwed{totalWithTax: t.TotalWithTax, payable: t.Payable}
	if t.Due != nil {
		due := *t.Due
		o.due = &due
	}
	return o
}

// equals reports whether both hold the same amounts, with no amount due on
// either counting as the same.
func (o amountsOwed) equals(o2 amountsOwed) bool {
	if !o.totalWithTax.Equals(o2.totalWithTax) || !o.payable.Equals(o2.payable) {
		return false
	}
	if o.due == nil || o2.due == nil {
		return o.due == nil && o2.due == nil
	}
	return o.due.Equals(*o2.due)
}

func (o amountsOwed) String() string {
	due := "none"
	if o.due != nil {
		due = o.due.String()
	}
	return fmt.Sprintf("total with tax %s, payable %s, due %s", o.totalWithTax, o.payable, due)
}

// PartyPair links a GOBL party with the UBL party it maps to.
type PartyPair struct {
	GOBL *org.Party
	UBL  *Party
}

// InvoiceParties lists the parties of the GOBL invoice alongside the UBL
// parties they map to in the UBL invoice, in either direction. Agents follow
// the party they act for, and parties missing on either side are left out.
func InvoiceParties(inv *bill.Invoice, doc *Invoice) []PartyPair {
	var pairs []PartyPair
	add := func(p *org.Party, out *Party) {
		if p == nil || out == nil {
			return
		}
		pairs = append(pairs, PartyPair{GOBL: p, UBL: out})
		if p.Agent != nil && out.AgentParty != nil {
			pairs = append(pairs, PartyPair{GOBL: p.Agent, UBL: out.AgentParty})
		}
	}
	serviceProvider := func(p *Party) *Party {
		if p == nil || p.ServiceProviderParty == nil {
			return nil
		}
		return p.ServiceProviderParty.Party
	}

	add(inv.Supplier, doc.AccountingSupplierParty.Party)
	add(inv.Customer, doc.AccountingCustomerParty.Party)
	if o := inv.Ordering; o != nil {
		add(o.Seller, doc.TaxRepresentativeParty)
		add(o.Issuer, serviceProvider(doc.AccountingSupplierParty.Party))
		add(o.Buyer, serviceProvider(doc.AccountingCustomerParty.Party))
	}
	if p := inv.Payment; p != nil {
		add(p.Payee, doc.PayeeParty)
		if len(doc.PaymentMeans) > 0 && doc.PaymentMeans[0].PaymentMandate != nil {
			add(p.Payer, doc.PaymentMeans[0].PaymentMandate.PayerParty)
		}
	}
	if lines := doc.partyLines(); len(lines) == len(inv.Lines) {
		for i, l := range inv.Lines {
			if lines[i].Item != nil {
				add(l.Seller, lines[i].Item.ManufacturerParty)
			}
		}
	}
	return pairs
}

// FormatDate formats the date as UBL expects it.
func FormatDate(date cal.Date) string {
	return formatDate(date)
}

// NewAmount builds a monetary amount in the currency, rounded to the
// currency's precision.
func NewAmount(a num.Amount, ccy string) Amount {
	return newAmount(a, ccy)
}

// TaxPointCode provides the UNTDID 2005 code for the GOBL tax point key.
func TaxPointCode(point cbc.Key) (string, bool) {
	code, ok := taxPointCodeMap[point]
	return code, ok
}

// CleanString removes the replacement characters left by broken text
// encodings from incoming text.
func CleanString(s string) string {
	return cleanString(s)
}

// ContactName provides a person's name the way a party contact (BG-6) is
// written.
func ContactName(n *org.Name) string {
	return contactName(n)
}
