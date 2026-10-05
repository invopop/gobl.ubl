package ubl

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
)

// Layer adds the behavior of a specification on top of the base conversion.
// Every function is optional, and is called after the base conversion of the
// element it receives, in the order the context lists its layers.
type Layer struct {
	// ConvertParty adjusts every party converted from GOBL, including nested
	// ones.
	ConvertParty func(ctx *Context, p *org.Party, out *Party)
	// ConvertInvoice adjusts the UBL invoice or credit note converted from GOBL.
	ConvertInvoice func(ctx *Context, inv *bill.Invoice, out *Invoice) error
	// ConvertStatus adjusts the UBL application response converted from GOBL.
	ConvertStatus func(ctx *Context, st *bill.Status, out *ApplicationResponse) error
	// ParseParty adjusts every party parsed into GOBL, including nested ones.
	ParseParty func(ctx *Context, in *Party, out *org.Party)
	// ParseInvoice adjusts the GOBL invoice parsed from UBL, before it is
	// calculated.
	ParseInvoice func(ctx *Context, in *Invoice, out *bill.Invoice) error
	// ParseStatus adjusts the GOBL status parsed from UBL, before it is
	// calculated.
	ParseStatus func(ctx *Context, in *ApplicationResponse, out *bill.Status) error
}

func (c *Context) convertParty(p *org.Party, out *Party) {
	for _, l := range c.Layers {
		if l.ConvertParty != nil {
			l.ConvertParty(c, p, out)
		}
	}
}

func (c *Context) convertInvoice(inv *bill.Invoice, out *Invoice) error {
	for _, l := range c.Layers {
		if l.ConvertInvoice != nil {
			if err := l.ConvertInvoice(c, inv, out); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Context) convertStatus(st *bill.Status, out *ApplicationResponse) error {
	for _, l := range c.Layers {
		if l.ConvertStatus != nil {
			if err := l.ConvertStatus(c, st, out); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Context) parseParty(in *Party, out *org.Party) {
	for _, l := range c.Layers {
		if l.ParseParty != nil {
			l.ParseParty(c, in, out)
		}
	}
}

func (c *Context) parseInvoice(in *Invoice, out *bill.Invoice) error {
	for _, l := range c.Layers {
		if l.ParseInvoice != nil {
			if err := l.ParseInvoice(c, in, out); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Context) parseStatus(in *ApplicationResponse, out *bill.Status) error {
	for _, l := range c.Layers {
		if l.ParseStatus != nil {
			if err := l.ParseStatus(c, in, out); err != nil {
				return err
			}
		}
	}
	return nil
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
