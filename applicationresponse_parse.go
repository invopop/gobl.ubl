package ubl

import (
	"cloud.google.com/go/civil"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/gobl/uuid"
)

// Convert turns a parsed UBL ApplicationResponse into a GOBL envelope wrapping a
// bill.Status.
func (ar *ApplicationResponse) Convert() (*gobl.Envelope, error) {
	o := new(options)
	profileID := ""
	if ar.ProfileID != nil {
		profileID = ar.ProfileID.Value
	}
	// Resolve by CustomizationID + ProfileID first, then fall back to the
	// CustomizationID alone. OIOUBL keeps a single CustomizationID across all
	// response types but swaps in a technical-response ProfileID for
	// acknowledgements (F-APR057/F-APR058), which would otherwise fail the
	// ProfileID-qualified lookup.
	ctx := FindContext(ar.CustomizationID, profileID)
	if ctx == nil {
		ctx = FindContext(ar.CustomizationID, "")
	}
	if ctx != nil {
		o.context = *ctx
	}

	st, err := ar.goblStatus(o)
	if err != nil {
		return nil, err
	}

	env := gobl.NewEnvelope()
	if err := env.Insert(st); err != nil {
		return nil, err
	}
	return env, nil
}

func (ar *ApplicationResponse) goblStatus(o *options) (*bill.Status, error) {
	out := &bill.Status{
		Addons:   tax.Addons{List: o.context.Addons},
		Type:     bill.StatusTypeResponse,
		Code:     cbc.Code(ar.ID),
		Supplier: goblParty(ar.ReceiverParty, o),
		Customer: goblParty(ar.SenderParty, o),
	}

	issueDate, err := parseDate(ar.IssueDate)
	if err != nil {
		return nil, err
	}
	out.IssueDate = issueDate

	if ar.IssueTime != "" {
		ct, err := civil.ParseTime(ar.IssueTime)
		if err != nil {
			return nil, err
		}
		out.IssueTime = &cal.Time{Time: ct}
	}

	for _, n := range ar.Note {
		out.Notes = append(out.Notes, &org.Note{Text: n})
	}

	for _, dr := range ar.DocumentResponse {
		line, err := goblStatusLine(dr, o)
		if err != nil {
			return nil, err
		}
		out.Lines = append(out.Lines, line)
	}

	return out, nil
}

// goblStatusLine maps the generic parts of a single UBL DocumentResponse. The
// response code and the status clarifications are context specific.
func goblStatusLine(dr *DocumentResponse, o *options) (*bill.StatusLine, error) {
	line := new(bill.StatusLine)
	if dr == nil {
		return line, nil
	}

	if r := dr.Response; r != nil {
		if len(r.Description) > 0 {
			line.Description = r.Description[0]
		}
		if r.EffectiveDate != "" {
			d, err := parseDate(r.EffectiveDate)
			if err != nil {
				return nil, err
			}
			line.Date = &d
		}
	}

	if ref := dr.DocumentReference; ref != nil {
		doc := &org.DocumentRef{
			Code: cbc.Code(ref.ID),
		}
		if ref.UUID != "" {
			doc.UUID = uuid.UUID(ref.UUID)
		}
		if ref.IssueDate != "" {
			d, err := parseDate(ref.IssueDate)
			if err != nil {
				return nil, err
			}
			doc.IssueDate = &d
		}
		line.Doc = doc
	}

	if o.context.Is(ContextPeppolInvoiceResponse) {
		applyPeppolStatusLine(line, dr)
	}
	if o.context.Is(ContextOIOUBL21) {
		applyOIOUBL21StatusLine(line, dr)
	}

	return line, nil
}

// goblResponseEvents reverses oioublResponseCodes, mapping an OIOUBL
// responsecode-1.1 value back to a GOBL status event.
var goblResponseEvents = map[string]cbc.Key{
	responseCodeBusinessAccept:  bill.StatusEventAccepted,
	responseCodeBusinessReject:  bill.StatusEventRejected,
	responseCodeTechnicalAccept: bill.StatusEventAcknowledged,
	responseCodeTechnicalReject: bill.StatusEventError,
	responseCodeProfileReject:   bill.StatusEventError,
}

// applyOIOUBL21StatusLine maps the OIOUBL 2.1 code-list values on a parsed
// ApplicationResponse back to GOBL: the responsecode-1.1 response code to a
// status event, and the responsedocumenttypecode-1.1 value to a document type.
func applyOIOUBL21StatusLine(line *bill.StatusLine, dr *DocumentResponse) {
	if r := dr.Response; r != nil && r.ResponseCode != nil {
		line.Key = goblResponseEvents[r.ResponseCode.Value]
	}
	if ref := dr.DocumentReference; ref != nil && line.Doc != nil &&
		ref.DocumentTypeCode != nil && ref.DocumentTypeCode.Value == responseDocTypeCreditNote {
		line.Doc.Type = bill.InvoiceTypeCreditNote
	}
}

// applyPeppolStatusLine maps the Peppol Invoice Response codes back to GOBL: the
// UNCL4343 response code to a status event, and each OPStatusReason /
// OPStatusAction cac:Status clarification to a reason or action.
func applyPeppolStatusLine(line *bill.StatusLine, dr *DocumentResponse) {
	r := dr.Response
	if r == nil {
		return
	}
	if r.ResponseCode != nil {
		line.Key = peppolResponseEvents[r.ResponseCode.Value]
	}
	for _, s := range r.Status {
		if s == nil || s.StatusReasonCode == nil {
			continue
		}
		listID := ""
		if s.StatusReasonCode.ListID != nil {
			listID = *s.StatusReasonCode.ListID
		}
		desc := ""
		if len(s.StatusReason) > 0 {
			desc = s.StatusReason[0]
		}
		switch listID {
		case peppolStatusReasonListID:
			if key := keyForCode(peppolStatusReasonCodes, s.StatusReasonCode.Value); key != "" {
				line.Reasons = append(line.Reasons, &bill.Reason{Key: key, Description: desc})
			}
		case peppolStatusActionListID:
			if key := keyForCode(peppolStatusActionCodes, s.StatusReasonCode.Value); key != "" {
				line.Actions = append(line.Actions, &bill.Action{Key: key, Description: desc})
			}
		}
	}

	// Map the mandatory DocumentTypeCode back to the referenced document type.
	if ref := dr.DocumentReference; ref != nil && line.Doc != nil &&
		ref.DocumentTypeCode != nil && ref.DocumentTypeCode.Value == documentTypeCodeCreditNote {
		line.Doc.Type = bill.InvoiceTypeCreditNote
	}
}

// keyForCode returns the GOBL key mapped to the given code in m, or empty if
// none matches.
func keyForCode(m map[cbc.Key]string, code string) cbc.Key {
	for k, v := range m {
		if v == code {
			return k
		}
	}
	return ""
}
