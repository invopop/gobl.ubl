package ubl

import (
	"github.com/invopop/gobl/addons/sa/zatca"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
)

var (
	MimeCodeTextPlain string = "text/plain"
)

// applyZATCA applies ZATCA-specific fields to a UBL invoice.
func applyZATCA(out *Invoice, inv *bill.Invoice) {

	// KSA-1
	out.UUID = string(inv.UUID)

	// KSA-25
	out.IssueTime = formatTime(*inv.IssueTime)

	// KSA-2
	if out.InvoiceTypeCode != nil {
		if invType := inv.Tax.GetExt(zatca.ExtKeyInvoiceTypeTransactions).String(); invType != "" {
			out.InvoiceTypeCode.Name = &invType
		}
	}

	// KSA-3: Assume district is mapped to StreetExtra in gobl
	moveStreetExtraToDistrict(out.AccountingSupplierParty.Party)
	moveStreetExtraToDistrict(out.AccountingCustomerParty.Party)

	stripVATCountryPrefix(out.AccountingSupplierParty.Party)
	stripVATCountryPrefix(out.AccountingCustomerParty.Party)

	// BR-KSA-17
	if inv.Preceding != nil {
		var reasons []string
		for _, ref := range inv.Preceding {
			if ref.Reason != "" {
				reasons = append(reasons, ref.Reason)
			}
		}
		out.PaymentMeans[0].InstructionNote = append(out.PaymentMeans[0].InstructionNote, reasons...)
	}

	// At the price level (cac:Price/cac:AllowanceCharge), the ChargeIndicator
	// must always be false. This structure represents the discount applied to
	// the gross price (BT-148) to derive the net price (BT-146), as per
	// BR-KSA-EN16931-07: Net price = Gross price - Allowance amount (BT-147).
	// Charges at the price level are not used in the ZATCA e-invoicing model;
	// surcharges should be applied at the line level instead.
	for _, line := range out.InvoiceLines {
		if line.Price != nil && line.Price.AllowanceCharge != nil {
			line.Price.AllowanceCharge.ChargeIndicator = false
		}
	}

	// BR-KSA-EN16931-09
	if out.TaxCurrencyCode != "" {
		out.TaxTotal = append(out.TaxTotal, TaxTotal{
			TaxAmount: out.TaxTotal[0].TaxAmount,
		})
	}
}

func stripVATCountryPrefix(p *Party) {
	if p == nil {
		return
	}
	for i := range p.PartyTaxScheme {
		id := p.PartyTaxScheme[i].CompanyID
		if id != nil && len(*id) > 2 {
			stripped := (*id)[2:]
			p.PartyTaxScheme[i].CompanyID = &stripped
		}
	}
}

func moveStreetExtraToDistrict(p *Party) {
	if p == nil || p.PostalAddress == nil {
		return
	}
	addr := p.PostalAddress
	if addr.AdditionalStreetName != nil && addr.CitySubdivisionName == nil {
		addr.CitySubdivisionName = addr.AdditionalStreetName
		addr.AdditionalStreetName = nil
	}
}

// SetICV sets the Invoice Counter Value as an AdditionalDocumentReference
func (inv *Invoice) SetICV(value string) {
	inv.AdditionalDocumentReference = append(inv.AdditionalDocumentReference, Reference{
		ID:   IDType{Value: "ICV"},
		UUID: value,
	})
}

// SetPIH sets the Previous Invoice Hash as an AdditionalDocumentReference
func (inv *Invoice) SetPIH(value string) {
	inv.AdditionalDocumentReference = append(inv.AdditionalDocumentReference, Reference{
		ID: IDType{Value: "PIH"},
		Attachment: &Attachment{
			EmbeddedDocumentBinaryObject: &BinaryObject{
				MimeCode: &MimeCodeTextPlain,
				Value:    value,
			},
		},
	})
}

// SetQRCode sets the QR code as an AdditionalDocumentReference
func (inv *Invoice) SetQRCode(value string) {
	inv.AdditionalDocumentReference = append(inv.AdditionalDocumentReference, Reference{
		ID: IDType{Value: "QR"},
		Attachment: &Attachment{
			EmbeddedDocumentBinaryObject: &BinaryObject{
				MimeCode: &MimeCodeTextPlain,
				Value:    value,
			},
		},
	})
}

func formatTime(t cal.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.String()
}
