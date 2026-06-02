package ubl

import (
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
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
