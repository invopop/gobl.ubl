// Package ubl helps convert GOBL into UBL documents and vice versa.
package ubl

import (
	"github.com/invopop/gobl"
	gtou "github.com/invopop/gobl.ubl/gtou"
	utog "github.com/invopop/gobl.ubl/utog"
)

// Converter is a struct that encapsulates both CtoG and GtoC converters
type Converter struct {
	UtoG *utog.Converter
	GtoU *gtou.Converter
}

// NewConverter creates a new Converter instance
func NewConverter() *Converter {
	c := new(Converter)
	c.UtoG = utog.NewConverter()
	c.GtoU = gtou.NewConverter()
	return c
}

// ConvertToGOBL converts a UBL document to a GOBL envelope
func (c *Converter) ConvertToGOBL(ublDoc []byte) (*gobl.Envelope, error) {
	return c.UtoG.ConvertToGOBL(ublDoc)
}

// ConvertToUBL converts a GOBL envelope to a UBL document
func (c *Converter) ConvertToUBL(env *gobl.Envelope) (*gtou.Document, error) {
	ublDoc, err := c.GtoU.ConvertToUBL(env)
	if err != nil {
		return nil, err
	}
	return ublDoc, nil
}
