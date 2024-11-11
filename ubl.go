// Package ubl helps convert GOBL into UBL documents and vice versa.
package ubl

import (
	"github.com/invopop/gobl"
	"github.com/invopop/gobl.ubl/document"
	"github.com/invopop/gobl.ubl/internal/gtou"
	"github.com/invopop/gobl.ubl/internal/utog"
)

// ToGOBL converts a UBL document to a GOBL envelope
func ToGOBL(ublDoc []byte) (*gobl.Envelope, error) {
	return utog.Convert(ublDoc)
}

// ToUBL converts a GOBL envelope to a UBL document
func ToUBL(env *gobl.Envelope) (*document.Document, error) {
	return gtou.Convert(env)
}
