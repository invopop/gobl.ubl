package ubl

import (
	"github.com/invopop/gobl/addons/sa/zatca"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/xmldsig"
)

var (
	MimeCodeTextPlain string = "text/plain"
)

// ZATCA UBL signature constants (from ZATCA KSA-15 spec).
const (
	signatureInformationID = "urn:oasis:names:specification:ubl:signature:1"
	referenceSignatureID   = "urn:oasis:names:specification:ubl:signature:Invoice"
	signatureMethod        = "urn:oasis:names:specification:ubl:dsig:enveloped:xades"

	namespaceSIG = "urn:oasis:names:specification:ubl:schema:xsd:CommonSignatureComponents-2"
	namespaceSAC = "urn:oasis:names:specification:ubl:schema:xsd:SignatureAggregateComponents-2"
	namespaceSBC = "urn:oasis:names:specification:ubl:schema:xsd:SignatureBasicComponents-2"
)

// UBLDocumentSignatures contains the signature information block.
type ZATCAUBLDocumentSignatures struct {
	SIGNamespace         string                `xml:"xmlns:sig,attr"`
	SACNamespace         string                `xml:"xmlns:sac,attr"`
	SBCNamespace         string                `xml:"xmlns:sbc,attr"`
	SignatureInformation *SignatureInformation `xml:"sac:SignatureInformation"`
}

// SignatureInformation holds the IDs and the ds:Signature.
type SignatureInformation struct {
	ID                    string             `xml:"cbc:ID"`
	ReferencedSignatureID string             `xml:"sbc:ReferencedSignatureID"`
	Signature             *xmldsig.Signature `xml:"ds:Signature"`
}

// applyZATCA applies ZATCA-specific fields to a UBL invoice.
func applyZATCA(out *Invoice, inv *bill.Invoice) {
	out.SchemaLocation = ""

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

// SetSignature sets the Signature as an UBLExtension
func (inv *Invoice) SetSignature(sig *xmldsig.Signature) {
	extURI := signatureMethod
	inv.UBLExtensions = &Extensions{
		UBLExtension: []UBLExtension{
			{
				ExtensionURI: &extURI,
				ExtensionContent: &ExtensionContent{
					ZATCAUBLDocumentSignatures: &ZATCAUBLDocumentSignatures{
						SIGNamespace: namespaceSIG,
						SACNamespace: namespaceSAC,
						SBCNamespace: namespaceSBC,
						SignatureInformation: &SignatureInformation{
							ID:                    signatureInformationID,
							ReferencedSignatureID: referenceSignatureID,
							Signature:             sig,
						},
					},
				},
			},
		},
	}

	// Add the cac:Signature reference
	sm := signatureMethod
	inv.Signature = append(inv.Signature, Signature{
		ID:              referenceSignatureID,
		SignatureMethod: &sm,
	})
}

func formatTime(t cal.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.String()
}

// InvoiceHash returns the document digest from the embedded ds:Signature,
// which is the SHA-256 hash of the filtered + canonicalized invoice XML.
// Returns an empty string if the signature is not set.
func (inv *Invoice) InvoiceHash() string {
	if inv.UBLExtensions == nil ||
		len(inv.UBLExtensions.UBLExtension) == 0 ||
		inv.UBLExtensions.UBLExtension[0].ExtensionContent == nil ||
		inv.UBLExtensions.UBLExtension[0].ExtensionContent.ZATCAUBLDocumentSignatures == nil ||
		inv.UBLExtensions.UBLExtension[0].ExtensionContent.ZATCAUBLDocumentSignatures.SignatureInformation == nil ||
		inv.UBLExtensions.UBLExtension[0].ExtensionContent.ZATCAUBLDocumentSignatures.SignatureInformation.Signature == nil ||
		inv.UBLExtensions.UBLExtension[0].ExtensionContent.ZATCAUBLDocumentSignatures.SignatureInformation.Signature.SignedInfo == nil ||
		len(inv.UBLExtensions.UBLExtension[0].ExtensionContent.ZATCAUBLDocumentSignatures.SignatureInformation.Signature.SignedInfo.Reference) == 0 {
		return ""
	}
	return inv.UBLExtensions.UBLExtension[0].ExtensionContent.ZATCAUBLDocumentSignatures.SignatureInformation.Signature.SignedInfo.Reference[0].DigestValue
}
