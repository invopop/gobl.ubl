package ubl

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOIOUBLResponseDocType(t *testing.T) {
	assert.Equal(t, responseDocTypeCreditNote, oioublResponseDocType(bill.InvoiceTypeCreditNote))
	assert.Equal(t, responseDocTypeInvoice, oioublResponseDocType(bill.InvoiceTypeStandard))
	assert.Equal(t, responseDocTypeInvoice, oioublResponseDocType(""))
}

func TestOIOUBLResponseReferenceID(t *testing.T) {
	assert.Equal(t, 1, responseReferenceID(0))
	assert.Equal(t, 1, responseReferenceID(-3))
	assert.Equal(t, 4, responseReferenceID(4))
}

// TestOIOUBLResponseCodeSymmetry checks every emitted response code parses back
// to a status event, so convert and parse stay in sync.
func TestOIOUBLResponseCodeSymmetry(t *testing.T) {
	for event, code := range oioublResponseCodes {
		got, ok := goblResponseEvents[code]
		assert.True(t, ok, "response code %q has no reverse mapping", code)
		assert.Equal(t, event, got, "response code %q does not round-trip", code)
	}
}

func TestOIOUBLResponseUnsupportedEvent(t *testing.T) {
	_, ok := oioublResponseCodes[bill.StatusEventPaid]
	assert.False(t, ok, "paid event is not representable in OIOUBL ApplicationResponse")
}

// The status-line parser handles ApplicationResponses that arrive from the
// NemHandel network, so the partial and malformed shapes below must degrade
// gracefully rather than panic.

func TestGoblStatusLineNilDocumentResponse(t *testing.T) {
	o := &options{context: ContextOIOUBL21}
	line, err := (&ApplicationResponse{}).goblStatusLine(o)
	require.NoError(t, err)
	assert.NotNil(t, line)
	assert.Empty(t, line.Key)
	assert.Nil(t, line.Doc)
}

func TestGoblStatusLineResponseCodes(t *testing.T) {
	o := &options{context: ContextOIOUBL21}
	for code, want := range goblResponseEvents {
		ar := &ApplicationResponse{DocumentResponse: &DocumentResponse{
			Response: &Response{
				ResponseCode: &IDType{Value: code},
				Description:  []string{"a reason"},
			},
		}}
		line, err := ar.goblStatusLine(o)
		require.NoError(t, err)
		assert.Equal(t, want, line.Key, "code %q", code)
		assert.Equal(t, "a reason", line.Description)
	}
}

func TestGoblStatusLineUnknownCode(t *testing.T) {
	o := &options{context: ContextOIOUBL21}
	ar := &ApplicationResponse{DocumentResponse: &DocumentResponse{
		Response: &Response{ResponseCode: &IDType{Value: "NotAKnownCode"}},
	}}
	line, err := ar.goblStatusLine(o)
	require.NoError(t, err)
	assert.Empty(t, line.Key, "an unknown response code maps to an empty key, never a panic")
}

func TestGoblStatusLineDocumentReference(t *testing.T) {
	o := &options{context: ContextOIOUBL21}
	ar := &ApplicationResponse{DocumentResponse: &DocumentResponse{
		DocumentReference: &ResponseDocumentReference{
			ID:               "INV-1",
			UUID:             "5d3a2c8e-1f6b-4f7a-9c1d-2b3e4f5a6b7c",
			IssueDate:        "2024-02-03",
			DocumentTypeCode: &IDType{Value: responseDocTypeCreditNote},
		},
	}}
	line, err := ar.goblStatusLine(o)
	require.NoError(t, err)
	require.NotNil(t, line.Doc)
	assert.Equal(t, "INV-1", line.Doc.Code.String())
	assert.Equal(t, bill.InvoiceTypeCreditNote, line.Doc.Type)
	require.NotNil(t, line.Doc.IssueDate)
}

func TestGoblStatusLineBadReferenceDate(t *testing.T) {
	o := &options{context: ContextOIOUBL21}
	ar := &ApplicationResponse{DocumentResponse: &DocumentResponse{
		DocumentReference: &ResponseDocumentReference{ID: "INV-1", IssueDate: "03/02/2024"},
	}}
	_, err := ar.goblStatusLine(o)
	assert.Error(t, err, "a malformed reference date is reported, not silently dropped")
}

func TestGoblStatusIssueTimeAndErrors(t *testing.T) {
	o := new(options)
	o.context = ContextOIOUBL21

	st, err := (&ApplicationResponse{ID: "R1", IssueDate: "2024-02-03", IssueTime: "10:30:00"}).goblStatus(o)
	require.NoError(t, err)
	require.NotNil(t, st.IssueTime)
	assert.Equal(t, bill.StatusTypeResponse, st.Type)

	_, err = (&ApplicationResponse{ID: "R1", IssueDate: "nope"}).goblStatus(o)
	assert.Error(t, err, "a malformed issue date is reported")

	_, err = (&ApplicationResponse{ID: "R1", IssueDate: "2024-02-03", IssueTime: "99:99"}).goblStatus(o)
	assert.Error(t, err, "a malformed issue time is reported")
}
