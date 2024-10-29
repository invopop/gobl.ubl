// Package ubl helps convert GOBL into UBL documents and vice versa.
package ubl

import (
	"github.com/invopop/gobl"
	gtou "github.com/invopop/gobl.ubl/gtou"
	utog "github.com/invopop/gobl.ubl/utog"
)

// Conversor is a struct that encapsulates both CtoG and GtoC conversors
type Conversor struct {
	UtoG *utog.Conversor
	GtoU *gtou.Converter
}

// NewConversor creates a new Conversor instance
func NewConversor() *Conversor {
	c := new(Conversor)
	c.UtoG = utog.NewConversor()
	c.GtoU = gtou.NewConverter()
	return c
}

// ConvertToGOBL converts a UBL document to a GOBL envelope
func (c *Conversor) ConvertToGOBL(ublDoc []byte) (*gobl.Envelope, error) {
	return c.UtoG.ConvertToGOBL(ublDoc)
}

// ConvertToUBL converts a GOBL envelope to a UBL document
func (c *Conversor) ConvertToUBL(env *gobl.Envelope) (*gtou.Document, error) {
	ublDoc, err := c.GtoU.ConvertToUBL(env)
	if err != nil {
		return nil, err
	}
	return ublDoc, nil
}
