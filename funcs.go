package ubl

import (
	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
)

// ExportFunc adjusts the UBL document exported from the GOBL envelope for a
// format.
type ExportFunc func(f *Format, env *gobl.Envelope, doc Document) error

// ImportFunc adjusts the GOBL envelope imported from the UBL document for a
// format, before the envelope is calculated.
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

// FoldToCore keeps what the EN 16931 core model allows once and a conformant
// extension may repeat: one buyer identifier (BT-46, UBL-SR-16), one payee
// identifier besides the SEPA creditor identifier (BT-60, UBL-SR-20) and one
// note per line (BT-127, UBL-SR-34). Formats that restrict the core, such as
// a CIUS, list it last in their ExportFuncs so it also covers what the others
// add.
func FoldToCore(_ *Format, _ *gobl.Envelope, doc Document) error {
	ui, ok := doc.(*Invoice)
	if !ok {
		return nil
	}
	if p := ui.AccountingCustomerParty.Party; p != nil {
		p.PartyIdentification = foldIdentifications(p.PartyIdentification)
	}
	if p := ui.PayeeParty; p != nil {
		p.PartyIdentification = foldIdentifications(p.PartyIdentification)
	}
	for _, lines := range [][]InvoiceLine{ui.InvoiceLines, ui.CreditNoteLines} {
		for i := range lines {
			lines[i].Note = joinNotes(lines[i].Note)
		}
	}
	return nil
}
