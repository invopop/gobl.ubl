package ubl

import (
	"encoding/xml"

	"github.com/invopop/xmldsig"
)

// UBL signature constants.
const (
	SignatureInformationID = "urn:oasis:names:specification:ubl:signature:1"
	ReferenceSignatureID   = "urn:oasis:names:specification:ubl:signature:Invoice"
	SignatureMethod        = "urn:oasis:names:specification:ubl:dsig:enveloped:xades"
)

// prefixCBC is the conventional prefix of the common basic components.
const prefixCBC = "cbc"

func init() {
	RegisterExtensionData(xml.Name{Space: NamespaceSIG, Local: "UBLDocumentSignatures"}, func() ExtensionData { return new(DocumentSignatures) })
}

// DocumentSignatures contains the signature information block.
type DocumentSignatures struct {
	XMLName              xml.Name              `xml:"sig:UBLDocumentSignatures"`
	SIGNamespace         string                `xml:"xmlns:sig,attr"`
	SACNamespace         string                `xml:"xmlns:sac,attr"`
	SBCNamespace         string                `xml:"xmlns:sbc,attr"`
	SignatureInformation *SignatureInformation `xml:"sac:SignatureInformation"`
}

// ExtensionNamespaces implements ExtensionData.
func (*DocumentSignatures) ExtensionNamespaces() map[string]string {
	return map[string]string{
		"sig":     NamespaceSIG,
		"sac":     NamespaceSAC,
		"sbc":     NamespaceSBC,
		prefixCBC: NamespaceCBC,
		"ds":      xmldsig.NamespaceDSig,
		"xades":   xmldsig.NamespaceXAdES,
	}
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
