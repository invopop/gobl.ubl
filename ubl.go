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
		})); err != nil {
			return nil, err
		}
		return in, nil

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

	doc := env.Extract()
	switch d := doc.(type) {
	case *bill.Invoice:
		// Check and add missing addons
		if err := ensureAddons(d, o.context.Addons); err != nil {
			return nil, err
		}

		// Removes included taxes as they are not supported in UBL
		if err := d.RemoveIncludedTaxes(); err != nil {
			return nil, fmt.Errorf("cannot convert invoice with included taxes: %w", err)
		}

		return ublInvoice(d, o)
	default:
		return nil, ErrUnsupportedDocumentType
	}
}

// ensureAddons checks if the invoice has all required addons and adds missing ones
func ensureAddons(inv *bill.Invoice, required []cbc.Key) error {
	if len(required) == 0 {
		return nil
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
	if err := inv.Calculate(); err != nil {
		return fmt.Errorf("gobl invoice missing addon %v: %w", missing, err)
	}
	if err := inv.Validate(); err != nil {
		return fmt.Errorf("gobl invoice missing addon %v: %w", missing, err)
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
	if inv, ok := in.(*Invoice); ok && inv.CustomizationID == ContextOIOUBL21.CustomizationID {
		// Legacy OIOUBL 2.1 requires scheme/list attributes on these elements.
		if inv.ProfileID != "" {
			raw := []byte(fmt.Sprintf("<cbc:ProfileID>%s</cbc:ProfileID>", inv.ProfileID))
			withAttrs := []byte(fmt.Sprintf("<cbc:ProfileID schemeAgencyID=\"320\" schemeID=\"urn:oioubl:id:profileid-1.4\">%s</cbc:ProfileID>", inv.ProfileID))
			b = bytes.ReplaceAll(b, raw, withAttrs)
		}
		if inv.InvoiceTypeCode != "" {
			raw := []byte(fmt.Sprintf("<cbc:InvoiceTypeCode>%s</cbc:InvoiceTypeCode>", inv.InvoiceTypeCode))
			withAttrs := []byte(fmt.Sprintf("<cbc:InvoiceTypeCode listAgencyID=\"320\" listID=\"urn:oioubl:codelist:invoicetypecode-1.1\">%s</cbc:InvoiceTypeCode>", inv.InvoiceTypeCode))
			b = bytes.ReplaceAll(b, raw, withAttrs)
		}
		if inv.CreditNoteTypeCode != "" {
			raw := []byte(fmt.Sprintf("<cbc:CreditNoteTypeCode>%s</cbc:CreditNoteTypeCode>", inv.CreditNoteTypeCode))
			withAttrs := []byte(fmt.Sprintf("<cbc:CreditNoteTypeCode listAgencyID=\"320\" listID=\"urn:oioubl:codelist:invoicetypecode-1.1\">%s</cbc:CreditNoteTypeCode>", inv.CreditNoteTypeCode))
			b = bytes.ReplaceAll(b, raw, withAttrs)
		}
	}
	return append([]byte(xml.Header), b...), nil
}
