package ubl

import (
	"testing"

	oioubl "github.com/invopop/gobl/addons/dk/oioubl-v2-1"
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

// The status-line parser handles ApplicationResponses that arrive from the
// NemHandel network, so the partial and malformed shapes below must degrade
// gracefully rather than panic.

func TestGoblStatusLineNilDocumentResponse(t *testing.T) {
	o := &options{context: ContextOIOUBL21}
	line, err := goblStatusLine(nil, o)
	require.NoError(t, err)
	assert.NotNil(t, line)
	assert.Empty(t, line.Key)
	assert.Nil(t, line.Doc)
}

func TestGoblStatusLineRecordsResponseCode(t *testing.T) {
	o := &options{context: ContextOIOUBL21}
	// The parser records the raw responsecode-1.1 value in the addon extension
	// (verbatim, even unknown values); the dk-oioubl normalizer recovers the GOBL
	// status event from it during Calculate.
	for _, code := range []string{"BusinessAccept", "NotAKnownCode"} {
		dr := &DocumentResponse{Response: &Response{
			ResponseCode: &IDType{Value: code},
			Description:  []string{"a reason"},
		}}
		line, err := goblStatusLine(dr, o)
		require.NoError(t, err)
		assert.Equal(t, code, line.Ext.Get(oioubl.ExtKeyResponseCode).String(), "code %q", code)
		assert.Empty(t, line.Key, "the parser does not set the key directly")
		assert.Equal(t, "a reason", line.Description)
	}
}

func TestGoblStatusLineDocumentReference(t *testing.T) {
	o := &options{context: ContextOIOUBL21}
	dr := &DocumentResponse{DocumentReference: &ResponseDocumentReference{
		ID:               "INV-1",
		UUID:             "5d3a2c8e-1f6b-4f7a-9c1d-2b3e4f5a6b7c",
		IssueDate:        "2024-02-03",
		DocumentTypeCode: &IDType{Value: responseDocTypeCreditNote},
	}}
	line, err := goblStatusLine(dr, o)
	require.NoError(t, err)
	require.NotNil(t, line.Doc)
	assert.Equal(t, "INV-1", line.Doc.Code.String())
	assert.Equal(t, bill.InvoiceTypeCreditNote, line.Doc.Type)
	require.NotNil(t, line.Doc.IssueDate)
}

func TestGoblStatusLineBadReferenceDate(t *testing.T) {
	o := &options{context: ContextOIOUBL21}
	dr := &DocumentResponse{DocumentReference: &ResponseDocumentReference{ID: "INV-1", IssueDate: "03/02/2024"}}
	_, err := goblStatusLine(dr, o)
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
