package ubl_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/org"
	"github.com/invopop/phive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const oioublResponseDir = "oioubl21-response"

func TestConvertToApplicationResponse(t *testing.T) {
	var pc phive.ValidationServiceClient
	if *validate {
		conn, err := grpc.NewClient(
			"127.0.0.1:9090",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer conn.Close() //nolint:errcheck
		pc = phive.NewValidationServiceClient(conn)
	}

	examples, err := filepath.Glob(filepath.Join(getConvertPath(), oioublResponseDir, jsonPattern))
	require.NoError(t, err)
	require.NotEmpty(t, examples, "no ApplicationResponse examples found")

	for _, example := range examples {
		inName := filepath.Base(example)
		outName := strings.Replace(inName, ".json", ".xml", 1)

		t.Run(inName, func(t *testing.T) {
			env, err := loadTestEnvelopeFromPath(example)
			require.NoError(t, err)

			doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextOIOUBL21))
			require.NoError(t, err)

			data, err := ubl.Bytes(doc)
			require.NoError(t, err)

			outPath := filepath.Join(getConvertPath(), oioublResponseDir, "out", outName)
			if *updateOut {
				require.NoError(t, os.WriteFile(outPath, data, 0644))
			}

			if *validate {
				vesid := ubl.ContextOIOUBL21.VESIDs.ApplicationResponse
				resp, err := pc.ValidateXml(context.Background(), &phive.ValidateXmlRequest{
					Vesid:      vesid,
					XmlContent: data,
				})
				require.NoError(t, err)
				results, err := json.MarshalIndent(resp.Results, "", "  ")
				require.NoError(t, err)
				require.True(t, resp.Success, "Generated XML should be valid for %s: %s", vesid, string(results))
			}

			output, err := os.ReadFile(outPath)
			assert.NoError(t, err)
			assert.Equal(t, string(output), string(data), "Output should match the expected XML. Update with --update flag.")
		})
	}
}

func TestParseOIOUBL21ApplicationResponse(t *testing.T) {
	examples, err := filepath.Glob(filepath.Join(getParsePath(), oioublResponseDir, xmlPattern))
	require.NoError(t, err)
	require.NotEmpty(t, examples, "no ApplicationResponse parse examples found")

	for _, example := range examples {
		inName := filepath.Base(example)
		outName := strings.Replace(inName, ".xml", ".json", 1)

		t.Run(inName, func(t *testing.T) {
			xmlData, err := os.ReadFile(example)
			require.NoError(t, err)

			doc, err := ubl.Parse(xmlData)
			require.NoError(t, err)
			ar, ok := doc.(*ubl.ApplicationResponse)
			require.True(t, ok, "Document should be an ApplicationResponse")

			env, err := ar.Convert()
			require.NoError(t, err)

			env.Head.UUID = staticUUID
			if st, ok := env.Extract().(*bill.Status); ok {
				st.UUID = staticUUID
			}
			require.NoError(t, env.Calculate())

			outPath := filepath.Join(getParsePath(), oioublResponseDir, "out", outName)
			if *updateOut {
				data, err := json.MarshalIndent(env, "", "\t")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(outPath, data, 0644))
			}

			status, ok := env.Extract().(*bill.Status)
			require.True(t, ok, "Document should be a status")
			data, err := json.MarshalIndent(status, "", "\t")
			require.NoError(t, err)

			output, err := os.ReadFile(outPath)
			assert.NoError(t, err)

			var expectedEnv gobl.Envelope
			require.NoError(t, json.Unmarshal(output, &expectedEnv))
			expectedStatus, ok := expectedEnv.Extract().(*bill.Status)
			require.True(t, ok, "Expected document should be a status")
			expectedData, err := json.MarshalIndent(expectedStatus, "", "\t")
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedData), string(data), "Status should match the expected JSON. Update with --update flag.")
		})
	}
}

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
				Key:         bill.StatusLineAccepted,
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
	// EN16931 has no ProfileID, so the element is omitted rather than emitted empty.
	assert.Nil(t, ar.ProfileID)

	// Customer maps to SenderParty, Supplier to ReceiverParty.
	require.NotNil(t, ar.SenderParty)
	require.NotNil(t, ar.SenderParty.PartyName)
	assert.Equal(t, "Buyer Co", ar.SenderParty.PartyName.Name)
	require.NotNil(t, ar.ReceiverParty)
	require.NotNil(t, ar.ReceiverParty.PartyName)
	assert.Equal(t, "Seller Co", ar.ReceiverParty.PartyName.Name)

	require.Len(t, ar.DocumentResponse, 1)
	dr := ar.DocumentResponse[0]
	require.NotNil(t, dr.Response)
	assert.Empty(t, dr.Response.ReferenceID)
	assert.Equal(t, []string{"All good"}, dr.Response.Description)
	assert.Equal(t, "2026-05-28", dr.Response.EffectiveDate)
	require.NotNil(t, dr.DocumentReference)
	assert.Equal(t, "INV-42", dr.DocumentReference.ID)

	// The response code and document-type code are profile-specific and are not
	// stamped by the generic conversion.
	assert.Nil(t, dr.Response.ResponseCode)
	assert.Nil(t, dr.DocumentReference.DocumentTypeCode)
}

