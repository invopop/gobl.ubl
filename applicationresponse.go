package ubl

import (
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/invopop/gobl/bill"
)

// NamespaceUBLApplicationResponse is the UBL 2.1 ApplicationResponse root namespace.
const NamespaceUBLApplicationResponse = "urn:oasis:names:specification:ubl:schema:xsd:ApplicationResponse-2"

// ApplicationResponse represents a UBL 2.1 ApplicationResponse document, used to
// return a response (accept or reject) for a previously received document such
// as an invoice.
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

	SenderParty      *Party            `xml:"cac:SenderParty"`
	ReceiverParty    *Party            `xml:"cac:ReceiverParty"`
	DocumentResponse *DocumentResponse `xml:"cac:DocumentResponse"`
}

// DocumentResponse couples a response with the document it concerns. It is
// modelled as a single response, which is all the supported profiles need.
type DocumentResponse struct {
	Response          *Response                  `xml:"cac:Response"`
	DocumentReference *ResponseDocumentReference `xml:"cac:DocumentReference"`
}

// Response carries the response code and an optional human description. The
// ResponseCode value and its code-list attributes are profile-specific and are
// stamped by the matching context.
type Response struct {
	ReferenceID   string   `xml:"cbc:ReferenceID"`
	ResponseCode  *IDType  `xml:"cbc:ResponseCode"`
	Description   []string `xml:"cbc:Description,omitempty"`
	EffectiveDate string   `xml:"cbc:EffectiveDate,omitempty"`
}

// ResponseDocumentReference identifies the document being responded to. The
// DocumentTypeCode is profile-specific and is stamped by the matching context.
type ResponseDocumentReference struct {
	ID               string  `xml:"cbc:ID"`
	UUID             string  `xml:"cbc:UUID,omitempty"`
	IssueDate        string  `xml:"cbc:IssueDate,omitempty"`
	DocumentTypeCode *IDType `xml:"cbc:DocumentTypeCode"`
}

// ublApplicationResponse builds the generic UBL 2.1 ApplicationResponse skeleton
// from a GOBL bill.Status. The profile-specific values (the response code and
// its code-list attributes, the document-type code, profile identifiers and any
// regional party formatting) are stamped afterwards by the matching context.
func ublApplicationResponse(st *bill.Status, o *options) (*ApplicationResponse, error) {
	if len(st.Lines) != 1 {
		return nil, fmt.Errorf("ApplicationResponse requires a single document response, got %d", len(st.Lines))
	}
	line := st.Lines[0]

	// The response travels from the responder (customer, or an intermediary
	// issuer) to the originating party (supplier, or its recipient).
	sender := st.Customer
	if st.Issuer != nil {
		sender = st.Issuer
	}
	receiver := st.Supplier
	if st.Recipient != nil {
		receiver = st.Recipient
	}

	out := &ApplicationResponse{
		XMLName:         xml.Name{Local: "ApplicationResponse"},
		CACNamespace:    NamespaceCAC,
		CBCNamespace:    NamespaceCBC,
		UBLNamespace:    NamespaceUBLApplicationResponse,
		UBLVersionID:    Version,
		CustomizationID: o.context.CustomizationID,
		ProfileID:       &IDType{Value: o.context.ProfileID},
		ID:              invoiceNumber(st.Series, st.Code),
		IssueDate:       formatDate(st.IssueDate),
		SenderParty:     newParty(sender),
		ReceiverParty:   newParty(receiver),
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

	out.DocumentResponse = &DocumentResponse{
		Response: &Response{
			ReferenceID: strconv.Itoa(responseReferenceID(line.Index)),
		},
	}
	if desc := responseDescription(line); desc != "" {
		out.DocumentResponse.Response.Description = []string{desc}
	}
	if line.Date != nil {
		out.DocumentResponse.Response.EffectiveDate = formatDate(*line.Date)
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
		out.DocumentResponse.DocumentReference = ref
	}

	return out, nil
}

// responseReferenceID returns a 1-based reference for the Response, as the
// UBL ApplicationResponse requires a non-empty ReferenceID.
func responseReferenceID(index int) int {
	if index < 1 {
		return 1
	}
	return index
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
