// Package ubl helps convert GOBL into UBL documents and vice versa.
package ubl

import (
	"fmt"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
)

// ParseInvoice parses a raw UBL Invoice and converts to a GOBL envelope
func ParseInvoice(ublDoc []byte) (*gobl.Envelope, error) {
	env := gobl.NewEnvelope()
	inv, err := parseInvoice(ublDoc)
	if err != nil {
		return nil, err
	}
	if err := env.Insert(inv); err != nil {
		return nil, err
	}
	return env, nil
}

// ConvertInvoice takes a GOBL envelope and converts to a UBL Invoice or Credit Note.
func ConvertInvoice(env *gobl.Envelope) (*Invoice, error) {
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, fmt.Errorf("expected bill.Inboice, got %T", env.Document)
	}
	return newInvoice(inv)
}
