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

// goblResponseEvents reverses oioublResponseCodes, mapping an OIOUBL
// responsecode-1.1 value back to a GOBL status event.
var goblResponseEvents = map[string]cbc.Key{
	responseCodeBusinessAccept:  bill.StatusEventAccepted,
	responseCodeBusinessReject:  bill.StatusEventRejected,
	responseCodeTechnicalAccept: bill.StatusEventAcknowledged,
	responseCodeTechnicalReject: bill.StatusEventError,
	responseCodeProfileReject:   bill.StatusEventError,
}

// Convert turns a parsed UBL ApplicationResponse into a GOBL envelope wrapping a
// bill.Status.
func (ar *ApplicationResponse) Convert() (*gobl.Envelope, error) {
	o := new(options)
	if ar.CustomizationID == ContextOIOUBL21.CustomizationID {
		o.context = ContextOIOUBL21
	} else if ctx := FindContext(ar.CustomizationID, profileIDValue(ar.ProfileID)); ctx != nil {
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

	line, err := ar.goblStatusLine()
	if err != nil {
		return nil, err
	}
	out.Lines = []*bill.StatusLine{line}

	return out, nil
}

func (ar *ApplicationResponse) goblStatusLine() (*bill.StatusLine, error) {
	line := new(bill.StatusLine)
	dr := ar.DocumentResponse
	if dr == nil {
		return line, nil
	}

	if r := dr.Response; r != nil {
		if r.ResponseCode != nil {
			line.Key = goblResponseEvents[r.ResponseCode.Value]
		}
		if len(r.Description) > 0 {
			line.Description = r.Description[0]
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
		if ref.DocumentTypeCode != nil && ref.DocumentTypeCode.Value == responseDocTypeCreditNote {
			doc.Type = bill.InvoiceTypeCreditNote
		}
		line.Doc = doc
	}

	return line, nil
}
