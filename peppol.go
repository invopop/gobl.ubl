package ubl

import (
	"strings"

	"github.com/invopop/gobl/bill"
)

// LayerPeppolBilling applies Peppol BIS Billing rules on top of the base
// conversion.
var LayerPeppolBilling = &Layer{
	ConvertInvoice: func(_ *Context, _ *bill.Invoice, out *Invoice) error {
		// Peppol only allows one note, so concatenate all notes
		if len(out.Note) > 1 {
			out.Note = []string{strings.Join(out.Note, "\n\n")}
		}
		return nil
	},
}

// LayerPeppolInvoiceResponse applies the Peppol BIS Invoice Response data
// model on top of the base application response.
var LayerPeppolInvoiceResponse = &Layer{
	ConvertStatus: func(_ *Context, st *bill.Status, out *ApplicationResponse) error {
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
	},
	ParseStatus: func(_ *Context, in *ApplicationResponse, out *bill.Status) error {
		for i, dr := range in.DocumentResponse {
			if dr != nil {
				applyPeppolStatusLine(out.Lines[i], dr)
			}
		}
		return nil
	},
}
