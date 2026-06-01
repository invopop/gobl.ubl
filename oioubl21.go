package ubl

import (
	"strings"

	"github.com/invopop/gobl/cbc"
)

const (
	oioubl21PaymentChannelIBAN       = "IBAN"
	oioubl21TaxSchemeVATCode         = "63"
	oioubl21SchemeDKCVR              = "DK:CVR"
	oioubl21TaxCategoryStandardRated = "StandardRated"
	oioubl21TaxCategoryZeroRated     = "ZeroRated"
	oioubl21TaxCategoryReverseCharge = "ReverseCharge"
)

func applyOIOUBL21Rules(out *Invoice) {
	if out == nil {
		return
	}

	applyOIOUBL21TypeCode(out.InvoiceTypeCode)
	applyOIOUBL21TypeCode(out.CreditNoteTypeCode)

	applyOIOUBL21Party(out.AccountingSupplierParty.Party)
	applyOIOUBL21Party(out.AccountingCustomerParty.Party)

	for i := range out.PaymentMeans {
		pm := &out.PaymentMeans[i]
		if pm.PaymentChannelCode == nil {
			pm.PaymentChannelCode = &IDType{Value: oioubl21PaymentChannelIBAN}
		}
		listID := "urn:oioubl:codelist:paymentchannelcode-1.1"
		pm.PaymentChannelCode.ListID = &listID
		if pm.PaymentChannelCode.Value == oioubl21PaymentChannelIBAN && pm.PayeeFinancialAccount != nil && pm.PayeeFinancialAccount.FinancialInstitutionBranch != nil {
			pm.PayeeFinancialAccount.FinancialInstitutionBranch.ID = nil
		}
		if out.DueDate != "" && pm.PaymentDueDate == nil {
			d := out.DueDate
			pm.PaymentDueDate = &d
		}
	}
	if len(out.PaymentMeans) > 0 && out.DueDate != "" {
		out.DueDate = ""
	}

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

	for i := range out.TaxTotal {
		for j := range out.TaxTotal[i].TaxSubtotal {
			applyOIOUBL21TaxCategory(&out.TaxTotal[i].TaxSubtotal[j].TaxCategory)
		}
	}
	for i := range out.InvoiceLines {
		if line := &out.InvoiceLines[i]; line.Item != nil && line.Item.ClassifiedTaxCategory != nil {
			applyOIOUBL21ClassifiedTaxCategory(line.Item.ClassifiedTaxCategory)
		}
		for j := range out.InvoiceLines[i].TaxTotal {
			for k := range out.InvoiceLines[i].TaxTotal[j].TaxSubtotal {
				applyOIOUBL21TaxCategory(&out.InvoiceLines[i].TaxTotal[j].TaxSubtotal[k].TaxCategory)
			}
		}
	}
	for i := range out.CreditNoteLines {
		if line := &out.CreditNoteLines[i]; line.Item != nil && line.Item.ClassifiedTaxCategory != nil {
			applyOIOUBL21ClassifiedTaxCategory(line.Item.ClassifiedTaxCategory)
		}
		for j := range out.CreditNoteLines[i].TaxTotal {
			for k := range out.CreditNoteLines[i].TaxTotal[j].TaxSubtotal {
				applyOIOUBL21TaxCategory(&out.CreditNoteLines[i].TaxTotal[j].TaxSubtotal[k].TaxCategory)
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
	if p.EndpointID == nil {
		if len(p.PartyTaxScheme) > 0 && p.PartyTaxScheme[0].CompanyID != nil {
			val := p.PartyTaxScheme[0].CompanyID.Value
			if !strings.HasPrefix(val, "DK") {
				val = "DK" + val
			}
			p.EndpointID = &EndpointID{
				SchemeID: oioubl21SchemeDKCVR,
				Value:    val,
			}
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
	// F-LIB035: StructuredDK addresses require either BuildingNumber
	// or Postbox. GOBL doesn't model BuildingNumber separately, so we
	// best-effort extract the trailing token of StreetName, falling
	// back to a placeholder when nothing usable is present.
	if p.PostalAddress != nil && p.PostalAddress.BuildingNumber == nil {
		if p.PostalAddress.StreetName != nil {
			parts := strings.Fields(*p.PostalAddress.StreetName)
			if n := len(parts); n > 0 {
				bn := parts[n-1]
				p.PostalAddress.BuildingNumber = &bn
			}
		}
		if p.PostalAddress.BuildingNumber == nil {
			fallback := "1"
			p.PostalAddress.BuildingNumber = &fallback
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
	// F-INV051: AccountingCustomerParty/Contact/ID must contain a
	// value. The dk-oioubl-v2-1 addon enforces customer.People at the
	// GOBL stage (F-INV046), so this fallback only fires for paths
	// that bypass the addon.
	if p.Contact == nil {
		p.Contact = &Contact{}
	}
	if p.Contact.ID == nil {
		id := "1"
		p.Contact.ID = &id
	}
}

func applyOIOUBL21TaxCategory(tc *TaxCategory) {
	if tc == nil {
		return
	}
	if tc.ID == nil {
		tc.ID = &IDType{Value: oioubl21TaxCategoryStandardRated}
	}
	tc.ID.Value = oioubl21TaxCategoryCode(tc.ID.Value)
	schemeID := "urn:oioubl:id:taxcategoryid-1.1"
	schemeAgencyID := "320"
	tc.ID.SchemeID = &schemeID
	tc.ID.SchemeAgencyID = &schemeAgencyID
	applyOIOUBL21TaxScheme(tc.TaxScheme)
}

func applyOIOUBL21ClassifiedTaxCategory(tc *ClassifiedTaxCategory) {
	if tc == nil {
		return
	}
	if tc.ID == nil {
		tc.ID = &IDType{Value: oioubl21TaxCategoryStandardRated}
	}
	tc.ID.Value = oioubl21TaxCategoryCode(tc.ID.Value)
	schemeID := "urn:oioubl:id:taxcategoryid-1.1"
	schemeAgencyID := "320"
	tc.ID.SchemeID = &schemeID
	tc.ID.SchemeAgencyID = &schemeAgencyID
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

func oioubl21TaxCategoryCode(in string) string {
	switch in {
	case "S", "Standard", "standard":
		return oioubl21TaxCategoryStandardRated
	case "Z", "Zero", "zero":
		return oioubl21TaxCategoryZeroRated
	case "AE", "ReverseCharge":
		return oioubl21TaxCategoryReverseCharge
	default:
		if in == "" {
			return oioubl21TaxCategoryStandardRated
		}
		return in
	}
}

// goblTaxSchemeCategory maps an OIOUBL TaxScheme ID back to the GOBL tax
// category code. OIOUBL identifies VAT as "63" (Moms); other UBL profiles
// already carry the GOBL "VAT" code, so the value passes through unchanged.
func goblTaxSchemeCategory(schemeID string) cbc.Code {
	if schemeID == oioubl21TaxSchemeVATCode {
		return cbc.Code(TaxSchemeVAT)
	}
	return cbc.Code(schemeID)
}

// goblTaxCategoryCode maps an OIOUBL TaxCategory ID back to the UNTDID 5305
// code GOBL expects (the inverse of oioubl21TaxCategoryCode). Values from
// other profiles, which already use the UNTDID codes, pass through unchanged.
func goblTaxCategoryCode(id string) cbc.Code {
	switch id {
	case oioubl21TaxCategoryStandardRated:
		return "S"
	case oioubl21TaxCategoryZeroRated:
		return "Z"
	case oioubl21TaxCategoryReverseCharge:
		return "AE"
	default:
		return cbc.Code(id)
	}
}
