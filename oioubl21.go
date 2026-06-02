package ubl

import (
	"strings"

	oioubl "github.com/invopop/gobl/addons/dk/oioubl-v2-1"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/tax"
)

// oioubl21TaxCategoryID returns the value to emit as cac:TaxCategory/cbc:ID. The
// dk-oioubl addon precomputes the OIOUBL taxcategoryid-1.1 code (StandardRated,
// …) in the dk-oioubl-tax-category extension; other profiles fall back to the
// UNTDID category, which they use directly.
func oioubl21TaxCategoryID(ext tax.Extensions) string {
	if c := ext.Get(oioubl.ExtKeyTaxCategory); c != "" {
		return c.String()
	}
	return ext.Get(untdid.ExtKeyTaxCategory).String()
}

const (
	// These OIOUBL wire values equal the dk-oioubl addon's extension values by
	// construction (the converter writes the extension straight to the XML and
	// reads it back on parse), so they are sourced from the addon to avoid drift.
	oioubl21PaymentChannelIBAN       = string(oioubl.ExtValuePaymentChannelIBAN)
	oioubl21PaymentChannelGiro       = string(oioubl.ExtValuePaymentChannelGiro)
	oioubl21PaymentChannelFIK        = string(oioubl.ExtValuePaymentChannelFIK)
	oioubl21TaxCategoryStandardRated = string(oioubl.ExtValueTaxCategoryStandardRated)
	oioubl21TaxCategoryZeroRated     = string(oioubl.ExtValueTaxCategoryZeroRated)
	oioubl21TaxCategoryReverseCharge = string(oioubl.ExtValueTaxCategoryReverseCharge)

	// Serialization-only values with no addon counterpart.
	oioubl21TaxSchemeVATCode = "63"     // taxschemeid-1.2 VAT (Moms)
	oioubl21SchemeDKCVR      = "DK:CVR" // party endpoint scheme
)

// applyOIOUBL21 post-processes an assembled UBL document into OIOUBL 2.1 form:
// scheme/codelist attributes, party identifiers, payment means and the
// OIOUBL-specific total adjustments. These are serialization transforms, not
// validations — OIOUBL validation lives entirely in the dk-oioubl GOBL addon.
func applyOIOUBL21(out *Invoice) {
	if out == nil {
		return
	}

	applyOIOUBL21TypeCode(out.InvoiceTypeCode)
	applyOIOUBL21TypeCode(out.CreditNoteTypeCode)

	applyOIOUBL21Party(out.AccountingSupplierParty.Party)
	applyOIOUBL21Party(out.AccountingCustomerParty.Party)

	applyOIOUBL21PaymentMeans(out)

	if out.PaymentTerms != nil && out.PaymentTerms.Amount == nil {
		out.PaymentTerms.Amount = &Amount{
			Value:      out.LegalMonetaryTotal.PayableAmount.Value,
			CurrencyID: out.LegalMonetaryTotal.PayableAmount.CurrencyID,
		}
	}

	// F-INV127: OIOUBL 2.1 defines TaxExclusiveAmount as the sum of
	// TaxTotal/TaxSubtotal/TaxAmount (i.e. the tax amount itself), not
	// the pre-tax subtotal as in generic UBL.
	if len(out.TaxTotal) > 0 {
		out.LegalMonetaryTotal.TaxExclusiveAmount = out.TaxTotal[0].TaxAmount
	}
	if out.CreditNoteTypeCode != nil {
		for i := range out.BillingReference {
			if ref := out.BillingReference[i]; ref != nil && ref.InvoiceDocumentReference != nil {
				// OIOUBL 2.1 credit-note schematron rejects DocumentTypeCode here.
				ref.InvoiceDocumentReference.DocumentTypeCode = ""
			}
		}
	}

	applyOIOUBL21TaxCategories(out)
}

// applyOIOUBL21PaymentMeans stamps the paymentchannelcode-1.1 list ID, defaults
// the per-means due date and strips a redundant FinancialInstitutionBranch from
// IBAN accounts (F-LIB295). The channel value itself is set from the
// dk-oioubl-payment-channel extension when the payment means is built.
func applyOIOUBL21PaymentMeans(out *Invoice) {
	for i := range out.PaymentMeans {
		pm := &out.PaymentMeans[i]
		if pm.PaymentChannelCode != nil {
			listID := "urn:oioubl:codelist:paymentchannelcode-1.1"
			pm.PaymentChannelCode.ListID = &listID
			if pm.PaymentChannelCode.Value == oioubl21PaymentChannelIBAN && pm.PayeeFinancialAccount != nil && pm.PayeeFinancialAccount.FinancialInstitutionBranch != nil {
				pm.PayeeFinancialAccount.FinancialInstitutionBranch.ID = nil
			}
		}
		if out.DueDate != "" && pm.PaymentDueDate == nil {
			d := out.DueDate
			pm.PaymentDueDate = &d
		}
	}
	if len(out.PaymentMeans) > 0 && out.DueDate != "" {
		out.DueDate = ""
	}
}

// applyOIOUBL21TaxCategories maps every TaxCategory and ClassifiedTaxCategory on
// the document totals, lines and allowance/charges to the OIOUBL codes. Without
// it they keep the raw GOBL values (cbc:ID "S", TaxScheme "VAT") and fail
// F-LIB066/F-LIB075.
func applyOIOUBL21TaxCategories(out *Invoice) {
	for i := range out.TaxTotal {
		for j := range out.TaxTotal[i].TaxSubtotal {
			applyOIOUBL21TaxCategory(&out.TaxTotal[i].TaxSubtotal[j].TaxCategory)
		}
	}
	for i := range out.AllowanceCharge {
		for _, tc := range out.AllowanceCharge[i].TaxCategory {
			applyOIOUBL21TaxCategory(tc)
		}
	}
	applyOIOUBL21LineTaxCategories(out.InvoiceLines)
	applyOIOUBL21LineTaxCategories(out.CreditNoteLines)
}

