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
