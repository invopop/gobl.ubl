package ubl_test

import (
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConverterContexts(t *testing.T) {
	for _, k := range []cbc.Key{
		ubl.KeyUBL,
		ubl.FormatEN16931.Key,
	} {
		ctx := convert.FormatFor(k)
		require.NotNil(t, ctx, k)
		assert.Equal(t, cbc.Key("ubl"), ctx.Syntax)
		assert.Empty(t, ctx.Countries)
	}

	t.Run("layered keys", func(t *testing.T) {
		assert.Equal(t, cbc.Key("ubl+en16931-v2017"), ubl.FormatEN16931.Key)
	})
}

func TestConverterDetect(t *testing.T) {
	tests := []struct {
		file string
		key  cbc.Key
	}{
		{"en16931/ubl-example1.xml", ubl.FormatEN16931.Key},
		{"en16931/credit-note1.xml", ubl.KeyUBL},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			data, err := testLoadXML(tt.file)
			require.NoError(t, err)
			ctx, err := convert.Detect(data)
			require.NoError(t, err)
			assert.Equal(t, tt.key, ctx.Key)
		})
	}

	t.Run("not ubl", func(t *testing.T) {
		for _, data := range []string{
			`<?xml version="1.0"?><Invoice xmlns="urn:example"/>`,
			`{"$schema":"https://gobl.org/draft-0/bill/invoice"}`,
			``,
		} {
			_, err := convert.Detect([]byte(data))
			assert.ErrorIs(t, err, convert.ErrUnknownFormat, data)
		}
	})

	t.Run("limited to keys", func(t *testing.T) {
		data, err := testLoadXML("en16931/ubl-example1.xml")
		require.NoError(t, err)
		_, err = convert.Detect(data, formatSpecsTest.Key)
		assert.ErrorIs(t, err, convert.ErrUnknownFormat)
	})
}

func TestReadDocumentContext(t *testing.T) {
	data, err := testLoadXML("en16931/ubl-example2.xml")
	require.NoError(t, err)
	in := convert.NewInput(data)

	dc := ubl.ReadHeader(in)
	require.NoError(t, dc.Err)
	assert.Equal(t, ubl.NamespaceUBLInvoice, dc.Namespace)
	assert.Equal(t, ubl.FormatEN16931.CustomizationID, dc.CustomizationID)
	assert.Equal(t, "Invoicing on purchase order", dc.ProfileID)
	assert.Same(t, dc, ubl.ReadHeader(in), "read only once")

	t.Run("malformed", func(t *testing.T) {
		dc := ubl.ReadHeader(convert.NewInput([]byte(`<Invoice><cbc:CustomizationID>`)))
		assert.Error(t, dc.Err)
	})
}

func TestConverterImport(t *testing.T) {
	data, err := testLoadXML("en16931/ubl-example1.xml")
	require.NoError(t, err)
	res, err := convert.Import(data)
	require.NoError(t, err)
	assert.Equal(t, ubl.FormatEN16931.Key, res.Format.Key)
	inv, ok := res.Envelope.Extract().(*bill.Invoice)
	require.True(t, ok)
	assert.Contains(t, inv.GetAddons(), en16931.V2017)
}

func TestConverterImportAttachments(t *testing.T) {
	data, err := testLoadXML("en16931/ubl-example2.xml")
	require.NoError(t, err)
	res, err := convert.Import(data)
	require.NoError(t, err)
	require.Len(t, res.Attachments, 1)
	a := res.Attachments[0]
	assert.Equal(t, "Doc2", a.ID)
	assert.Equal(t, "test.pdf", a.Name)
	assert.Equal(t, "application/pdf", a.MIME)
	assert.Equal(t, []byte("Testing Base64 encoding"), a.Data, "decoded")
}

