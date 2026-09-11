// Package ubl helps convert GOBL into UBL documents and vice versa.
package ubl

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/xmlctx"
)

// setEnvelopeRouting records the transport addresses the document was received
// with (opts' WithRouting) on the envelope's Head.From / Head.To, BEFORE the
// envelope is calculated. The addresses are the fully-qualified participant
// URIs the routing layer supplies and are recorded verbatim. GOBL's
// normalizeRouting only fills empty routing fields, so setting them first makes
// it respect them rather than derive them from the document assuming an
// outgoing (supplier→customer) direction — wrong for a received document.
// Applies to any document type.
func setEnvelopeRouting(env *gobl.Envelope, o *options) {
	env.Head.From = o.from
	env.Head.To = o.to
}

var (
	// ErrUnknownDocumentType is returned when the document type
	// is not recognized during parsing.
	ErrUnknownDocumentType = fmt.Errorf("unknown document type")

	// ErrUnsupportedDocumentType is returned when the document type
	// is not supported for conversion.
	ErrUnsupportedDocumentType = fmt.Errorf("unsupported document type")
)

// Version is the version of UBL documents that will be generated
// by this package.
const Version = "2.1"

// Parse parses a raw UBL document and returns the underlying Go struct.
// The returned value should be type asserted to the appropriate type.
//
// Supported types:
//   - *Invoice (for both Invoice and CreditNote documents)
//
// Example usage:
//
//	doc, err := ubl.Parse(xmlData)
//	if err != nil {
//	    // handle error
//	}
//	if inv, ok := doc.(*ubl.Invoice); ok {
//	    env, err := inv.Convert()
//	    attachments := inv.ExtractBinaryAttachments()
//	    // ...
//	}
func Parse(data []byte) (any, error) {
	ns, err := extractRootNamespace(data)
	if err != nil {
		return nil, err
	}

	switch ns {
	case NamespaceUBLInvoice, NamespaceUBLCreditNote:
		in := new(Invoice)
		if err := xmlctx.Unmarshal(data, in, xmlctx.WithNamespaces(map[string]string{
			"":     ns,
			"cbc":  NamespaceCBC,
			"cac":  NamespaceCAC,
			"qdt":  NamespaceQDT,
			"udt":  NamespaceUDT,
			"ccts": NamespaceCCTS,
			"xsi":  NamespaceXSI,
			"ext":  NamespaceEXT,
		})); err != nil {
			return nil, err
		}
		return in, nil

	case NamespaceUBLApplicationResponse:
		ar := new(ApplicationResponse)
		if err := xmlctx.Unmarshal(data, ar, xmlctx.WithNamespaces(map[string]string{
			"":    ns,
			"cbc": NamespaceCBC,
			"cac": NamespaceCAC,
		})); err != nil {
			return nil, err
		}
		return ar, nil

	// Future document types can be added here
	// case NamespaceUBLOrder:
	//     order := new(Order)
	//     if err := xmlctx.Parse(data, order, xmlctx.WithNamespaces(map[string]string{
	//         "cbc":  NamespaceCBC,
	//         "cac":  NamespaceCAC,
	//         "qdt":  NamespaceQDT,
	//         "udt":  NamespaceUDT,
	//         "ccts": NamespaceCCTS,
	//         "xsi":  NamespaceXSI,
	//         "ext":  "urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2",
	//     })); err != nil {
	//         return nil, err
	//     }
	//     return order, nil

	default:
		return nil, ErrUnknownDocumentType
	}
}

// Convert takes a GOBL envelope and converts to a UBL document of one
// of the supported types.
//
// Add a WithContext option to specify the desired UBL Guideline and Profile ID.
// If none is provided, EN16931 will be used by default.
func Convert(env *gobl.Envelope, opts ...Option) (any, error) {
	o := &options{
		context: ContextEN16931,
	}
	for _, opt := range opts {
		opt(o)
	}

	switch doc := env.Extract().(type) {
	case *bill.Invoice:
		// Check and add missing addons
		if err := ensureAddons(env, o.context.Addons); err != nil {
			return nil, err
		}
		// Removes included taxes as they are not supported in UBL
		if err := doc.RemoveIncludedTaxes(); err != nil {
			return nil, fmt.Errorf("cannot convert invoice with included taxes: %w", err)
		}
		if err := roundToCurrency(doc); err != nil {
			return nil, fmt.Errorf("cannot round invoice to currency precision: %w", err)
		}
		return ublInvoice(doc, o)
	case *bill.Status:
		return ublApplicationResponse(doc, o), nil
	default:
		return nil, ErrUnsupportedDocumentType
	}
}

