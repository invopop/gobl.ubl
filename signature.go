package ubl

import "github.com/invopop/xmldsig"

// UBL signature constants.
const (
	SignatureInformationID = "urn:oasis:names:specification:ubl:signature:1"
	ReferenceSignatureID   = "urn:oasis:names:specification:ubl:signature:Invoice"
	SignatureMethod        = "urn:oasis:names:specification:ubl:dsig:enveloped:xades"
)

// UBLDocumentSignatures contains the signature information block.
type UBLDocumentSignatures struct {
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

// AddSignatureReference adds a reference to a signature
func (ui *Invoice) AddSignatureReference(signatureMethod, referenceSignatureID string) {
	ui.Signature = append(ui.Signature, Signature{
		ID:              referenceSignatureID,
		SignatureMethod: &signatureMethod,
	})
}
