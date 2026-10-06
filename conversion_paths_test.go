package ubl_test

import (
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/addons/de/xrechnung"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/dsig"
	"github.com/invopop/gobl/note"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertDeliveryPeriod(t *testing.T) {
	env := loadTestEnvelope(t, "invoice-minimal.json")
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)
	inv.Delivery = &bill.DeliveryDetails{
		Period: &cal.Period{Start: cal.NewDate(2024, 1, 1), End: cal.NewDate(2024, 1, 31)},
	}
	require.NoError(t, env.Calculate())

	doc, err := ubl.ExportInvoice(env)
	require.NoError(t, err)
	require.Len(t, doc.Delivery, 1)
	assert.Equal(t, "2024-01-01", *doc.Delivery[0].ActualDeliveryDate)
	assert.Equal(t, "2024-01-31", *doc.Delivery[0].LatestDeliveryDate)
}

func TestConvertTaxCurrencyTotal(t *testing.T) {
	env := loadTestEnvelope(t, "invoice-minimal.json")
	inv, ok := env.Extract().(*bill.Invoice)
	require.True(t, ok)
	inv.Currency = currency.USD
	inv.ExchangeRates = []*currency.ExchangeRate{
		{From: currency.USD, To: currency.EUR, Amount: num.MakeAmount(92, 2)},
	}
	require.NoError(t, env.Calculate())

	doc, err := ubl.ExportInvoice(env)
	require.NoError(t, err)
	require.Len(t, doc.TaxTotal, 2)
	assert.Equal(t, "EUR", *doc.TaxTotal[1].TaxAmount.CurrencyID)
	assert.Equal(t, "EUR", doc.TaxCurrencyCode)
}

func TestConvertAttachmentDetails(t *testing.T) {
	doc := new(ubl.Invoice)
	doc.AddAttachments([]*org.Attachment{{
		Identify: uuid.Identify{UUID: uuid.UUID("0195ce71-dc9c-72c8-bf2c-9890a4a9f0a2")},
		Code:     "ATT-1",
		URL:      "https://example.com/att.pdf",
		Digest: &dsig.Digest{
			Algorithm: dsig.DigestSHA256,
			Value:     "abc123",
		},
	}})
	require.Len(t, doc.AdditionalDocumentReference, 1)
	ref := doc.AdditionalDocumentReference[0]
	assert.Equal(t, "0195ce71-dc9c-72c8-bf2c-9890a4a9f0a2", ref.UUID)
	assert.Equal(t, "abc123", ref.Attachment.ExternalReference.DocumentHash)
	assert.Equal(t, string(dsig.DigestSHA256), ref.Attachment.ExternalReference.HashAlgorithmMethod)

	doc.AddBinaryAttachment(ubl.BinaryAttachment{
		ID:               "BIN-1",
		Data:             []byte("text"),
		CharacterSetCode: "UTF-8",
	})
	require.Len(t, doc.AdditionalDocumentReference, 2)
	obj := doc.AdditionalDocumentReference[1].Attachment.EmbeddedDocumentBinaryObject
	require.NotNil(t, obj.CharacterSetCode)
	assert.Equal(t, "UTF-8", *obj.CharacterSetCode)
}

func TestConvertEnsureAddons(t *testing.T) {
	t.Run("no required addons", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")
		_, err := ubl.Export(env, ubl.WithFormat(ubl.Format{CustomizationID: "urn:example"}))
		assert.NoError(t, err)
	})
	t.Run("invoice fails the added addon's rules", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")
		_, err := ubl.Export(env, ubl.WithFormat(ubl.Format{Addons: []cbc.Key{xrechnung.V3}}))
		assert.Error(t, err)
	})
	t.Run("unsupported document", func(t *testing.T) {
		env, err := gobl.Envelop(&note.Message{Content: "hello"})
		require.NoError(t, err)
		_, err = ubl.Export(env)
		assert.ErrorIs(t, err, ubl.ErrUnsupportedDocumentType)
	})
}

func TestParseMalformed(t *testing.T) {
	for _, ns := range []string{
		ubl.NamespaceUBLInvoice,
		ubl.NamespaceUBLApplicationResponse,
	} {
		_, err := ubl.Decode([]byte(`<Root xmlns="` + ns + `"><cbc:ID>`))
		assert.Error(t, err, ns)
	}
}

