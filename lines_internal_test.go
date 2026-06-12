package ubl

import (
	"encoding/xml"
	"testing"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddLinesOriginCountryOIOUBLGate(t *testing.T) {
	mk := func() *bill.Invoice {
		amount := num.MakeAmount(10000, 2)
		return &bill.Invoice{
			Currency: "DKK",
			Lines: []*bill.Line{
				{
					Index:    1,
					Quantity: num.MakeAmount(1, 0),
					Item:     &org.Item{Name: "Widget", Origin: "DE"},
					Sum:      &amount,
					Total:    &amount,
				},
			},
		}
	}

	t.Run("OIOUBL omits cac:OriginCountry (F-INV211/F-CRN109)", func(t *testing.T) {
		ui := &Invoice{XMLName: xml.Name{Local: "Invoice"}}
		ui.addLines(mk(), ContextOIOUBL21)
		require.Len(t, ui.InvoiceLines, 1)
		assert.Nil(t, ui.InvoiceLines[0].Item.OriginCountry)
	})

	t.Run("non-OIOUBL keeps cac:OriginCountry", func(t *testing.T) {
		ui := &Invoice{XMLName: xml.Name{Local: "Invoice"}}
		ui.addLines(mk(), ContextEN16931)
		require.Len(t, ui.InvoiceLines, 1)
		require.NotNil(t, ui.InvoiceLines[0].Item.OriginCountry)
		assert.Equal(t, "DE", ui.InvoiceLines[0].Item.OriginCountry.IdentificationCode)
	})
}