func TestConverterExport(t *testing.T) {
	t.Run("en16931", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")
		out, err := convert.Export(env, ubl.FormatEN16931.Key)
		require.NoError(t, err)
		assert.Equal(t, ubl.FormatEN16931.Key, out.Format.Key)
		assert.Contains(t, string(out.Data), ubl.FormatEN16931.CustomizationID)

		ctx, err := convert.Detect(out.Data)
		require.NoError(t, err)
		assert.Equal(t, ubl.FormatEN16931.Key, ctx.Key, "detected again")
	})
	t.Run("skips format for another schema", func(t *testing.T) {
		env, err := gobl.Envelop(baseStatus())
		require.NoError(t, err)
		out, err := convert.Export(env, ubl.FormatEN16931.Key, convert.WithAlternatives(formatSpecsTest.Key))
		require.NoError(t, err)
		assert.Equal(t, formatSpecsTest.Key, out.Format.Key)
		assert.Contains(t, string(out.Data), specStatusTest.CustomizationID)
	})
	t.Run("not supported", func(t *testing.T) {
		env, err := gobl.Envelop(baseStatus())
		require.NoError(t, err)
		_, err = convert.Export(env, ubl.FormatEN16931.Key, convert.WithAlternatives(ubl.KeyUBL))
		assert.ErrorIs(t, err, convert.ErrNotSupported)
	})
	t.Run("missing specification addon", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")
		env.Extract().(*bill.Invoice).SetAddons()
		_, err := convert.Export(env, ubl.FormatEN16931.Key)
		assert.ErrorIs(t, err, convert.ErrNotSupported)
		var ge *gobl.Error
		require.ErrorAs(t, err, &ge)
		require.NotNil(t, ge.Faults())
		f := ge.Faults().First()
		assert.Equal(t, rules.Code("UBL-EN16931-V2017-BILL-INVOICE-01"), f.Code())
		assert.Equal(t, []string{"$.$addons"}, f.Paths())
		assert.Equal(t, "requires addon eu-en16931-v2017", f.Message())
	})
	t.Run("regional context not registered", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")
		_, err := convert.Export(env, "ubl+de-xrechnung-v3")
		assert.ErrorIs(t, err, convert.ErrUnknownFormat)
	})
}

func TestConverterExportAttachments(t *testing.T) {
	pdf := &convert.Attachment{
		ID:          "INV-PDF",
		Name:        "invoice.pdf",
		Description: "Invoice PDF",
		MIME:        "application/pdf",
		Data:        []byte("%PDF-1.7"),
	}
	t.Run("embedded in invoices", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")
		csv := &convert.Attachment{Name: "lines.csv", MIME: "text/csv", Data: []byte("a,b")}
		out, err := convert.Export(env, ubl.FormatEN16931.Key, convert.WithAttachments(pdf, csv))
		require.NoError(t, err)

		doc, err := ubl.Decode(out.Data)
		require.NoError(t, err)
		atts := doc.(*ubl.Invoice).ExtractBinaryAttachments()
		require.Len(t, atts, 2)
		assert.Equal(t, "INV-PDF", atts[0].ID)
		assert.Equal(t, "Invoice PDF", atts[0].Description)
		assert.Equal(t, "application/pdf", atts[0].MimeCode)
		assert.Equal(t, "invoice.pdf", atts[0].Filename)
		assert.Equal(t, []byte("%PDF-1.7"), atts[0].Data)
		assert.Equal(t, "lines.csv", atts[1].ID, "named by the file without an ID")

		res, err := convert.Import(out.Data)
		require.NoError(t, err)
		require.Len(t, res.Attachments, 2)
		assert.Equal(t, pdf, res.Attachments[0], "imported again")
		assert.Equal(t, "lines.csv", res.Attachments[1].Name)
		assert.Equal(t, []byte("a,b"), res.Attachments[1].Data)
	})
	t.Run("not embedded in statuses", func(t *testing.T) {
		env, err := gobl.Envelop(baseStatus())
		require.NoError(t, err)
		_, err = convert.Export(env, formatSpecsTest.Key, convert.WithAttachments(pdf))
		assert.ErrorIs(t, err, convert.ErrConversion)
		assert.ErrorContains(t, err, ubl.ErrUnsupportedAttachments.Error())
	})
	t.Run("formats embed them", func(t *testing.T) {
		assert.True(t, convert.FormatFor(ubl.FormatEN16931.Key).Attachments)
		assert.False(t, convert.FormatFor(ubl.KeyUBL).Attachments)
	})
}
