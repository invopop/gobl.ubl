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
	if ctx := FindContext(ar.CustomizationID, profileID); ctx != nil {
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
		Supplier: goblParty(ar.ReceiverParty),
		Customer: goblParty(ar.SenderParty),
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
// response code and the status clarifications are profile-specific and are
// mapped back by the matching context.
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

	return line, nil
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
			line.Reasons = append(line.Reasons, &bill.Reason{
				Key:         keyForCode(peppolStatusReasonCodes, s.StatusReasonCode.Value),
				Description: desc,
			})
		case peppolStatusActionListID:
			line.Actions = append(line.Actions, &bill.Action{
				Key:         keyForCode(peppolStatusActionCodes, s.StatusReasonCode.Value),
				Description: desc,
			})
		}
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
