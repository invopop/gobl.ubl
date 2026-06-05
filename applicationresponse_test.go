package ubl_test

import (
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleApplicationResponse = `<?xml version="1.0" encoding="UTF-8"?>
<ApplicationResponse xmlns="urn:oasis:names:specification:ubl:schema:xsd:ApplicationResponse-2" xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">
  <cbc:ID>RESP-1</cbc:ID>
  <cbc:IssueDate>2026-05-29</cbc:IssueDate>
  <cbc:Note>Processed automatically</cbc:Note>
  <cac:SenderParty><cac:PartyName><cbc:Name>Buyer Co</cbc:Name></cac:PartyName></cac:SenderParty>
  <cac:ReceiverParty><cac:PartyName><cbc:Name>Seller Co</cbc:Name></cac:PartyName></cac:ReceiverParty>
  <cac:DocumentResponse>
    <cac:Response>
      <cbc:ReferenceID>1</cbc:ReferenceID>
      <cbc:ResponseCode>BusinessAccept</cbc:ResponseCode>
      <cbc:Description>All good</cbc:Description>
      <cbc:EffectiveDate>2026-05-28</cbc:EffectiveDate>
    </cac:Response>
    <cac:DocumentReference>
      <cbc:ID>INV-42</cbc:ID>
      <cbc:IssueDate>2026-05-20</cbc:IssueDate>
    </cac:DocumentReference>
  </cac:DocumentResponse>
</ApplicationResponse>`

func TestParseApplicationResponse(t *testing.T) {
	doc, err := ubl.Parse([]byte(sampleApplicationResponse))
	require.NoError(t, err)

	ar, ok := doc.(*ubl.ApplicationResponse)
	require.True(t, ok, "parsed document should be an ApplicationResponse")

	env, err := ar.Convert()
	require.NoError(t, err)

	st, ok := env.Extract().(*bill.Status)
	require.True(t, ok, "converted document should be a bill.Status")

	assert.Equal(t, bill.StatusTypeResponse, st.Type)
	assert.Equal(t, "RESP-1", st.Code.String())

	// SenderParty maps to the customer (the responder), ReceiverParty to the
	// supplier (the originator).
	require.NotNil(t, st.Customer)
	assert.Equal(t, "Buyer Co", st.Customer.Name)
	require.NotNil(t, st.Supplier)
	assert.Equal(t, "Seller Co", st.Supplier.Name)

	require.Len(t, st.Notes, 1)
	assert.Equal(t, "Processed automatically", st.Notes[0].Text)

	require.Len(t, st.Lines, 1)
	assert.Equal(t, "All good", st.Lines[0].Description)
	require.NotNil(t, st.Lines[0].Date)
	assert.Equal(t, "2026-05-28", st.Lines[0].Date.String())
	require.NotNil(t, st.Lines[0].Doc)
	assert.Equal(t, "INV-42", st.Lines[0].Doc.Code.String())

	// The response code -> status event mapping is profile-specific and is not
	// applied by the generic mapping.
	assert.Empty(t, st.Lines[0].Key)
}

func TestConvertApplicationResponseSkeleton(t *testing.T) {
	effDate := cal.MakeDate(2026, 5, 28)
	st := &bill.Status{
		Type:      bill.StatusTypeResponse,
		Code:      "RESP-1",
		IssueDate: cal.MakeDate(2026, 5, 29),
		Supplier:  &org.Party{Name: "Seller Co"},
		Customer:  &org.Party{Name: "Buyer Co"},
		Notes:     []*org.Note{{Text: "Processed automatically"}},
		Lines: []*bill.StatusLine{
			{
				Index:       1,
				Key:         bill.StatusEventAccepted,
				Date:        &effDate,
				Description: "All good",
				Doc:         &org.DocumentRef{Code: "INV-42"},
			},
		},
	}
	env, err := gobl.Envelop(st)
	require.NoError(t, err)

	doc, err := ubl.Convert(env)
	require.NoError(t, err)

	ar, ok := doc.(*ubl.ApplicationResponse)
	require.True(t, ok, "converted document should be an ApplicationResponse")

	assert.Equal(t, "RESP-1", ar.ID)
	assert.Equal(t, ubl.Version, ar.UBLVersionID)
	assert.Equal(t, []string{"Processed automatically"}, ar.Note)

	// Customer maps to SenderParty, Supplier to ReceiverParty.
	require.NotNil(t, ar.SenderParty)
	require.NotNil(t, ar.SenderParty.PartyName)
	assert.Equal(t, "Buyer Co", ar.SenderParty.PartyName.Name)
	require.NotNil(t, ar.ReceiverParty)
	require.NotNil(t, ar.ReceiverParty.PartyName)
	assert.Equal(t, "Seller Co", ar.ReceiverParty.PartyName.Name)

	require.NotNil(t, ar.DocumentResponse)
	require.NotNil(t, ar.DocumentResponse.Response)
	assert.Equal(t, "1", ar.DocumentResponse.Response.ReferenceID)
	assert.Equal(t, []string{"All good"}, ar.DocumentResponse.Response.Description)
	assert.Equal(t, "2026-05-28", ar.DocumentResponse.Response.EffectiveDate)
	require.NotNil(t, ar.DocumentResponse.DocumentReference)
	assert.Equal(t, "INV-42", ar.DocumentResponse.DocumentReference.ID)

	// The response code and document-type code are profile-specific and are not
	// stamped by the generic conversion.
	assert.Nil(t, ar.DocumentResponse.Response.ResponseCode)
	assert.Nil(t, ar.DocumentResponse.DocumentReference.DocumentTypeCode)
}

func TestConvertPeppolInvoiceResponse(t *testing.T) {
	st := &bill.Status{
		Type:      bill.StatusTypeResponse,
		Code:      "RESP-9",
		IssueDate: cal.MakeDate(2026, 5, 29),
		Supplier:  &org.Party{Name: "Seller Co"},
		Customer:  &org.Party{Name: "Buyer Co"},
		Lines: []*bill.StatusLine{
			{
				Index: 1,
				Key:   bill.StatusEventRejected,
				Doc:   &org.DocumentRef{Code: "INV-9"},
				Reasons: []*bill.Reason{
					{Key: bill.ReasonKeyReferences, Description: "missing PO"},
				},
				Actions: []*bill.Action{
					{Key: bill.ActionKeyReissue, Description: "please reissue"},
				},
			},
		},
	}
	env, err := gobl.Envelop(st)
	require.NoError(t, err)

	doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextPeppolInvoiceResponse))
	require.NoError(t, err)
	ar, ok := doc.(*ubl.ApplicationResponse)
	require.True(t, ok)

	assert.Equal(t, "urn:fdc:peppol.eu:poacc:trns:invoice_response:3", ar.CustomizationID)
	require.NotNil(t, ar.ProfileID)
	assert.Equal(t, "urn:fdc:peppol.eu:poacc:bis:invoice_response:3", ar.ProfileID.Value)

	resp := ar.DocumentResponse.Response
	require.NotNil(t, resp.ResponseCode)
	assert.Equal(t, "RE", resp.ResponseCode.Value)

	require.Len(t, resp.Status, 2)
	require.NotNil(t, resp.Status[0].StatusReasonCode.ListID)
	assert.Equal(t, "OPStatusReason", *resp.Status[0].StatusReasonCode.ListID)
	assert.Equal(t, "REF", resp.Status[0].StatusReasonCode.Value)
	assert.Equal(t, []string{"missing PO"}, resp.Status[0].StatusReason)
	assert.Equal(t, "OPStatusAction", *resp.Status[1].StatusReasonCode.ListID)
	assert.Equal(t, "NIN", resp.Status[1].StatusReasonCode.Value)
	assert.Equal(t, []string{"please reissue"}, resp.Status[1].StatusReason)
}

