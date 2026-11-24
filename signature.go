package ubl

// UBL signature constants.
const (
	SignatureInformationID = "urn:oasis:names:specification:ubl:signature:1"
	ReferenceSignatureID   = "urn:oasis:names:specification:ubl:signature:Invoice"
	SignatureMethod        = "urn:oasis:names:specification:ubl:dsig:enveloped:xades"
)

// AddSignatureReference adds a reference to a signature
func (ui *Invoice) AddSignatureReference(signatureMethod, referenceSignatureID string) {
	ui.Signature = append(ui.Signature, Signature{
		ID:              referenceSignatureID,
		SignatureMethod: &signatureMethod,
	})
}
