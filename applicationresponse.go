package ubl

import (
	"encoding/xml"

	"github.com/invopop/gobl/bill"
)

// NamespaceUBLApplicationResponse is the UBL 2.1 ApplicationResponse root namespace.
const NamespaceUBLApplicationResponse = "urn:oasis:names:specification:ubl:schema:xsd:ApplicationResponse-2"

// ApplicationResponse represents a UBL 2.1 ApplicationResponse document, used to
// return a response (accept or reject) for a previously received document.
type ApplicationResponse struct {
	XMLName      xml.Name
	CACNamespace string `xml:"xmlns:cac,attr"`
	CBCNamespace string `xml:"xmlns:cbc,attr"`
	UBLNamespace string `xml:"xmlns,attr"`

	UBLVersionID    string  `xml:"cbc:UBLVersionID,omitempty"`
	CustomizationID string  `xml:"cbc:CustomizationID,omitempty"`
	ProfileID       *IDType `xml:"cbc:ProfileID,omitempty"`
	ID              string  `xml:"cbc:ID"`
	UUID            string  `xml:"cbc:UUID,omitempty"`
	IssueDate       string  `xml:"cbc:IssueDate"`
	IssueTime       string  `xml:"cbc:IssueTime,omitempty"`

	Note []string `xml:"cbc:Note,omitempty"`

	SenderParty      *Party              `xml:"cac:SenderParty"`
	ReceiverParty    *Party              `xml:"cac:ReceiverParty"`
	DocumentResponse []*DocumentResponse `xml:"cac:DocumentResponse"`
}

// DocumentResponse pairs one Response with the document it concerns. An
// ApplicationResponse may carry several (generic UBL allows one per status
// line); profiles such as the Peppol Invoice Response restrict it to one.
type DocumentResponse struct {
	Response          *Response                  `xml:"cac:Response"`
	DocumentReference *ResponseDocumentReference `xml:"cac:DocumentReference"`
}

// Response carries the response code and an optional human description. The
// ResponseCode value and its code-list attributes are profile-specific and are
// stamped by the matching format.
type Response struct {
	ReferenceID   string    `xml:"cbc:ReferenceID,omitempty"`
	ResponseCode  *IDType   `xml:"cbc:ResponseCode"`
	Description   []string  `xml:"cbc:Description,omitempty"`
	EffectiveDate string    `xml:"cbc:EffectiveDate,omitempty"`
	Status        []*Status `xml:"cac:Status,omitempty"`
}

// Status carries a coded clarification within a Response. The code list it draws
// from is identified by the StatusReasonCode listID and is profile-specific.
type Status struct {
	StatusReasonCode *IDType  `xml:"cbc:StatusReasonCode,omitempty"`
	StatusReason     []string `xml:"cbc:StatusReason,omitempty"`
}

// ResponseDocumentReference identifies the document being responded to. The
// DocumentTypeCode is profile-specific (drawn from a profile's code list) and is
// stamped by the matching format; the generic mapping leaves it unset.
type ResponseDocumentReference struct {
	ID               string  `xml:"cbc:ID"`
	UUID             string  `xml:"cbc:UUID,omitempty"`
	IssueDate        string  `xml:"cbc:IssueDate,omitempty"`
	DocumentTypeCode *IDType `xml:"cbc:DocumentTypeCode"`
}

func ublApplicationResponse(st *bill.Status, o *options) *ApplicationResponse {
	// SenderParty is who sends the response, ReceiverParty who receives it: a
	// response travels customer->supplier, an update supplier->customer.
	sender, receiver := st.Customer, st.Supplier
	if st.Type == bill.StatusTypeUpdate {
		sender, receiver = st.Supplier, st.Customer
	}

	out := &ApplicationResponse{
		XMLName:         xml.Name{Local: "ApplicationResponse"},
		CACNamespace:    NamespaceCAC,
		CBCNamespace:    NamespaceCBC,
		UBLNamespace:    NamespaceUBLApplicationResponse,
		UBLVersionID:    Version,
		CustomizationID: o.spec.CustomizationID,
		ID:              invoiceNumber(st.Series, st.Code),
		IssueDate:       formatDate(st.IssueDate),
		SenderParty:     newParty(sender),
		ReceiverParty:   newParty(receiver),
	}
	if o.spec.ProfileID != "" {
		out.ProfileID = &IDType{Value: o.spec.ProfileID}
	}
	if !st.UUID.IsZero() {
		out.UUID = st.UUID.String()
	}
	if st.IssueTime != nil {
		out.IssueTime = st.IssueTime.String()
	}
	for _, n := range st.Notes {
		if n != nil && n.Text != "" {
			out.Note = append(out.Note, n.Text)
		}
	}

	for _, line := range st.Lines {
		dr := &DocumentResponse{Response: &Response{}}
		if desc := responseDescription(line); desc != "" {
			dr.Response.Description = []string{desc}
		}
		if line.Date != nil {
			dr.Response.EffectiveDate = formatDate(*line.Date)
		}
		if line.Doc != nil {
			ref := &ResponseDocumentReference{
				ID: invoiceNumber(line.Doc.Series, line.Doc.Code),
			}
			if !line.Doc.UUID.IsZero() {
				ref.UUID = line.Doc.UUID.String()
			}
			if line.Doc.IssueDate != nil {
				ref.IssueDate = formatDate(*line.Doc.IssueDate)
			}
			dr.DocumentReference = ref
		}

		out.DocumentResponse = append(out.DocumentResponse, dr)
	}

	return out
}

// responseDescription prefers the line description and falls back to the first
// reason's description.
func responseDescription(line *bill.StatusLine) string {
	if line.Description != "" {
		return line.Description
	}
	for _, r := range line.Reasons {
		if r != nil && r.Description != "" {
			return r.Description
		}
	}
	return ""
}