func TestConvertPeppolInvoiceResponseErrorMapsToRejected(t *testing.T) {
	st := &bill.Status{
		Type:      bill.StatusTypeResponse,
		Code:      "RESP-E",
		IssueDate: cal.MakeDate(2026, 5, 29),
		Supplier:  &org.Party{Name: "Seller Co"},
		Customer:  &org.Party{Name: "Buyer Co"},
		Lines: []*bill.StatusLine{
			{
				Index:   1,
				Key:     bill.StatusEventError,
				Doc:     &org.DocumentRef{Code: "INV-E"},
				Reasons: []*bill.Reason{{Key: bill.ReasonKeyOther, Description: "system failure"}},
			},
		},
	}
	env, err := gobl.Envelop(st)
	require.NoError(t, err)

	doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextPeppolInvoiceResponse))
	require.NoError(t, err)
	ar, ok := doc.(*ubl.ApplicationResponse)
	require.True(t, ok)

	// A technical error has no Invoice Response code, so it falls back to RE.
	require.NotNil(t, ar.DocumentResponse.Response.ResponseCode)
	assert.Equal(t, "RE", ar.DocumentResponse.Response.ResponseCode.Value)
}

func TestConvertPeppolInvoiceResponseRequiresClarification(t *testing.T) {
	st := &bill.Status{
		Type:      bill.StatusTypeResponse,
		Code:      "RESP-R",
		IssueDate: cal.MakeDate(2026, 5, 29),
		Supplier:  &org.Party{Name: "Seller Co"},
		Customer:  &org.Party{Name: "Buyer Co"},
		Lines: []*bill.StatusLine{
			{Index: 1, Key: bill.StatusEventRejected, Doc: &org.DocumentRef{Code: "INV-R"}},
		},
	}
	env, err := gobl.Envelop(st)
	require.NoError(t, err)

	// Rejecting without any reason or action is invalid for Peppol.
	_, err = ubl.Convert(env, ubl.WithContext(ubl.ContextPeppolInvoiceResponse))
	require.Error(t, err)
}

