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
	"github.com/invopop/xmlctx"
)

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
//   - *ApplicationResponse (for OIOUBL invoice responses)
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
		if err := xmlctx.Unmarshal(data, ar, xmlctx.WithNamespaces(cbcCacNamespaces(ns))); err != nil {
			return nil, err
		}
		return ar, nil

	case NamespaceUBLReminder:
		rem := new(Reminder)
		if err := xmlctx.Unmarshal(data, rem, xmlctx.WithNamespaces(cbcCacNamespaces(ns))); err != nil {
			return nil, err
		}
		return rem, nil

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
		return ublInvoice(doc, o)
	case *bill.Status:
		// The ApplicationResponse converter maps only; it neither auto-adds
		// addons nor validates (a keyless or partial status is a generic UBL
		// shape). Correctness is gated by the addon rules at envelope validation
		// and by schematron downstream.
		return ublApplicationResponse(doc, o), nil
	case *bill.Payment:
		// A payment request maps to the OIOUBL Reminder (Rykker). Add any missing
		// addons so the reminder-type/sequence rules surface, then convert.
		if err := ensureAddons(env, o.context.Addons); err != nil {
			return nil, err
		}
		return ublReminder(doc, o), nil
	default:
		return nil, ErrUnsupportedDocumentType
	}
}

// addonsDocument is implemented by the GOBL document types that carry addons.
type addonsDocument interface {
	GetAddons() []cbc.Key
	SetAddons(...cbc.Key)
}

// ensureAddons checks if the document has all required addons and adds the
// missing ones, recalculating and revalidating the envelope so any rule the
// newly added addon enforces is surfaced (as a *gobl.Error carrying the faults).
func ensureAddons(env *gobl.Envelope, required []cbc.Key) error {
	if len(required) == 0 {
		return nil
	}

	doc, ok := env.Extract().(addonsDocument)
	if !ok {
		return ErrUnsupportedDocumentType
	}

	var missing []cbc.Key
	existing := doc.GetAddons()
	for _, addon := range required {
		if !addon.In(existing...) {
			missing = append(missing, addon)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	doc.SetAddons(append(existing, missing...)...)
	if err := env.Calculate(); err != nil {
		return err
	}
	return env.Validate()
}

// cbcCacNamespaces returns the namespace prefix map for the simpler UBL
// documents (ApplicationResponse, Reminder) that use only the cbc and cac
// component namespaces alongside their root namespace.
func cbcCacNamespaces(ns string) map[string]string {
	return map[string]string{
		"":    ns,
		"cbc": NamespaceCBC,
		"cac": NamespaceCAC,
	}
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

	if creditNoteNeedsTaxPointDateReorder(in) {
		b = reorderCreditNoteTaxPointDate(b)
	}

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

	if creditNoteNeedsTaxPointDateReorder(in) {
		b = reorderCreditNoteTaxPointDate(b)
	}

	return append([]byte(xml.Header), b...), nil
}

// creditNoteNeedsTaxPointDateReorder reports whether in is a credit note that
// carries a cbc:TaxPointDate. The shared Invoice struct emits TaxPointDate after
// CreditNoteTypeCode (correct for Invoice), but the UBL CreditNote XSD sequences
// it before — see reorderCreditNoteTaxPointDate.
func creditNoteNeedsTaxPointDateReorder(in any) bool {
	var inv *Invoice
	switch v := in.(type) {
	case *Invoice:
		inv = v
	case Invoice:
		inv = &v
	default:
		return false
	}
	return inv != nil && inv.XMLName.Local == rootNameCreditNote && inv.TaxPointDate != ""
}

// reorderCreditNoteTaxPointDate moves the cbc:TaxPointDate element ahead of
// cbc:CreditNoteTypeCode so the output matches the UBL CreditNote XSD sequence
// (TaxPointDate precedes the type code). Invoice and CreditNote share one Go
// struct whose field order is correct for Invoice but not CreditNote, and Go's
// xml package can neither express two element orders for one struct nor survive a
// decode/re-encode pass (it mangles the cac:/cbc: prefixes). Adjusting the
// already-marshaled bytes keeps the exact prefixes and indentation, and only runs
// for the rare credit note that carries a TaxPointDate.
func reorderCreditNoteTaxPointDate(b []byte) []byte {
	const (
		open     = "<cbc:TaxPointDate>"
		closeTag = "</cbc:TaxPointDate>"
		typeCode = "<cbc:CreditNoteTypeCode"
	)

	tpd := bytes.Index(b, []byte(open))
	tc := bytes.Index(b, []byte(typeCode))
	if tpd < 0 || tc < 0 || tpd < tc {
		return b // type code absent, or already correctly ordered
	}
	rel := bytes.Index(b[tpd:], []byte(closeTag))
	if rel < 0 {
		return b
	}
	elemEnd := tpd + rel + len(closeTag)
	elem := append([]byte(nil), b[tpd:elemEnd]...)

	// Drop the element together with the newline + indent that preceded it.
	cut := tpd
	for cut > 0 && (b[cut-1] == ' ' || b[cut-1] == '\t') {
		cut--
	}
	if cut > 0 && b[cut-1] == '\n' {
		cut--
	}
	rest := append(append([]byte(nil), b[:cut]...), b[elemEnd:]...)

	// Re-insert it before the type code, reusing that line's leading whitespace
	// (empty for the compact, non-indented output).
	tc = bytes.Index(rest, []byte(typeCode))
	indentStart := tc
	for indentStart > 0 && (rest[indentStart-1] == ' ' || rest[indentStart-1] == '\t') {
		indentStart--
	}
	if indentStart > 0 && rest[indentStart-1] == '\n' {
		indentStart--
	}
	sep := rest[indentStart:tc]

	out := make([]byte, 0, len(rest)+len(elem)+len(sep))
	out = append(out, rest[:tc]...)
	out = append(out, elem...)
	out = append(out, sep...)
	out = append(out, rest[tc:]...)
	return out
}