// applyOIOUBL21LineTaxCategories maps the tax categories on a set of lines: the
// item classified category, the line-level subtotals, and any promoted
// allowance/charges. Invoice and credit-note lines share the InvoiceLine type.
func applyOIOUBL21LineTaxCategories(lines []InvoiceLine) {
	for i := range lines {
		line := &lines[i]
		if line.Item != nil && line.Item.ClassifiedTaxCategory != nil {
			applyOIOUBL21ClassifiedTaxCategory(line.Item.ClassifiedTaxCategory)
		}
		for j := range line.TaxTotal {
			for k := range line.TaxTotal[j].TaxSubtotal {
				applyOIOUBL21TaxCategory(&line.TaxTotal[j].TaxSubtotal[k].TaxCategory)
			}
		}
		for _, ac := range line.AllowanceCharge {
			for _, tc := range ac.TaxCategory {
				applyOIOUBL21TaxCategory(tc)
			}
		}
	}
}

func applyOIOUBL21TypeCode(t *IDType) {
	if t == nil {
		return
	}
	listID := "urn:oioubl:codelist:invoicetypecode-1.1"
	listAgencyID := "320"
	t.ListID = &listID
	t.ListAgencyID = &listAgencyID
}

// oioubl21EndpointSchemes maps the ISO 6523 ICDs that Peppol-style endpoints
// use to the symbolic OIOUBL EndpointID schemeID codelist (F-LIB179).
var oioubl21EndpointSchemes = map[string]string{
	"0088": "GLN",
	"0184": oioubl21SchemeDKCVR,
	"0198": "DK:SE",
}

func applyOIOUBL21Party(p *Party) {
	if p == nil {
		return
	}
	if p.EndpointID != nil {
		if mapped, ok := oioubl21EndpointSchemes[p.EndpointID.SchemeID]; ok {
			p.EndpointID.SchemeID = mapped
		}
		// OIOUBL CVR endpoints must carry the DK-prefixed form (F-LIB180).
		if p.EndpointID.SchemeID == oioubl21SchemeDKCVR && !strings.HasPrefix(p.EndpointID.Value, "DK") {
			p.EndpointID.Value = "DK" + p.EndpointID.Value
		}
	}
	if p.PartyName == nil && len(p.PartyIdentification) == 0 {
		if p.PartyLegalEntity != nil && p.PartyLegalEntity.RegistrationName != nil {
			p.PartyName = &PartyName{
				Name: *p.PartyLegalEntity.RegistrationName,
			}
		}
	}
	if p.PostalAddress != nil && p.PostalAddress.AddressFormatCode == nil {
		listID := "urn:oioubl:codelist:addressformatcode-1.1"
		listAgencyID := "320"
		p.PostalAddress.AddressFormatCode = &IDType{
			ListID:       &listID,
			ListAgencyID: &listAgencyID,
			Value:        "StructuredDK",
		}
	}
	if p.PartyTaxScheme != nil {
		for i := range p.PartyTaxScheme {
			pts := &p.PartyTaxScheme[i]
			if pts.CompanyID != nil {
				scheme := "DK:SE"
				pts.CompanyID.SchemeID = &scheme
				if !strings.HasPrefix(pts.CompanyID.Value, "DK") {
					pts.CompanyID.Value = "DK" + pts.CompanyID.Value
				}
			}
			applyOIOUBL21TaxScheme(pts.TaxScheme)
		}
	}
	if p.PartyLegalEntity != nil && p.PartyLegalEntity.CompanyID != nil {
		scheme := oioubl21SchemeDKCVR
		p.PartyLegalEntity.CompanyID.SchemeID = &scheme
		if !strings.HasPrefix(p.PartyLegalEntity.CompanyID.Value, "DK") {
			p.PartyLegalEntity.CompanyID.Value = "DK" + p.PartyLegalEntity.CompanyID.Value
		}
	}
}

// oioubl21CategoryID stamps the taxcategoryid-1.1 codelist attributes onto a
// tax-category cbc:ID, defaulting an absent category to StandardRated. The code
// itself is set at build time from the dk-oioubl-tax-category extension (see
// oioubl21TaxCategoryID).
func oioubl21CategoryID(id *IDType) *IDType {
	if id == nil {
		id = &IDType{Value: oioubl21TaxCategoryStandardRated}
	}
	schemeID := "urn:oioubl:id:taxcategoryid-1.1"
	schemeAgencyID := "320"
	id.SchemeID = &schemeID
	id.SchemeAgencyID = &schemeAgencyID
	return id
}

func applyOIOUBL21TaxCategory(tc *TaxCategory) {
	if tc == nil {
		return
	}
	tc.ID = oioubl21CategoryID(tc.ID)
	applyOIOUBL21TaxScheme(tc.TaxScheme)
}

func applyOIOUBL21ClassifiedTaxCategory(tc *ClassifiedTaxCategory) {
	if tc == nil {
		return
	}
	tc.ID = oioubl21CategoryID(tc.ID)
	applyOIOUBL21TaxScheme(tc.TaxScheme)
}

func applyOIOUBL21TaxScheme(ts *TaxScheme) {
	if ts == nil {
		return
	}
	schemeID := "urn:oioubl:id:taxschemeid-1.2"
	schemeAgencyID := "320"
	ts.ID = IDType{
		SchemeID:       &schemeID,
		SchemeAgencyID: &schemeAgencyID,
		Value:          oioubl21TaxSchemeVATCode,
	}
	name := "Moms"
	ts.Name = &name
}