func TestConvertPeppolInvoiceResponseRejectsUnmappedEvent(t *testing.T) {
	st := &bill.Status{
		Type:      bill.StatusTypeResponse,
		Code:      "RESP-I",
		IssueDate: cal.MakeDate(2026, 5, 29),
		Supplier:  &org.Party{Name: "Seller Co"},
		Customer:  &org.Party{Name: "Buyer Co"},
		Lines: []*bill.StatusLine{
			{Index: 1, Key: bill.StatusEventIssued, Doc: &org.DocumentRef{Code: "INV-I"}},
		},
	}
	env, err := gobl.Envelop(st)
	require.NoError(t, err)

	// "issued" has no Invoice Response code; rather than emit a document missing
	// the mandatory ResponseCode, conversion fails.
	_, err = ubl.Convert(env, ubl.WithContext(ubl.ContextPeppolInvoiceResponse))
	require.Error(t, err)
}

const samplePeppolInvoiceResponse = `<?xml version="1.0" encoding="UTF-8"?>
<ApplicationResponse xmlns="urn:oasis:names:specification:ubl:schema:xsd:ApplicationResponse-2" xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">
  <cbc:CustomizationID>urn:fdc:peppol.eu:poacc:trns:invoice_response:3</cbc:CustomizationID>
  <cbc:ProfileID>urn:fdc:peppol.eu:poacc:bis:invoice_response:3</cbc:ProfileID>
  <cbc:ID>RESP-9</cbc:ID>
  <cbc:IssueDate>2026-05-29</cbc:IssueDate>
  <cac:SenderParty><cac:PartyName><cbc:Name>Buyer Co</cbc:Name></cac:PartyName></cac:SenderParty>
  <cac:ReceiverParty><cac:PartyName><cbc:Name>Seller Co</cbc:Name></cac:PartyName></cac:ReceiverParty>
  <cac:DocumentResponse>
    <cac:Response>
      <cbc:ResponseCode>RE</cbc:ResponseCode>
      <cac:Status>
        <cbc:StatusReasonCode listID="OPStatusReason">REF</cbc:StatusReasonCode>
        <cbc:StatusReason>missing PO</cbc:StatusReason>
      </cac:Status>
      <cac:Status>
        <cbc:StatusReasonCode listID="OPStatusAction">NIN</cbc:StatusReasonCode>
        <cbc:StatusReason>please reissue</cbc:StatusReason>
      </cac:Status>
    </cac:Response>
    <cac:DocumentReference><cbc:ID>INV-9</cbc:ID></cac:DocumentReference>
  </cac:DocumentResponse>
</ApplicationResponse>`

