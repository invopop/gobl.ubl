package ubl

import (
	"encoding/xml"
	"strconv"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
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

// DocumentResponse pairs one Response with the document it concerns; an
// ApplicationResponse carries one per status line.
type DocumentResponse struct {
	Response          *Response                  `xml:"cac:Response"`
	DocumentReference *ResponseDocumentReference `xml:"cac:DocumentReference"`
}

// Response carries the response code and an optional human description. The
// ResponseCode value and its code-list attributes are profile-specific and are
// stamped by the matching context.
type Response struct {
	ReferenceID   string   `xml:"cbc:ReferenceID,omitempty"`
	ResponseCode  *IDType  `xml:"cbc:ResponseCode"`
	Description   []string `xml:"cbc:Description,omitempty"`
	EffectiveDate string   `xml:"cbc:EffectiveDate,omitempty"`
}

// ResponseDocumentReference identifies the document being responded to. The
// DocumentTypeCode is profile-specific (drawn from a profile's code list) and is
// stamped by the matching context; the generic mapping leaves it unset.
type ResponseDocumentReference struct {
	ID               string  `xml:"cbc:ID"`
	UUID             string  `xml:"cbc:UUID,omitempty"`
	IssueDate        string  `xml:"cbc:IssueDate,omitempty"`
	DocumentTypeCode *IDType `xml:"cbc:DocumentTypeCode"`
}

func ublApplicationResponse(st *bill.Status, o *options) *ApplicationResponse {
	// SenderParty is who sends the response, ReceiverParty who receives it. The
	// supplier/customer roles flip with the status type (a response travels
	// customer->supplier, an update supplier->customer).
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
		CustomizationID: o.context.CustomizationID,
		ID:              invoiceNumber(st.Series, st.Code),
		IssueDate:       formatDate(st.IssueDate),
		SenderParty:     newParty(sender, o.context),
		ReceiverParty:   newParty(receiver, o.context),
	}
	if o.context.ProfileID != "" {
		out.ProfileID = &IDType{Value: o.context.ProfileID}
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

	if o.context.Is(ContextOIOUBL21) {
		applyOIOUBL21Party(out.SenderParty)
		applyOIOUBL21Party(out.ReceiverParty)
		applyOIOUBL21ResponseProfile(out, st)
	}

	// One DocumentResponse per status line: its response (description, effective
	// date) plus a reference to the document being responded to.
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

		if o.context.Is(ContextOIOUBL21) {
			applyOIOUBL21DocumentResponse(dr, line)
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

// OIOUBL 2.1 ApplicationResponse specifics follow.

// oioublProfileTechnicalID is the technical-response profile the schematron
// couples with the TechnicalAccept response code (F-APR057/058). Both
// ApplicationResponse profiles only appear in the profileid-1.4 code list
// (oioublSchemeProfileV14), unlike invoices, which use 1.2 (see invoice.go).
const oioublProfileTechnicalID = "Procurement-TecRes-1.0"

// applyOIOUBL21ResponseProfile stamps the OIOUBL profileid-1.4 code-list
// attributes onto the ProfileID and, for a technical acknowledgement, swaps in
// the technical-response profile. F-APR057/F-APR058 bind the TechnicalAccept
// response code to that profile; every other response rides the billing profile.
func applyOIOUBL21ResponseProfile(out *ApplicationResponse, st *bill.Status) {
	if out.ProfileID == nil {
		return
	}
	out.ProfileID.SchemeAgencyID = ptr(oioublAgencyID)
	out.ProfileID.SchemeID = ptr(oioublSchemeProfileV14)
	if len(st.Lines) > 0 && st.Lines[0].Key == bill.StatusLineAcknowledged {
		out.ProfileID.Value = oioublProfileTechnicalID
	}
}

// OIOUBL responsecode-1.1 wire values. Derived from the status event (see
// oioubl21ResponseCode), not carried in an extension. F-APR018 accepts five of
// the six codelist values (ProfileAccept is rejected).
const (
	oioubl21ResponseCodeBusinessAccept  cbc.Code = "BusinessAccept"
	oioubl21ResponseCodeBusinessReject  cbc.Code = "BusinessReject"
	oioubl21ResponseCodeTechnicalAccept cbc.Code = "TechnicalAccept"
	oioubl21ResponseCodeTechnicalReject cbc.Code = "TechnicalReject"
	oioubl21ResponseCodeProfileReject   cbc.Code = "ProfileReject"
)

// oioubl21ResponseCode maps a GOBL status event to its OIOUBL responsecode-1.1
// value, or "" for events OIOUBL does not represent (rejected by F-APR018). The
// code is derived from the event rather than carried in an extension.
func oioubl21ResponseCode(key cbc.Key) string {
	switch key {
	case bill.StatusLineAccepted:
		return string(oioubl21ResponseCodeBusinessAccept)
	case bill.StatusLineRejected:
		return string(oioubl21ResponseCodeBusinessReject)
	case bill.StatusLineAcknowledged:
		return string(oioubl21ResponseCodeTechnicalAccept)
	case bill.StatusLineError:
		return string(oioubl21ResponseCodeTechnicalReject)
	}
	return ""
}

// applyOIOUBL21DocumentResponse stamps the OIOUBL 2.1 specifics onto a single
// DocumentResponse: the mandatory ReferenceID (F-APR016), the responsecode-1.1
// value with its code-list attributes, and the document-type code list.
func applyOIOUBL21DocumentResponse(dr *DocumentResponse, line *bill.StatusLine) {
	resp := dr.Response
	resp.ReferenceID = strconv.Itoa(responseReferenceID(line.Index))

	if code := oioubl21ResponseCode(line.Key); code != "" {
		resp.ResponseCode = &IDType{
			ListAgencyID: ptr(oioublAgencyID),
			ListID:       ptr(oioublListResponseCode),
			Value:        code,
		}
	}

	if ref := dr.DocumentReference; ref != nil && line.Doc != nil {
		ref.DocumentTypeCode = &IDType{
			ListAgencyID: ptr(oioublAgencyID),
			ListID:       ptr(oioublListResponseDocType),
			Value:        oioublResponseDocType(line.Doc.Type),
		}
	}
}

// responseReferenceID returns the 1-based line reference for the Response,
// falling back to 1 for an unset index (F-APR016 requires a non-empty value).
func responseReferenceID(index int) int {
	if index < 1 {
		return 1
	}
	return index
}

// oioublResponseDocType maps a referenced GOBL document type to the OIOUBL
// responsedocumenttypecode-1.1 value.
func oioublResponseDocType(t cbc.Key) string {
	if t == bill.InvoiceTypeCreditNote {
		return "CreditNote"
	}
	return "Invoice"
}
