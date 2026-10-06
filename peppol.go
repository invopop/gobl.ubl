package ubl

import (
	"strings"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
)

// exportPeppolBilling applies the Peppol BIS Billing rules to an exported
// invoice.
func exportPeppolBilling(_ *Format, _ *gobl.Envelope, doc Document) error {
	out, ok := doc.(*Invoice)
	if !ok {
		return nil
	}
	// Peppol only allows one note, so concatenate all notes
	if len(out.Note) > 1 {
		out.Note = []string{strings.Join(out.Note, "\n\n")}
	}
	return nil
}

// exportPeppolInvoiceResponse applies the Peppol BIS Invoice Response data
// model to an exported application response.
func exportPeppolInvoiceResponse(_ *Format, env *gobl.Envelope, doc Document) error {
	st, ok := env.Extract().(*bill.Status)
	out, ok2 := doc.(*ApplicationResponse)
	if !ok || !ok2 {
		return nil
	}
	// T111's data model omits these root elements and restricts the parties.
	out.UBLVersionID = ""
	out.UUID = ""
	trimToResponseParty(out.SenderParty)
	trimToResponseParty(out.ReceiverParty)
	for i, dr := range out.DocumentResponse {
		// Description is valid generic UBL but is not part of the Peppol
		// Invoice Response Response.
		dr.Response.Description = nil
		applyPeppolDocumentResponse(dr, st.Lines[i])
	}
	return nil
}

// importPeppolInvoiceResponse maps the Peppol Invoice Response codes of an
// imported application response.
func importPeppolInvoiceResponse(_ *Format, doc Document, env *gobl.Envelope) error {
	in, ok := doc.(*ApplicationResponse)
	st, ok2 := env.Extract().(*bill.Status)
	if !ok || !ok2 {
		return nil
	}
	for i, dr := range in.DocumentResponse {
		if dr != nil {
			applyPeppolStatusLine(st.Lines[i], dr)
		}
	}
	return nil
}
