package ubl

import (
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
)

// NamespaceUBLApplicationResponse is the UBL 2.1 ApplicationResponse root namespace.
const NamespaceUBLApplicationResponse = "urn:oasis:names:specification:ubl:schema:xsd:ApplicationResponse-2"

// OIOUBL ApplicationResponse code list identifiers and the technical-response
// profile that the schematron couples with the TechnicalAccept response code.
const (
	responseCodeListID       = "urn:oioubl:codelist:responsecode-1.1"
	responseDocTypeListID    = "urn:oioubl:codelist:responsedocumenttypecode-1.1"
	oioublProfileSchemeID    = "urn:oioubl:id:profileid-1.4"
	oioublProfileTechnicalID = "Procurement-TecRes-1.0"
	oioublCodeListAgencyID   = "320"
)

// OIOUBL responsecode-1.1 values accepted by the ApplicationResponse schematron
// (F-APR018).
const (
	responseCodeBusinessAccept  = "BusinessAccept"
	responseCodeBusinessReject  = "BusinessReject"
	responseCodeTechnicalAccept = "TechnicalAccept"
	responseCodeTechnicalReject = "TechnicalReject"
	responseCodeProfileReject   = "ProfileReject"
)

// OIOUBL responsedocumenttypecode-1.1 values for the referenced document.
const (
	responseDocTypeInvoice    = "Invoice"
	responseDocTypeCreditNote = "CreditNote"
)

// oioublResponseCodes maps GOBL status events to the OIOUBL responsecode-1.1
// values accepted by the ApplicationResponse schematron (F-APR018).
var oioublResponseCodes = map[cbc.Key]string{
	bill.StatusEventAccepted:     responseCodeBusinessAccept,
	bill.StatusEventRejected:     responseCodeBusinessReject,
	bill.StatusEventAcknowledged: responseCodeTechnicalAccept,
	bill.StatusEventError:        responseCodeTechnicalReject,
}

// ApplicationResponse represents a UBL 2.1 ApplicationResponse document. On the
// Danish NemHandel network it is used to return a business response (accept or
// reject) for a previously received document such as an invoice.
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

	SenderParty      *Party            `xml:"cac:SenderParty"`
	ReceiverParty    *Party            `xml:"cac:ReceiverParty"`
	DocumentResponse *DocumentResponse `xml:"cac:DocumentResponse"`
}

// DocumentResponse couples a response with the document it concerns. OIOUBL
// allows at most one of each (F-APR051/F-APR054).
type DocumentResponse struct {
	Response          *Response                  `xml:"cac:Response"`
	DocumentReference *ResponseDocumentReference `xml:"cac:DocumentReference"`
}

// Response carries the response code and an optional human description.
type Response struct {
	ReferenceID  string   `xml:"cbc:ReferenceID"`
	ResponseCode *IDType  `xml:"cbc:ResponseCode"`
	Description  []string `xml:"cbc:Description,omitempty"`
}

// ResponseDocumentReference identifies the document being responded to.
type ResponseDocumentReference struct {
	ID               string  `xml:"cbc:ID"`
	UUID             string  `xml:"cbc:UUID,omitempty"`
	IssueDate        string  `xml:"cbc:IssueDate,omitempty"`
	DocumentTypeCode *IDType `xml:"cbc:DocumentTypeCode"`
}

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
		SenderParty:     newParty(sender, o.context),
		ReceiverParty:   newParty(receiver, o.context),
	}
	if !st.UUID.IsZero() {
		out.UUID = st.UUID.String()
	}
	if st.IssueTime != nil {
		out.IssueTime = st.IssueTime.String()
	}

	out.DocumentResponse = &DocumentResponse{
		Response: &Response{
			ReferenceID: strconv.Itoa(responseReferenceID(line.Index)),
		},
	}
	if desc := responseDescription(line); desc != "" {
		out.DocumentResponse.Response.Description = []string{desc}
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

	switch {
	case o.context.Is(ContextOIOUBL21):
		if err := applyOIOUBL21ApplicationResponse(out, line); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%w: ApplicationResponse", ErrUnsupportedDocumentType)
	}

	return out, nil
}

// applyOIOUBL21ApplicationResponse stamps the OIOUBL 2.1 specifics onto a
// generic UBL ApplicationResponse: the responsecode-1.1 value and its code-list
// attributes, the document-type code list, the technical-response profile
// coupling, and the Danish party formatting.
func applyOIOUBL21ApplicationResponse(out *ApplicationResponse, line *bill.StatusLine) error {
	code, ok := oioublResponseCodes[line.Key]
	if !ok {
		return fmt.Errorf("OIOUBL ApplicationResponse does not support status event %q", line.Key)
	}

	agencyID := oioublCodeListAgencyID
	schemeID := oioublProfileSchemeID
	out.ProfileID.SchemeAgencyID = &agencyID
	out.ProfileID.SchemeID = &schemeID
	// F-APR057/F-APR058 bind the TechnicalAccept response code to the
	// technical-response profile; everything else rides the billing profile.
	if code == responseCodeTechnicalAccept {
		out.ProfileID.Value = oioublProfileTechnicalID
	}

	applyOIOUBL21Party(out.SenderParty)
	applyOIOUBL21Party(out.ReceiverParty)

	codeListID := responseCodeListID
	out.DocumentResponse.Response.ResponseCode = &IDType{
		ListAgencyID: &agencyID,
		ListID:       &codeListID,
		Value:        code,
	}

	if ref := out.DocumentResponse.DocumentReference; ref != nil {
		docTypeListID := responseDocTypeListID
		ref.DocumentTypeCode = &IDType{
			ListAgencyID: &agencyID,
			ListID:       &docTypeListID,
			Value:        oioublResponseDocType(line.Doc.Type),
		}
	}

	return nil
}

// responseReferenceID returns a 1-based reference for the Response, as the
// schematron requires a non-empty ReferenceID (F-APR016).
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

// oioublResponseDocType maps a referenced GOBL document type to the OIOUBL
// responsedocumenttypecode-1.1 value.
func oioublResponseDocType(t cbc.Key) string {
	if t == bill.InvoiceTypeCreditNote {
		return responseDocTypeCreditNote
	}
	return responseDocTypeInvoice
}
