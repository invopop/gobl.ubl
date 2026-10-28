package ubl_test

import (
	"strings"
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
		ubl.FormatPeppol.Key,
	} {
		ctx := convert.FormatFor(k)
		require.NotNil(t, ctx, k)
		assert.Equal(t, cbc.Key("ubl"), ctx.Syntax)
		assert.Empty(t, ctx.Countries)
	}

	t.Run("layered keys", func(t *testing.T) {
		assert.Equal(t, cbc.Key("ubl+en16931-v2017"), ubl.FormatEN16931.Key)
		assert.Equal(t, cbc.Key("ubl+peppol+bis-v3"), ubl.FormatPeppol.Key)
	})
}

func TestConverterDetect(t *testing.T) {
	tests := []struct {
		file string
		key  cbc.Key
	}{
		{"en16931/ubl-example1.xml", ubl.FormatEN16931.Key},
		{"peppol/base-example.xml", ubl.FormatPeppol.Key},
		{"peppol/sg-invoice.xml", ubl.KeyUBL},
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
		data, err := testLoadXML("peppol/base-example.xml")
		require.NoError(t, err)
		_, err = convert.Detect(data, ubl.FormatEN16931.Key)
		assert.ErrorIs(t, err, convert.ErrUnknownFormat)
	})
}

func TestReadDocumentContext(t *testing.T) {
	data, err := testLoadXML("peppol/base-example.xml")
	require.NoError(t, err)
	in := convert.NewInput(data)

	dc := ubl.ReadHeader(in)
	require.NoError(t, dc.Err)
	assert.Equal(t, ubl.NamespaceUBLInvoice, dc.Namespace)
	assert.Equal(t, ubl.FormatPeppol.CustomizationID, dc.CustomizationID)
	assert.Equal(t, ubl.FormatPeppol.ProfileID, dc.ProfileID)
	assert.Same(t, dc, ubl.ReadHeader(in), "read only once")

	t.Run("malformed", func(t *testing.T) {
		dc := ubl.ReadHeader(convert.NewInput([]byte(`<Invoice><cbc:CustomizationID>`)))
		assert.Error(t, dc.Err)
	})
}

func TestConverterImport(t *testing.T) {
	data, err := testLoadXML("peppol/base-example.xml")
	require.NoError(t, err)
	res, err := convert.Import(data)
	require.NoError(t, err)
	assert.Equal(t, ubl.FormatPeppol.Key, res.Format.Key)
	inv, ok := res.Envelope.Extract().(*bill.Invoice)
	require.True(t, ok)
	assert.Contains(t, inv.GetAddons(), en16931.V2017)
}

func TestConverterImportRouting(t *testing.T) {
	data, err := testLoadXML("peppol/base-example.xml")
	require.NoError(t, err)
	res, err := convert.Import(data, convert.WithRouting("iso6523-actorid-upis::0088:sender", "iso6523-actorid-upis::0088:receiver"))
	require.NoError(t, err)
	assert.Equal(t, cbc.URI("iso6523-actorid-upis::0088:sender"), res.Envelope.Head.From)
	assert.Equal(t, cbc.URI("iso6523-actorid-upis::0088:receiver"), res.Envelope.Head.To)
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
	t.Run("peppol", func(t *testing.T) {
		env := loadTestEnvelope(t, "peppol/invoice-minimal.json")
		out, err := convert.Export(env, ubl.FormatPeppol.Key)
		require.NoError(t, err)
		assert.Equal(t, ubl.FormatPeppol.Key, out.Format.Key)
		assert.Contains(t, string(out.Data), ubl.FormatPeppol.CustomizationID)

		ctx, err := convert.Detect(out.Data)
		require.NoError(t, err)
		assert.Equal(t, ubl.FormatPeppol.Key, ctx.Key, "detected again")
	})
	t.Run("skips format for another schema", func(t *testing.T) {
		env, err := gobl.Envelop(basePeppolStatus())
		require.NoError(t, err)
		out, err := convert.Export(env, ubl.FormatEN16931.Key, convert.WithAlternatives(ubl.FormatPeppol.Key))
		require.NoError(t, err)
		assert.Equal(t, ubl.FormatPeppol.Key, out.Format.Key)
		assert.Contains(t, string(out.Data), ubl.SpecificationPeppolInvoiceResponse.CustomizationID)
	})
	t.Run("not supported", func(t *testing.T) {
		env, err := gobl.Envelop(basePeppolStatus())
		require.NoError(t, err)
		_, err = convert.Export(env, ubl.FormatEN16931.Key, convert.WithAlternatives(ubl.KeyUBL))
		assert.ErrorIs(t, err, convert.ErrNotSupported)
	})
	t.Run("missing specification addon", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")
		env.Extract().(*bill.Invoice).SetAddons()
		_, err := convert.Export(env, ubl.FormatPeppol.Key)
		assert.ErrorIs(t, err, convert.ErrNotSupported)
		var ge *gobl.Error
		require.ErrorAs(t, err, &ge)
		require.NotNil(t, ge.Faults())
		f := ge.Faults().First()
		assert.Equal(t, rules.Code("UBL-PEPPOL-BIS-V3-BILL-INVOICE-01"), f.Code())
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
		env := loadTestEnvelope(t, "peppol/invoice-minimal.json")
		csv := &convert.Attachment{Name: "lines.csv", MIME: "text/csv", Data: []byte("a,b")}
		out, err := convert.Export(env, ubl.FormatPeppol.Key, convert.WithAttachments(pdf, csv))
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
		env, err := gobl.Envelop(basePeppolStatus())
		require.NoError(t, err)
		_, err = convert.Export(env, ubl.FormatPeppol.Key, convert.WithAttachments(pdf))
		assert.ErrorIs(t, err, convert.ErrConversion)
		assert.ErrorContains(t, err, ubl.ErrUnsupportedAttachments.Error())
	})
	t.Run("formats embed them", func(t *testing.T) {
		assert.True(t, convert.FormatFor(ubl.FormatPeppol.Key).Attachments)
		assert.False(t, convert.FormatFor(ubl.KeyUBL).Attachments)
	})
}

// A document in a single-byte encoding it declares itself imports like a UTF-8
// one, its non-ASCII characters intact.
func TestConverterImportLatin1(t *testing.T) {
	data, err := testLoadXML("peppol/base-example.xml")
	require.NoError(t, err)
	doc := strings.Replace(string(data), `encoding="UTF-8"`, `encoding="ISO-8859-1"`, 1)
	doc = strings.Replace(doc, "SupplierOfficialName Ltd", "Société Démo", 1)
	latin1 := make([]byte, 0, len(doc))
	for _, r := range doc {
		require.Less(t, r, rune(0x100))
		latin1 = append(latin1, byte(r))
	}

	res, err := convert.Import(latin1)
	require.NoError(t, err)
	inv, ok := res.Envelope.Extract().(*bill.Invoice)
	require.True(t, ok)
	assert.Equal(t, "Société Démo", inv.Supplier.Name)
}