// roundToCurrency recalculates an invoice using the currency rounding rule so
// that every monetary amount fits the number of decimals UBL allows.
//
// EN 16931's BR-DEC-* rules and UBL-DT-01 cap almost all amounts — line net
// amounts (BT-131), tax bases (BT-116), document totals — at the currency's
// precision, two decimals for most currencies. GOBL's `precise` rounding rule
// keeps line totals at a higher precision, and since v0.505
// RemoveIncludedTaxes switches a document away from `currency` rounding so
// that the tax-exclusive amounts it derives still add up to the original
// tax-inclusive ones. Emitting those line totals as-is breaks BR-DEC-23, and
// rounding them individually on the way out breaks BR-CO-10, as the rounded
// lines no longer sum to the document total.
//
// Recalculating instead rounds every amount consistently, and any difference
// against the amount originally payable is carried in the invoice's rounding
// total, which UBL expresses as the payable rounding amount (BT-114).
func roundToCurrency(inv *bill.Invoice) error {
	if !exceedsCurrencyPrecision(inv) {
		return nil
	}
	payable := inv.Totals.Payable

	if inv.Tax == nil {
		inv.Tax = new(bill.Tax)
	}
	inv.Tax.Rounding = tax.RoundingRuleCurrency
	if err := inv.Calculate(); err != nil {
		return err
	}

	// Preserve the amount actually owed: rounding each line to the currency
	// may shift the total by a cent or two.
	diff := payable.Subtract(inv.Totals.Payable)
	if diff.IsZero() {
		return nil
	}
	rnd := diff
	if inv.Totals.Rounding != nil {
		rnd = inv.Totals.Rounding.Add(diff)
	}
	inv.Totals.Rounding = &rnd
	return inv.Calculate()
}

// exceedsCurrencyPrecision reports whether the invoice holds an amount with
// more decimals than its currency allows, and so cannot be written out as it
// stands. Only the amounts the calculator keeps at the rounding rule's
// precision are checked; the document totals are always rounded to the
// currency, and unit prices (BT-146, BT-148) are exempt from the BR-DEC rules.
//
// Invoices already within the currency's precision are left untouched, so a
// document that needs no rounding is never recalculated.
func exceedsCurrencyPrecision(inv *bill.Invoice) bool {
	if inv.Totals == nil {
		return false
	}
	def := inv.Currency.Def()
	if def == nil {
		return false
	}
	exp := def.Subunits

	over := func(a *num.Amount) bool {
		return a != nil && a.Exp() > exp
	}
	for _, l := range inv.Lines {
		if over(l.Total) || over(l.Sum) {
			return true
		}
		for _, d := range l.Discounts {
			if over(&d.Amount) {
				return true
			}
		}
		for _, c := range l.Charges {
			if over(&c.Amount) {
				return true
			}
		}
	}
	for _, d := range inv.Discounts {
		if over(&d.Amount) || over(d.Base) {
			return true
		}
	}
	for _, c := range inv.Charges {
		if over(&c.Amount) || over(c.Base) {
			return true
		}
	}
	if t := inv.Totals.Taxes; t != nil {
		for _, cat := range t.Categories {
			for _, r := range cat.Rates {
				if over(&r.Base) || over(&r.Amount) {
					return true
				}
			}
		}
	}
	return false
}

// ensureAddons checks if the invoice has all required addons and adds missing ones
func ensureAddons(env *gobl.Envelope, required []cbc.Key) error {
	if len(required) == 0 {
		return nil
	}

	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return fmt.Errorf("expected bill.Invoice, got %T", env.Extract())
	}

	var missing []cbc.Key
	existing := inv.GetAddons()
	for _, addon := range required {
		if !addon.In(existing...) {
			missing = append(missing, addon)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	inv.SetAddons(append(existing, missing...)...)
	if err := env.Calculate(); err != nil {
		return err
	}
	if err := env.Validate(); err != nil {
		return err
	}
	return nil
}

func extractRootNamespace(data []byte) (string, error) {
	dc := xml.NewDecoder(bytes.NewReader(data))
	for {
		tk, err := dc.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("error parsing XML: %w", err)
		}
		switch t := tk.(type) {
		case xml.StartElement:
			return t.Name.Space, nil // Extract and return the namespace
		}
	}
	return "", ErrUnknownDocumentType
}

// Bytes returns the raw XML of the UBL document including
// the XML Header.
func Bytes(in any) ([]byte, error) {
	b, err := xml.MarshalIndent(in, "", "  ")
	if err != nil {
		return nil, err
	}

	// Go's xml.Marshal encodes single quotes as &#39,
	// this is a quick fix
	b = bytes.ReplaceAll(b, []byte("&#39;"), []byte("'"))
	return append([]byte(xml.Header), b...), nil
}

// BytesCompact returns the raw XML of the UBL document without
// indentation, including the XML Header.
func BytesCompact(in any) ([]byte, error) {
	b, err := xml.Marshal(in)
	if err != nil {
		return nil, err
	}
	b = bytes.ReplaceAll(b, []byte("&#39;"), []byte("'"))
	return append([]byte(xml.Header), b...), nil
}
