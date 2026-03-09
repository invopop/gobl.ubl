package ubl

import "strings"

func applyLegacyOIOUBL21Rules(out *Invoice) {
	if out == nil {
		return
	}

	applyLegacyOIOUBL21Party(out.AccountingSupplierParty.Party)
	applyLegacyOIOUBL21Party(out.AccountingCustomerParty.Party)

	for i := range out.PaymentMeans {
		pm := &out.PaymentMeans[i]
		if pm.PaymentChannelCode == nil {
			pm.PaymentChannelCode = &IDType{Value: "IBAN"}
		}
		listID := "urn:oioubl:codelist:paymentchannelcode-1.1"
		pm.PaymentChannelCode.ListID = &listID
		if pm.PaymentChannelCode.Value == "IBAN" && pm.PayeeFinancialAccount != nil && pm.PayeeFinancialAccount.FinancialInstitutionBranch != nil {
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

	// Legacy validator expects this relationship in the 2.1 profile.
	if len(out.TaxTotal) > 0 {
		out.LegalMonetaryTotal.TaxExclusiveAmount = out.TaxTotal[0].TaxAmount
	}
	if out.LegalMonetaryTotal.PayableAmount != nil && len(out.PaymentTerms) > 0 && out.PaymentTerms[0].Amount == nil {
		out.PaymentTerms[0].Amount = out.LegalMonetaryTotal.PayableAmount
	}
	if out.CreditNoteTypeCode != "" {
		for i := range out.BillingReference {
			if ref := out.BillingReference[i]; ref != nil && ref.InvoiceDocumentReference != nil {
				// Legacy OIOUBL 2.1 credit-note schematron rejects DocumentTypeCode here.
				ref.InvoiceDocumentReference.DocumentTypeCode = ""
			}
		}
	}

	for i := range out.TaxTotal {
		for j := range out.TaxTotal[i].TaxSubtotal {
			applyLegacyOIOUBL21TaxCategory(&out.TaxTotal[i].TaxSubtotal[j].TaxCategory)
		}
	}
	for i := range out.InvoiceLines {
		if line := &out.InvoiceLines[i]; line.Item != nil && line.Item.ClassifiedTaxCategory != nil {
			applyLegacyOIOUBL21ClassifiedTaxCategory(line.Item.ClassifiedTaxCategory)
		}
		for j := range out.InvoiceLines[i].TaxTotal {
			for k := range out.InvoiceLines[i].TaxTotal[j].TaxSubtotal {
				applyLegacyOIOUBL21TaxCategory(&out.InvoiceLines[i].TaxTotal[j].TaxSubtotal[k].TaxCategory)
			}
		}
	}
	for i := range out.CreditNoteLines {
		if line := &out.CreditNoteLines[i]; line.Item != nil && line.Item.ClassifiedTaxCategory != nil {
			applyLegacyOIOUBL21ClassifiedTaxCategory(line.Item.ClassifiedTaxCategory)
		}
		for j := range out.CreditNoteLines[i].TaxTotal {
			for k := range out.CreditNoteLines[i].TaxTotal[j].TaxSubtotal {
				applyLegacyOIOUBL21TaxCategory(&out.CreditNoteLines[i].TaxTotal[j].TaxSubtotal[k].TaxCategory)
			}
		}
	}
}

func applyLegacyOIOUBL21Party(p *Party) {
	if p == nil {
		return
	}
	if p.EndpointID != nil && p.EndpointID.SchemeID == "0088" {
		p.EndpointID.SchemeID = "GLN"
	}
	if p.EndpointID == nil {
		if len(p.PartyTaxScheme) > 0 && p.PartyTaxScheme[0].CompanyID != nil {
			val := p.PartyTaxScheme[0].CompanyID.Value
			if !strings.HasPrefix(val, "DK") {
				val = "DK" + val
			}
			p.EndpointID = &EndpointID{
				SchemeID: "DK:CVR",
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
			ListID:      &listID,
			ListAgencyID: &listAgencyID,
			Value:       "StructuredDK",
		}
	}
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
			applyLegacyOIOUBL21TaxScheme(pts.TaxScheme)
		}
	}
	if p.PartyLegalEntity != nil && p.PartyLegalEntity.CompanyID != nil {
		scheme := "DK:CVR"
		p.PartyLegalEntity.CompanyID.SchemeID = &scheme
		if !strings.HasPrefix(p.PartyLegalEntity.CompanyID.Value, "DK") {
			p.PartyLegalEntity.CompanyID.Value = "DK" + p.PartyLegalEntity.CompanyID.Value
		}
	}
	if p.Contact == nil {
		p.Contact = &Contact{}
	}
	if p.Contact.ID == nil {
		id := "1"
		p.Contact.ID = &id
	}
}

func applyLegacyOIOUBL21TaxCategory(tc *TaxCategory) {
	if tc == nil {
		return
	}
	if tc.ID == nil {
		tc.ID = &IDType{Value: "StandardRated"}
	}
	tc.ID.Value = legacyTaxCategoryCode(tc.ID.Value)
	schemeID := "urn:oioubl:id:taxcategoryid-1.1"
	schemeAgencyID := "320"
	tc.ID.SchemeID = &schemeID
	tc.ID.SchemeAgencyID = &schemeAgencyID
	applyLegacyOIOUBL21TaxScheme(tc.TaxScheme)
}

func applyLegacyOIOUBL21ClassifiedTaxCategory(tc *ClassifiedTaxCategory) {
	if tc == nil {
		return
	}
	if tc.ID == nil {
		tc.ID = &IDType{Value: "StandardRated"}
	}
	tc.ID.Value = legacyTaxCategoryCode(tc.ID.Value)
	schemeID := "urn:oioubl:id:taxcategoryid-1.1"
	schemeAgencyID := "320"
	tc.ID.SchemeID = &schemeID
	tc.ID.SchemeAgencyID = &schemeAgencyID
	applyLegacyOIOUBL21TaxScheme(tc.TaxScheme)
}

func applyLegacyOIOUBL21TaxScheme(ts *TaxScheme) {
	if ts == nil {
		return
	}
	schemeID := "urn:oioubl:id:taxschemeid-1.2"
	schemeAgencyID := "320"
	ts.ID = IDType{
		SchemeID:       &schemeID,
		SchemeAgencyID: &schemeAgencyID,
		Value:          "63",
	}
	name := "Moms"
	ts.Name = &name
}

func legacyTaxCategoryCode(in string) string {
	switch in {
	case "S", "Standard", "standard":
		return "StandardRated"
	case "Z", "Zero", "zero":
		return "ZeroRated"
	case "AE", "ReverseCharge":
		return "ReverseCharge"
	default:
		if in == "" {
			return "StandardRated"
		}
		return in
	}
}
