package ubl

import "strings"

// TaxDataDocument is the extract of the reported invoice or credit note (TDG-02).
type TaxDataDocument struct {
	CustomizationID         string              `xml:"cbc:CustomizationID"`
	ProfileID               string              `xml:"cbc:ProfileID"`
	ID                      string              `xml:"cbc:ID"`
	UUID                    string              `xml:"cbc:UUID"`
	IssueDate               string              `xml:"cbc:IssueDate"`
	IssueTime               string              `xml:"cbc:IssueTime,omitempty"`
	DocumentTypeCode        string              `xml:"pxs:DocumentTypeCode"`
	Note                    string              `xml:"cbc:Note,omitempty"`
	TaxPointDate            string              `xml:"cbc:TaxPointDate,omitempty"`
	DocumentCurrencyCode    string              `xml:"cbc:DocumentCurrencyCode"`
	TaxCurrencyCode         string              `xml:"cbc:TaxCurrencyCode,omitempty"`
	InvoicePeriod           *Period             `xml:"cac:InvoicePeriod,omitempty"`
	BillingReference        []*BillingReference `xml:"cac:BillingReference,omitempty"`
	AccountingSupplierParty SupplierParty       `xml:"cac:AccountingSupplierParty"`
	AccountingCustomerParty CustomerParty       `xml:"cac:AccountingCustomerParty"`
	TaxRepresentativeParty  *Party              `xml:"cac:TaxRepresentativeParty,omitempty"`
	Delivery                *Delivery           `xml:"cac:Delivery,omitempty"`
	PaymentMeans            []PaymentMeans      `xml:"cac:PaymentMeans,omitempty"`
	AllowanceCharge         []AllowanceCharge   `xml:"cac:AllowanceCharge,omitempty"`
	TaxTotal                []TaxTotal          `xml:"cac:TaxTotal,omitempty"`
	MonetaryTotal           MonetaryTotal       `xml:"pxs:MonetaryTotal"`
	DocumentLines           []TaxDataLine       `xml:"pxs:DocumentLine"`
}

// TaxDataLine is one line of the reported document (BG-25).
type TaxDataLine struct {
	ID                  string             `xml:"cbc:ID"`
	Note                string             `xml:"cbc:Note,omitempty"`
	InvoicedQuantity    *Quantity          `xml:"cbc:InvoicedQuantity,omitempty"`
	LineExtensionAmount Amount             `xml:"cbc:LineExtensionAmount"`
	InvoicePeriod       *Period            `xml:"cac:InvoicePeriod,omitempty"`
	AllowanceCharge     []*AllowanceCharge `xml:"cac:AllowanceCharge,omitempty"`
	Item                *Item              `xml:"cac:Item,omitempty"`
	Price               *Price             `xml:"cac:Price,omitempty"`
}