func TestConvertApplicationResponseFansOutLines(t *testing.T) {
	st := &bill.Status{
		Type:      bill.StatusTypeResponse,
		Code:      "RESP-MULTI",
		IssueDate: cal.MakeDate(2026, 5, 29),
		Supplier:  &org.Party{Name: "Seller Co"},
		Customer:  &org.Party{Name: "Buyer Co"},
		Lines: []*bill.StatusLine{
			{Index: 1, Doc: &org.DocumentRef{Code: "INV-1"}},
			{Index: 2, Doc: &org.DocumentRef{Code: "INV-2"}},
		},
	}
	env, err := gobl.Envelop(st)
	require.NoError(t, err)

	// Generic UBL fans every line into its own DocumentResponse in one response.
	doc, err := ubl.Convert(env)
	require.NoError(t, err)
	ar, ok := doc.(*ubl.ApplicationResponse)
	require.True(t, ok)

	require.Len(t, ar.DocumentResponse, 2)
	assert.Equal(t, "INV-1", ar.DocumentResponse[0].DocumentReference.ID)
	assert.Equal(t, "INV-2", ar.DocumentResponse[1].DocumentReference.ID)
}

func TestConvertApplicationResponseUpdateFlipsDirection(t *testing.T) {
	st := &bill.Status{
		Type:      bill.StatusTypeUpdate,
		Code:      "UPD-1",
		IssueDate: cal.MakeDate(2026, 5, 29),
		Supplier:  &org.Party{Name: "Seller Co"},
		Customer:  &org.Party{Name: "Buyer Co"},
		Lines: []*bill.StatusLine{
			{Index: 1, Key: bill.StatusLinePaid, Doc: &org.DocumentRef{Code: "INV-1"}},
		},
	}
	env, err := gobl.Envelop(st)
	require.NoError(t, err)

	doc, err := ubl.Convert(env)
	require.NoError(t, err)
	ar, ok := doc.(*ubl.ApplicationResponse)
	require.True(t, ok)

	// An update travels supplier -> customer, the reverse of a response.
	require.NotNil(t, ar.SenderParty)
	assert.Equal(t, "Seller Co", ar.SenderParty.PartyName.Name)
	require.NotNil(t, ar.ReceiverParty)
	assert.Equal(t, "Buyer Co", ar.ReceiverParty.PartyName.Name)
}

const sampleGenericMultiResponse = `<?xml version="1.0" encoding="UTF-8"?>
<ApplicationResponse xmlns="urn:oasis:names:specification:ubl:schema:xsd:ApplicationResponse-2" xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">
  <cbc:ID>RESP-MULTI</cbc:ID>
  <cbc:IssueDate>2026-05-29</cbc:IssueDate>
  <cac:SenderParty><cac:PartyName><cbc:Name>Buyer Co</cbc:Name></cac:PartyName></cac:SenderParty>
  <cac:ReceiverParty><cac:PartyName><cbc:Name>Seller Co</cbc:Name></cac:PartyName></cac:ReceiverParty>
  <cac:DocumentResponse>
    <cac:Response><cbc:ReferenceID>1</cbc:ReferenceID></cac:Response>
    <cac:DocumentReference><cbc:ID>INV-1</cbc:ID></cac:DocumentReference>
  </cac:DocumentResponse>
  <cac:DocumentResponse>
    <cac:Response><cbc:ReferenceID>2</cbc:ReferenceID></cac:Response>
    <cac:DocumentReference><cbc:ID>INV-2</cbc:ID></cac:DocumentReference>
  </cac:DocumentResponse>
</ApplicationResponse>`

func TestParseApplicationResponseFansOutLines(t *testing.T) {
	doc, err := ubl.Parse([]byte(sampleGenericMultiResponse))
	require.NoError(t, err)
	ar, ok := doc.(*ubl.ApplicationResponse)
	require.True(t, ok)

	env, err := ar.Convert()
	require.NoError(t, err)
	st, ok := env.Extract().(*bill.Status)
	require.True(t, ok)

	// Each DocumentResponse maps back to its own status line.
	require.Len(t, st.Lines, 2)
	require.NotNil(t, st.Lines[0].Doc)
	assert.Equal(t, "INV-1", st.Lines[0].Doc.Code.String())
	require.NotNil(t, st.Lines[1].Doc)
	assert.Equal(t, "INV-2", st.Lines[1].Doc.Code.String())
}
