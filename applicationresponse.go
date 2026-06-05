package ubl

import (
	"encoding/xml"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
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

	SenderParty      *Party              `xml:"cac:SenderParty"`
	ReceiverParty    *Party              `xml:"cac:ReceiverParty"`
	DocumentResponse []*DocumentResponse `xml:"cac:DocumentResponse"`
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
// stamped by the matching context; the generic mapping leaves it unset.
type ResponseDocumentReference struct {
	ID               string  `xml:"cbc:ID"`
	UUID             string  `xml:"cbc:UUID,omitempty"`
	IssueDate        string  `xml:"cbc:IssueDate,omitempty"`
	DocumentTypeCode *IDType `xml:"cbc:DocumentTypeCode"`
}

// ublApplicationResponse builds the generic UBL 2.1 ApplicationResponse skeleton
// from a GOBL bill.Status.
func ublApplicationResponse(st *bill.Status, o *options) *ApplicationResponse {
	// SenderParty is who sends the response, ReceiverParty who receives it. The
	// base supplier/customer roles flip with the status type (a response travels
	// customer->supplier, an update supplier->customer, e.g. towards a tax agency
	// held as the recipient);
	sender, receiver := st.Customer, st.Supplier
	if st.Type == bill.StatusTypeUpdate {
		sender, receiver = st.Supplier, st.Customer
	}
	if st.Issuer != nil {
		sender = st.Issuer
	}
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
		ID:              invoiceNumber(st.Series, st.Code),
		IssueDate:       formatDate(st.IssueDate),
		SenderParty:     newParty(sender),
		ReceiverParty:   newParty(receiver),
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

	if o.context.Is(ContextPeppolInvoiceResponse) {
		// T111's data model omits these root elements and restricts the parties.
		out.UBLVersionID = ""
		out.UUID = ""
		trimToResponseParty(out.SenderParty)
		trimToResponseParty(out.ReceiverParty)
	}

	for _, line := range st.Lines {
		dr := &DocumentResponse{Response: &Response{}}
		// ReferenceID and Description are valid generic UBL but are not part of the
		// Peppol Invoice Response Response, so they are only emitted off-profile.
		if !o.context.Is(ContextPeppolInvoiceResponse) {
			if desc := responseDescription(line); desc != "" {
				dr.Response.Description = []string{desc}
			}
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

		if o.context.Is(ContextPeppolInvoiceResponse) {
			applyPeppolDocumentResponse(dr, line)
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

// Peppol BIS Invoice Response specifics follow.

// Peppol Invoice Response code-list identifiers for the StatusReasonCode.
const (
	peppolStatusReasonListID = "OPStatusReason"
	peppolStatusActionListID = "OPStatusAction"
)

// Peppol Invoice Response document-type codes (UNCL1001) for the referenced
// document; DocumentTypeCode is mandatory under T111.
const (
	documentTypeCodeListID     = "UNCL1001"
	documentTypeCodeInvoice    = "380"
	documentTypeCodeCreditNote = "381"
)

// peppolResponseCodes maps GOBL status events to the Peppol Invoice Response
// status codes (UNCL4343 subset, transaction T111). A technical "error" properly
// belongs in the Message Level Response (T71), but the Invoice Response has no
// technical-reject code, so it falls back to RE with the detail carried in the
// status clarification. "issued" is a pre-response state with no code.
var peppolResponseCodes = map[cbc.Key]string{
	bill.StatusEventAcknowledged: "AB",
	bill.StatusEventProcessing:   "IP",
	bill.StatusEventQuerying:     "UQ",
	bill.StatusEventRejected:     "RE",
	bill.StatusEventAccepted:     "AP",
	bill.StatusEventPaid:         "PD",
	bill.StatusEventError:        "RE",
}

// peppolResponseEvents reverses peppolResponseCodes for parsing. RE maps back to
// the business rejection (the error fallback is send-side only); CA
// (conditionally accepted) normalizes to accepted, with the conditions carried
// in the status line's reasons and actions.
var peppolResponseEvents = map[string]cbc.Key{
	"AB": bill.StatusEventAcknowledged,
	"IP": bill.StatusEventProcessing,
	"UQ": bill.StatusEventQuerying,
	"RE": bill.StatusEventRejected,
	"AP": bill.StatusEventAccepted,
	"PD": bill.StatusEventPaid,
	"CA": bill.StatusEventAccepted,
}

// peppolStatusReasonCodes maps GOBL reason keys to OPStatusReason codes.
var peppolStatusReasonCodes = map[cbc.Key]string{
	bill.ReasonKeyNone:            "NON",
	bill.ReasonKeyReferences:      "REF",
	bill.ReasonKeyLegal:           "LEG",
	bill.ReasonKeyUnknownReceiver: "REC",
	bill.ReasonKeyQuality:         "QUA",
	bill.ReasonKeyDelivery:        "DEL",
	bill.ReasonKeyPrices:          "PRI",
	bill.ReasonKeyQuantity:        "QTY",
	bill.ReasonKeyItems:           "ITM",
	bill.ReasonKeyPaymentTerms:    "PAY",
	bill.ReasonKeyNotRecognized:   "UNR",
	bill.ReasonKeyFinanceTerms:    "FIN",
	bill.ReasonKeyPartial:         "PPD",
	bill.ReasonKeyOther:           "OTH",
}

// peppolStatusActionCodes maps GOBL action keys to OPStatusAction codes.
var peppolStatusActionCodes = map[cbc.Key]string{
	bill.ActionKeyNone:          "NOA",
	bill.ActionKeyProvide:       "PIN",
	bill.ActionKeyReissue:       "NIN",
	bill.ActionKeyCreditFull:    "CNF",
	bill.ActionKeyCreditPartial: "CNP",
	bill.ActionKeyCreditAmount:  "CNA",
	bill.ActionKeyOther:         "OTH",
}

// applyPeppolDocumentResponse stamps the Peppol Invoice Response specifics onto
// a single DocumentResponse: the UNCL4343 response code, and one cac:Status per
// reason and action carrying the OPStatusReason / OPStatusAction clarification.
func applyPeppolDocumentResponse(dr *DocumentResponse, line *bill.StatusLine) {
	resp := dr.Response
	if code := peppolResponseCodes[line.Key]; code != "" {
		resp.ResponseCode = &IDType{Value: code}
	}
	for _, r := range line.Reasons {
		if r == nil {
			continue
		}
		if rc := peppolStatusReasonCodes[r.Key]; rc != "" {
			resp.Status = append(resp.Status, peppolStatus(peppolStatusReasonListID, rc, r.Description))
		}
	}
	for _, a := range line.Actions {
		if a == nil {
			continue
		}
		if ac := peppolStatusActionCodes[a.Key]; ac != "" {
			resp.Status = append(resp.Status, peppolStatus(peppolStatusActionListID, ac, a.Description))
		}
	}
	if dr.DocumentReference != nil {
		docType := documentTypeCodeInvoice
		if line.Doc != nil && line.Doc.Type == bill.InvoiceTypeCreditNote {
			docType = documentTypeCodeCreditNote
		}
		listID := documentTypeCodeListID
		dr.DocumentReference.DocumentTypeCode = &IDType{ListID: &listID, Value: docType}
	}
}

// trimToResponseParty reduces a party to the elements the Peppol Invoice
// Response (T111) permits on SenderParty/ReceiverParty: EndpointID,
// PartyIdentification, PartyLegalEntity and Contact. The CIUS forbids the rest.
func trimToResponseParty(p *Party) {
	if p == nil {
		return
	}
	p.PartyName = nil
	p.PostalAddress = nil
	p.PartyTaxScheme = nil
}

func peppolStatus(listID, code, reason string) *Status {
	s := &Status{StatusReasonCode: &IDType{ListID: &listID, Value: code}}
	if reason != "" {
		s.StatusReason = []string{reason}
	}
	return s
}