// taxDataDocument copies the parts of the invoice that a TDD carries, and
// nothing else, as the TDD Schematron rejects any other element.
func taxDataDocument(inv *Invoice, profile TaxDataProfile) TaxDataDocument {
	doc := TaxDataDocument{
		CustomizationID:      inv.CustomizationID,
		ID:                   inv.ID,
		IssueDate:            inv.IssueDate,
		IssueTime:            inv.IssueTime,
		TaxPointDate:         inv.TaxPointDate,
		DocumentCurrencyCode: inv.DocumentCurrencyCode,
		TaxCurrencyCode:      inv.TaxCurrencyCode,
		AccountingSupplierParty: SupplierParty{
			Party: taxDataParty(inv.AccountingSupplierParty.Party, false),
		},
		AccountingCustomerParty: CustomerParty{
			Party: taxDataParty(inv.AccountingCustomerParty.Party, profile.BuyerName),
		},
		MonetaryTotal: inv.LegalMonetaryTotal,
	}
	if inv.ProfileID != nil {
		doc.ProfileID = inv.ProfileID.Value
	}
	if tc := typeCodeOf(inv); tc != "" {
		doc.DocumentTypeCode = tc
	}
	// The TDD allows a single note.
	if len(inv.Note) > 0 {
		doc.Note = strings.Join(inv.Note, "\n\n")
	}
	if len(inv.InvoicePeriod) > 0 {
		p := inv.InvoicePeriod[0]
		doc.InvoicePeriod = &p
	}
	for _, br := range inv.BillingReference {
		if br == nil || br.InvoiceDocumentReference == nil {
			continue
		}
		doc.BillingReference = append(doc.BillingReference, &BillingReference{
			InvoiceDocumentReference: &Reference{
				ID:        IDType{Value: br.InvoiceDocumentReference.ID.Value},
				IssueDate: br.InvoiceDocumentReference.IssueDate,
			},
		})
	}
	if inv.TaxRepresentativeParty != nil {
		doc.TaxRepresentativeParty = taxDataParty(inv.TaxRepresentativeParty, false)
	}
	for _, d := range inv.Delivery {
		if d != nil && d.ActualDeliveryDate != nil {
			doc.Delivery = &Delivery{ActualDeliveryDate: d.ActualDeliveryDate}
			break
		}
	}
	for _, pm := range inv.PaymentMeans {
		doc.PaymentMeans = append(doc.PaymentMeans, taxDataPaymentMeans(pm))
	}
	for _, ac := range inv.AllowanceCharge {
		ac := ac
		ac.TaxCategory = taxDataCategories(ac.TaxCategory)
		doc.AllowanceCharge = append(doc.AllowanceCharge, ac)
	}
	for _, tt := range inv.TaxTotal {
		doc.TaxTotal = append(doc.TaxTotal, taxDataTaxTotal(tt))
	}
	lines := inv.InvoiceLines
	if len(inv.CreditNoteLines) > 0 {
		lines = inv.CreditNoteLines
	}
	for _, l := range lines {
		doc.DocumentLines = append(doc.DocumentLines, taxDataLine(l))
	}
	return doc
}

func typeCodeOf(inv *Invoice) string {
	if inv.InvoiceTypeCode != nil {
		return inv.InvoiceTypeCode.Value
	}
	if inv.CreditNoteTypeCode != nil {
		return inv.CreditNoteTypeCode.Value
	}
	return ""
}

func partyEndpoint(p *Party) *EndpointID {
	if p == nil || p.EndpointID == nil || p.EndpointID.Value == "" {
		return nil
	}
	e := *p.EndpointID
	return &e
}

// taxDataParty keeps the country, the VAT identifier and, when asked, the
// registration name.
func taxDataParty(p *Party, withName bool) *Party {
	out := new(Party)
	if p == nil {
		return out
	}
	if p.PostalAddress != nil && p.PostalAddress.Country != nil {
		out.PostalAddress = &PostalAddress{Country: &Country{IdentificationCode: p.PostalAddress.Country.IdentificationCode}}
	}
	if pts := vatTaxScheme(p); pts != nil {
		out.PartyTaxScheme = []PartyTaxScheme{{
			CompanyID: &IDType{Value: pts.CompanyID.Value},
			TaxScheme: &TaxScheme{ID: IDType{Value: "VAT"}},
		}}
	}
	if withName && p.PartyLegalEntity != nil && p.PartyLegalEntity.RegistrationName != nil {
		out.PartyLegalEntity = &PartyLegalEntity{RegistrationName: p.PartyLegalEntity.RegistrationName}
	}
	return out
}

// vatTaxScheme provides the party's VAT registration, if any.
func vatTaxScheme(p *Party) *PartyTaxScheme {
	if p == nil {
		return nil
	}
	for i := range p.PartyTaxScheme {
		pts := &p.PartyTaxScheme[i]
		if pts.CompanyID != nil && pts.TaxScheme != nil && pts.TaxScheme.ID.Value == "VAT" {
			return pts
		}
	}
	return nil
}