func TestStatusPaths(t *testing.T) {
	st := basePeppolStatus()
	st.IssueTime = cal.NewTime(10, 20, 30)
	st.Lines[0].Doc.UUID = uuid.UUID("0195ce71-dc9c-72c8-bf2c-9890a4a9f0a2")
	st.Lines[0].Doc.IssueDate = cal.NewDate(2026, 5, 1)
	st.Lines[0].Reasons = []*bill.Reason{nil}
	st.Lines[0].Actions = []*bill.Action{nil}
	env, err := gobl.Envelop(st)
	require.NoError(t, err)

	doc, err := ubl.Export(env, ubl.WithFormat(ubl.FormatPeppolInvoiceResponse))
	require.NoError(t, err)
	ar, ok := doc.(*ubl.ApplicationResponse)
	require.True(t, ok)
	assert.Equal(t, "10:20:30", ar.IssueTime)
	ref := ar.DocumentResponse[0].DocumentReference
	assert.Equal(t, "0195ce71-dc9c-72c8-bf2c-9890a4a9f0a2", ref.UUID)
	assert.Equal(t, "2026-05-01", ref.IssueDate)

	t.Run("parse back", func(t *testing.T) {
		data, err := ubl.Encode(ar)
		require.NoError(t, err)
		parsed, err := ubl.Decode(data)
		require.NoError(t, err)
		in := parsed.(*ubl.ApplicationResponse)
		in.IssueTime = "10:20:30"
		in.DocumentResponse[0].DocumentReference.UUID = "0195ce71-dc9c-72c8-bf2c-9890a4a9f0a2"
		in.DocumentResponse[0].Response.Status = append(in.DocumentResponse[0].Response.Status, nil)
		in.DocumentResponse = append(in.DocumentResponse, nil, &ubl.DocumentResponse{})

		out, err := ubl.Import(in)
		require.NoError(t, err)
		st, ok := out.Extract().(*bill.Status)
		require.True(t, ok)
		require.NotNil(t, st.IssueTime)
		assert.Equal(t, "10:20:30", st.IssueTime.String())
		assert.Equal(t, "0195ce71-dc9c-72c8-bf2c-9890a4a9f0a2", st.Lines[0].Doc.UUID.String())
		assert.Len(t, st.Lines, 3)
	})

	t.Run("parse errors", func(t *testing.T) {
		data, err := ubl.Encode(ar)
		require.NoError(t, err)
		for name, mutate := range map[string]func(in *ubl.ApplicationResponse){
			"issue date": func(in *ubl.ApplicationResponse) { in.IssueDate = "bad" },
			"issue time": func(in *ubl.ApplicationResponse) { in.IssueTime = "bad" },
			"effective date": func(in *ubl.ApplicationResponse) {
				in.DocumentResponse[0].Response.EffectiveDate = "bad"
			},
			"document date": func(in *ubl.ApplicationResponse) {
				in.DocumentResponse[0].DocumentReference.IssueDate = "bad"
			},
		} {
			t.Run(name, func(t *testing.T) {
				parsed, err := ubl.Decode(data)
				require.NoError(t, err)
				in := parsed.(*ubl.ApplicationResponse)
				mutate(in)
				_, err = ubl.Import(in)
				assert.Error(t, err)
			})
		}
	})
}

func TestConverterPaths(t *testing.T) {
	t.Run("import application response", func(t *testing.T) {
		env, err := gobl.Envelop(basePeppolStatus())
		require.NoError(t, err)
		out, err := convert.Export(env, ubl.FormatPeppolInvoiceResponse.Key)
		require.NoError(t, err)

		imported, err := convert.Import(out.Data)
		require.NoError(t, err)
		_, ok := imported.Extract().(*bill.Status)
		assert.True(t, ok)
	})

	t.Run("import detected but malformed document", func(t *testing.T) {
		data := []byte(`<Invoice xmlns="` + ubl.NamespaceUBLInvoice + `" xmlns:cbc="` + ubl.NamespaceCBC + `">` +
			`<cbc:CustomizationID>urn:cen.eu:en16931:2017</cbc:CustomizationID><cbc:ID>`)
		_, err := convert.Import(data)
		assert.ErrorIs(t, err, convert.ErrConversion)
	})

	t.Run("export conversion error", func(t *testing.T) {
		env := loadTestEnvelope(t, "invoice-minimal.json")
		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		// Payment instructions need the UNTDID payment means code.
		inv.Payment.Instructions.Ext = inv.Payment.Instructions.Ext.Delete("untdid-payment-means")
		_, err := convert.Export(env, ubl.FormatEN16931.Key)
		assert.ErrorIs(t, err, convert.ErrConversion)
	})

	t.Run("unreadable header", func(t *testing.T) {
		dc := ubl.ReadHeader(convert.NewInput([]byte(`<Invoice><UBLExtensions>`)))
		assert.Error(t, dc.Err)
	})
}

func TestContextIs(t *testing.T) {
	assert.True(t, ubl.FormatPeppol.Is(ubl.FormatPeppol))
	assert.False(t, ubl.FormatPeppol.Is(ubl.FormatEN16931))
}

func TestFindContextProfileMismatch(t *testing.T) {
	assert.Nil(t, ubl.FindFormat(ubl.FormatPeppol.CustomizationID, "urn:example:other-process"))
}

func TestParsePartyPaths(t *testing.T) {
	t.Run("nil party", func(t *testing.T) {
		assert.Nil(t, ubl.ParseParty(nil))
	})

	t.Run("city subdivision fills the street extra", func(t *testing.T) {
		p := ubl.ParseParty(&ubl.Party{
			PartyName: &ubl.PartyName{Name: "Seller"},
			PostalAddress: &ubl.PostalAddress{
				StreetName:          strPtr("Main Street 1"),
				CitySubdivisionName: strPtr("Old Town"),
				Country:             &ubl.Country{IdentificationCode: "DE"},
			},
		})
		require.Len(t, p.Addresses, 1)
		assert.Equal(t, "Old Town", p.Addresses[0].StreetExtra)
	})

	t.Run("VAT scheme written differently from the default", func(t *testing.T) {
		party := func(ts *ubl.TaxScheme) *ubl.Party {
			return &ubl.Party{
				PartyName:     &ubl.PartyName{Name: "Seller"},
				PostalAddress: &ubl.PostalAddress{Country: &ubl.Country{IdentificationCode: "DE"}},
				PartyTaxScheme: []ubl.PartyTaxScheme{{
					CompanyID: &ubl.IDType{Value: "DE111111125"},
					TaxScheme: ts,
				}},
			}
		}
		p := ubl.ParseParty(party(&ubl.TaxScheme{ID: ubl.IDType{Value: "vat"}}))
		require.NotNil(t, p.TaxID)
		assert.Equal(t, cbc.Code("vat"), p.TaxID.Scheme)

		p = ubl.ParseParty(party(&ubl.TaxScheme{
			ID:          ubl.IDType{Value: "vat"},
			TaxTypeCode: &ubl.IDType{Value: "GST"},
		}))
		require.NotNil(t, p.TaxID)
		assert.Equal(t, cbc.Code("GST"), p.TaxID.Scheme)
	})
}
