package ubl

import (
	"encoding/xml"

	"github.com/invopop/gobl/addons/sa/zatca"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/xmldsig"
)

var (
	mimeCodeTextPlain = "text/plain"
)

// ZATCA UBL signature constants.
const (
	signatureInformationID = "urn:oasis:names:specification:ubl:signature:1"
	referenceSignatureID   = "urn:oasis:names:specification:ubl:signature:Invoice"
	signatureMethod        = "urn:oasis:names:specification:ubl:dsig:enveloped:xades"

	namespaceSIG = "urn:oasis:names:specification:ubl:schema:xsd:CommonSignatureComponents-2"
	namespaceSAC = "urn:oasis:names:specification:ubl:schema:xsd:SignatureAggregateComponents-2"
	namespaceSBC = "urn:oasis:names:specification:ubl:schema:xsd:SignatureBasicComponents-2"
)

// ZATCAUBLDocumentSignatures contains the signature information block.
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

func applyZATCA(out *Invoice, inv *bill.Invoice) {
	out.SchemaLocation = ""

	// KSA-1
	out.UUID = string(inv.UUID)

	// KSA-25
	out.IssueTime = inv.IssueTime.String()

	// BR-KSA-68
	out.TaxCurrencyCode = string(inv.RegimeDef().Currency)

	// KSA-2
	if out.InvoiceTypeCode != nil {
		if invType := inv.Tax.GetExt(zatca.ExtKeyInvoiceTypeTransactions).String(); invType != "" {
			out.InvoiceTypeCode.Name = &invType
		}
	}

	// ZATCA treats all documents as invoices
	if out.CreditNoteTypeCode != "" {
		out.XMLName = xml.Name{Local: "Invoice"}
		out.UBLNamespace = NamespaceUBLInvoice
		out.SchemaLocation = SchemaLocationInvoice

		// BR-KSA-05
		if invType := inv.Tax.GetExt(zatca.ExtKeyInvoiceTypeTransactions).String(); invType != "" {
			out.InvoiceTypeCode = &IDType{
				Value: out.CreditNoteTypeCode,
				Name:  &invType,
			}
		}

		if len(out.CreditNoteLines) > 0 {
			out.InvoiceLines = []InvoiceLine{}
			for _, line := range out.CreditNoteLines {
				out.InvoiceLines = append(out.InvoiceLines, InvoiceLine{
					ID:                  line.ID,
					Note:                line.Note,
					InvoicedQuantity:    line.CreditedQuantity,
					LineExtensionAmount: line.LineExtensionAmount,
					AccountingCost:      line.AccountingCost,
					InvoicePeriod:       line.InvoicePeriod,
					OrderLineReference:  line.OrderLineReference,
					DocumentReference:   line.DocumentReference,
					AllowanceCharge:     line.AllowanceCharge,
					TaxTotal:            line.TaxTotal,
					Item:                line.Item,
					Price:               line.Price,
				})
			}
			out.CreditNoteLines = []InvoiceLine{}
		}

		out.CreditNoteTypeCode = ""
	}

	// KSA-3: District is mapped to StreetExtra in gobl
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
	// must always be false.
	// Charges at the price level are not used in the ZATCA e-invoicing model;
	// surcharges should be applied at the line level instead.
	for _, line := range out.InvoiceLines {
		if line.Price != nil && line.Price.AllowanceCharge != nil {
			line.Price.AllowanceCharge.ChargeIndicator = false
		}
	}

	// BR-KSA-EN16931-09
	if out.TaxCurrencyCode != "" && out.DocumentCurrencyCode == out.TaxCurrencyCode {
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
func (ui *Invoice) SetICV(value string) {
	ui.AdditionalDocumentReference = append(ui.AdditionalDocumentReference, Reference{
		ID:   IDType{Value: "ICV"},
		UUID: value,
	})
}

// SetPIH sets the Previous Invoice Hash as an AdditionalDocumentReference
func (ui *Invoice) SetPIH(value string) {
	ui.AdditionalDocumentReference = append(ui.AdditionalDocumentReference, Reference{
		ID: IDType{Value: "PIH"},
		Attachment: &Attachment{
			EmbeddedDocumentBinaryObject: &BinaryObject{
				MimeCode: &mimeCodeTextPlain,
				Value:    value,
			},
		},
	})
}

// SetQRCode sets the QR code as an AdditionalDocumentReference
func (ui *Invoice) SetQRCode(value string) {
	ui.AdditionalDocumentReference = append(ui.AdditionalDocumentReference, Reference{
		ID: IDType{Value: "QR"},
		Attachment: &Attachment{
			EmbeddedDocumentBinaryObject: &BinaryObject{
				MimeCode: &mimeCodeTextPlain,
				Value:    value,
			},
		},
	})
}

// SetSignature sets the Signature as an UBLExtension
func (ui *Invoice) SetSignature(sig *xmldsig.Signature) {
	extURI := signatureMethod
	ui.Extensions = &Extensions{
		Extension: []Extension{
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
	ui.Signature = append(ui.Signature, Signature{
		ID:              referenceSignatureID,
		SignatureMethod: &sm,
	})
}