func taxDataPaymentMeans(pm PaymentMeans) PaymentMeans {
	out := PaymentMeans{
		PaymentMeansCode: pm.PaymentMeansCode,
		PaymentID:        pm.PaymentID,
	}
	if pm.CardAccount != nil {
		out.CardAccount = &CardAccount{
			PrimaryAccountNumberID: pm.CardAccount.PrimaryAccountNumberID,
			NetworkID:              pm.CardAccount.NetworkID,
			HolderName:             pm.CardAccount.HolderName,
		}
	}
	if fa := pm.PayeeFinancialAccount; fa != nil {
		out.PayeeFinancialAccount = &FinancialAccount{ID: fa.ID}
		if fa.FinancialInstitutionBranch != nil && fa.FinancialInstitutionBranch.ID != nil {
			out.PayeeFinancialAccount.FinancialInstitutionBranch = &Branch{ID: fa.FinancialInstitutionBranch.ID}
		}
	}
	if m := pm.PaymentMandate; m != nil {
		out.PaymentMandate = &PaymentMandate{ID: m.ID}
		if m.PayerFinancialAccount != nil {
			out.PaymentMandate.PayerFinancialAccount = &FinancialAccount{ID: m.PayerFinancialAccount.ID}
		}
	}
	return out
}

// taxDataCategory keeps the identifier, the rate and the scheme, and the
// exemption reason when asked.
func taxDataCategory(c *TaxCategory, withExemption bool) *TaxCategory {
	if c == nil {
		return nil
	}
	out := &TaxCategory{ID: c.ID, Percent: c.Percent}
	if withExemption {
		out.TaxExemptionReasonCode = c.TaxExemptionReasonCode
		out.TaxExemptionReason = c.TaxExemptionReason
	}
	if c.TaxScheme != nil {
		out.TaxScheme = &TaxScheme{ID: IDType{Value: c.TaxScheme.ID.Value}}
	}
	return out
}

func taxDataCategories(list []*TaxCategory) []*TaxCategory {
	var out []*TaxCategory
	for _, c := range list {
		if c != nil {
			out = append(out, taxDataCategory(c, false))
		}
	}
	return out
}

func taxDataTaxTotal(tt TaxTotal) TaxTotal {
	out := TaxTotal{TaxAmount: tt.TaxAmount}
	for _, st := range tt.TaxSubtotal {
		out.TaxSubtotal = append(out.TaxSubtotal, TaxSubtotal{
			TaxableAmount: st.TaxableAmount,
			TaxAmount:     st.TaxAmount,
			TaxCategory:   *taxDataCategory(&st.TaxCategory, true),
		})
	}
	return out
}

func taxDataLine(l InvoiceLine) TaxDataLine {
	out := TaxDataLine{
		ID:                  l.ID,
		InvoicedQuantity:    l.InvoicedQuantity,
		LineExtensionAmount: l.LineExtensionAmount,
	}
	if out.InvoicedQuantity == nil {
		// Credit notes carry the credited quantity, reported as invoiced.
		out.InvoicedQuantity = l.CreditedQuantity
	}
	if len(l.Note) > 0 {
		out.Note = strings.Join(l.Note, "\n\n")
	}
	if l.InvoicePeriod != nil {
		out.InvoicePeriod = &Period{StartDate: l.InvoicePeriod.StartDate, EndDate: l.InvoicePeriod.EndDate}
	}
	for _, ac := range l.AllowanceCharge {
		if ac == nil {
			continue
		}
		c := *ac
		// Line allowances and charges carry no tax category in the TDD.
		c.TaxCategory = nil
		out.AllowanceCharge = append(out.AllowanceCharge, &c)
	}
	if it := l.Item; it != nil {
		out.Item = &Item{
			Description:             it.Description,
			Name:                    it.Name,
			CommodityClassification: it.CommodityClassification,
		}
		if ct := it.ClassifiedTaxCategory; ct != nil {
			out.Item.ClassifiedTaxCategory = &ClassifiedTaxCategory{ID: ct.ID, Percent: ct.Percent}
			if ct.TaxScheme != nil {
				out.Item.ClassifiedTaxCategory.TaxScheme = &TaxScheme{ID: IDType{Value: ct.TaxScheme.ID.Value}}
			}
		}
	}
	if p := l.Price; p != nil {
		out.Price = &Price{PriceAmount: p.PriceAmount, BaseQuantity: p.BaseQuantity}
	}
	return out
}
