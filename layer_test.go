package ubl_test

import (
	"errors"
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testCustomizationID = "urn:example:layer-test"
	testProfileID       = "layer-test"
)

// testLayer records the order it is applied in through the invoice notes,
// and marks every party it converts.
func testLayer(name string) *ubl.Layer {
	return &ubl.Layer{
		ConvertParty: func(_ *ubl.Context, _ *org.Party, out *ubl.Party) {
			out.IndustryClassificationCode += name
		},
		ConvertInvoice: func(_ *ubl.Context, _ *bill.Invoice, out *ubl.Invoice) error {
			out.Note = append(out.Note, name)
			return nil
		},
		ParseParty: func(_ *ubl.Context, _ *ubl.Party, out *org.Party) {
			out.Alias += name
		},
		ParseInvoice: func(_ *ubl.Context, _ *ubl.Invoice, out *bill.Invoice) error {
			out.Code = cbc.Code(out.Code.String() + "-" + name)
			return nil
		},
	}
}

var contextLayerTest = ubl.Context{
	Key:                   "ubl+layer-test",
	Schemas:               ubl.ContextEN16931.Schemas,
	CustomizationID:       testCustomizationID,
	OutputCustomizationID: "urn:example:layer-test-output",
	Addons:                ubl.ContextEN16931.Addons,
	Match: func(_, profileID string) bool {
		return profileID == testProfileID
	},
	Fallback: func(customizationID, _ string) bool {
		return customizationID == "urn:example:layer-test-fallback"
	},
	Layers: []*ubl.Layer{testLayer("a"), testLayer("b")},
}

func init() {
	ubl.RegisterContexts(contextLayerTest, contextLayerErrors)
}

func TestLayers(t *testing.T) {
	env := loadTestEnvelope(t, "invoice-minimal.json")
	doc, err := ubl.ConvertInvoice(env, ubl.WithContext(contextLayerTest))
	require.NoError(t, err)

	t.Run("convert applies the layers in order", func(t *testing.T) {
		n := len(doc.Note)
		require.GreaterOrEqual(t, n, 2)
		assert.Equal(t, []string{"a", "b"}, doc.Note[n-2:])
		assert.Equal(t, "ab", doc.AccountingSupplierParty.Party.IndustryClassificationCode)
		assert.Equal(t, "ab", doc.AccountingCustomerParty.Party.IndustryClassificationCode)
	})

	t.Run("parse applies the layers in order", func(t *testing.T) {
		data, err := ubl.Bytes(doc)
		require.NoError(t, err)
		parsed, err := ubl.Parse(data)
		require.NoError(t, err)
		out, err := parsed.(*ubl.Invoice).Convert()
		require.NoError(t, err)
		inv, ok := out.Extract().(*bill.Invoice)
		require.True(t, ok)
		assert.Equal(t, "SAMPLE-001-a-b", inv.Code.String())
		assert.Equal(t, "ab", inv.Supplier.Alias)
	})

	t.Run("other contexts are unaffected", func(t *testing.T) {
		doc, err := ubl.ConvertInvoice(env, ubl.WithContext(ubl.ContextEN16931))
		require.NoError(t, err)
		assert.Empty(t, doc.AccountingSupplierParty.Party.IndustryClassificationCode)
	})
}

func TestFindContextMatching(t *testing.T) {
	t.Run("match before customization", func(t *testing.T) {
		ctx := ubl.FindContext(ubl.ContextEN16931.CustomizationID, testProfileID)
		require.NotNil(t, ctx)
		assert.Equal(t, contextLayerTest.Key, ctx.Key)
	})
	t.Run("output customization", func(t *testing.T) {
		ctx := ubl.FindContext("urn:example:layer-test-output", "")
		require.NotNil(t, ctx)
		assert.Equal(t, contextLayerTest.Key, ctx.Key)
	})
	t.Run("fallback after customization", func(t *testing.T) {
		ctx := ubl.FindContext("urn:example:layer-test-fallback", "")
		require.NotNil(t, ctx)
		assert.Equal(t, contextLayerTest.Key, ctx.Key)
	})
	t.Run("registered converter detects its own context", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")
		out, err := convert.Export(env, contextLayerTest.Key)
		require.NoError(t, err)
		ctx, err := convert.Detect(out.Data)
		require.NoError(t, err)
		assert.Equal(t, contextLayerTest.Key, ctx.Key)
	})
}

var errLayer = errors.New("layer failed")

// contextLayerErrors has layers that fail in every direction.
var contextLayerErrors = ubl.Context{
	Key:             "ubl+layer-errors",
	CustomizationID: "urn:example:layer-errors",
	Layers: []*ubl.Layer{{
		ConvertInvoice: func(*ubl.Context, *bill.Invoice, *ubl.Invoice) error { return errLayer },
		ConvertStatus: func(*ubl.Context, *bill.Status, *ubl.ApplicationResponse) error {
			return errLayer
		},
		ParseInvoice: func(*ubl.Context, *ubl.Invoice, *bill.Invoice) error { return errLayer },
		ParseStatus: func(*ubl.Context, *ubl.ApplicationResponse, *bill.Status) error {
			return errLayer
		},
	}},
}

func TestLayerErrors(t *testing.T) {
	ctx := contextLayerErrors

	t.Run("convert invoice", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")
		_, err := ubl.ConvertInvoice(env, ubl.WithContext(ctx))
		assert.ErrorIs(t, err, errLayer)
	})
	t.Run("convert status", func(t *testing.T) {
		env, err := gobl.Envelop(basePeppolStatus())
		require.NoError(t, err)
		_, err = ubl.Convert(env, ubl.WithContext(ctx))
		assert.ErrorIs(t, err, errLayer)
	})
	t.Run("parse invoice", func(t *testing.T) {
		in := parsedFixture(t, "peppol/base-example.xml")
		_, err := in.Convert(ubl.WithContext(ctx))
		assert.ErrorIs(t, err, errLayer)
	})
	t.Run("parse status", func(t *testing.T) {
		env, err := gobl.Envelop(basePeppolStatus())
		require.NoError(t, err)
		doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextPeppolInvoiceResponse))
		require.NoError(t, err)
		ar := doc.(*ubl.ApplicationResponse)
		ar.CustomizationID = ctx.CustomizationID
		ar.ProfileID = nil
		_, err = ar.Convert()
		assert.ErrorIs(t, err, errLayer)
	})
}

func TestLayerHelpers(t *testing.T) {
	assert.Equal(t, "2024-01-02", ubl.FormatDate(cal.MakeDate(2024, 1, 2)))
	assert.Equal(t, "10.00", ubl.NewAmount(num.MakeAmount(10, 0), "EUR").Value)
	code, ok := ubl.TaxPointCode(tax.PointDelivery)
	assert.True(t, ok)
	assert.Equal(t, "35", code)
	_, ok = ubl.TaxPointCode("unknown")
	assert.False(t, ok)
	assert.Equal(t, "ab", ubl.CleanString("a\uFFFDb"))
}