const samplePeppolConditionallyAccepted = `<?xml version="1.0" encoding="UTF-8"?>
<ApplicationResponse xmlns="urn:oasis:names:specification:ubl:schema:xsd:ApplicationResponse-2" xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">
  <cbc:CustomizationID>urn:fdc:peppol.eu:poacc:trns:invoice_response:3</cbc:CustomizationID>
  <cbc:ProfileID>urn:fdc:peppol.eu:poacc:bis:invoice_response:3</cbc:ProfileID>
  <cbc:ID>RESP-CA</cbc:ID>
  <cbc:IssueDate>2026-05-29</cbc:IssueDate>
  <cac:SenderParty><cac:PartyName><cbc:Name>Buyer Co</cbc:Name></cac:PartyName></cac:SenderParty>
  <cac:ReceiverParty><cac:PartyName><cbc:Name>Seller Co</cbc:Name></cac:PartyName></cac:ReceiverParty>
  <cac:DocumentResponse>
    <cac:Response>
      <cbc:ResponseCode>CA</cbc:ResponseCode>
      <cac:Status>
        <cbc:StatusReasonCode listID="OPStatusReason">PRI</cbc:StatusReasonCode>
        <cbc:StatusReason>price to be confirmed</cbc:StatusReason>
      </cac:Status>
    </cac:Response>
    <cac:DocumentReference><cbc:ID>INV-CA</cbc:ID></cac:DocumentReference>
  </cac:DocumentResponse>
</ApplicationResponse>`

func TestParsePeppolConditionallyAccepted(t *testing.T) {
	doc, err := ubl.Parse([]byte(samplePeppolConditionallyAccepted))
	require.NoError(t, err)
	ar, ok := doc.(*ubl.ApplicationResponse)
	require.True(t, ok)

	env, err := ar.Convert()
	require.NoError(t, err)
	st, ok := env.Extract().(*bill.Status)
	require.True(t, ok)

	require.Len(t, st.Lines, 1)
	// CA normalizes to accepted, carrying the conditions as reasons.
	assert.Equal(t, bill.StatusEventAccepted, st.Lines[0].Key)
	require.Len(t, st.Lines[0].Reasons, 1)
	assert.Equal(t, bill.ReasonKeyPrices, st.Lines[0].Reasons[0].Key)
	assert.Equal(t, "price to be confirmed", st.Lines[0].Reasons[0].Description)
}

func TestParsePeppolInvoiceResponse(t *testing.T) {
	doc, err := ubl.Parse([]byte(samplePeppolInvoiceResponse))
	require.NoError(t, err)
	ar, ok := doc.(*ubl.ApplicationResponse)
	require.True(t, ok)

	env, err := ar.Convert()
	require.NoError(t, err)
	st, ok := env.Extract().(*bill.Status)
	require.True(t, ok)

	require.Len(t, st.Lines, 1)
	assert.Equal(t, bill.StatusEventRejected, st.Lines[0].Key)

	require.Len(t, st.Lines[0].Reasons, 1)
	assert.Equal(t, bill.ReasonKeyReferences, st.Lines[0].Reasons[0].Key)
	assert.Equal(t, "missing PO", st.Lines[0].Reasons[0].Description)

	require.Len(t, st.Lines[0].Actions, 1)
	assert.Equal(t, bill.ActionKeyReissue, st.Lines[0].Actions[0].Key)
	assert.Equal(t, "please reissue", st.Lines[0].Actions[0].Description)
}
