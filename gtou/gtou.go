// Package gtou provides a conversor from GOBL to UBL.
package gtou

import (
	"encoding/xml"

	"github.com/invopop/gobl"
)

// Conversor is a struct that contains the necessary elements to convert between GOBL and UBL
type Conversor struct {
	doc *Document
}

// NewConversor creates a new Conversor instance
func NewConversor() *Conversor {
	c := new(Conversor)
	c.doc = new(Document)
	return c
}

// GetDocument returns the document from the conversor
func (c *Conversor) GetDocument() *Document {
	return c.doc
}

// ConvertToUBL converts a GOBL envelope into a UBL document
func (c *Conversor) ConvertToUBL(env *gobl.Envelope) (*Document, error) {

	// inv, ok := env.Extract().(*bill.Invoice)
	// if !ok {
	// 	return nil, fmt.Errorf("invalid type %T", env.Document)
	// }

	// transaction, err := NewTransaction(inv)
	// if err != nil {
	// 	return nil, err
	// }

	doc := Document{
		CACNamespace:    CAC,
		CBCNamespace:    CBC,
		UBLNamespace:    UBL,
		CustomizationID: "urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100",
		ProfileID:       "urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100",
		// Transaction:     transaction,
	}
	return &doc, nil
}

// Bytes returns the XML representation of the document in bytes
func (d *Document) Bytes() ([]byte, error) {
	bytes, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), bytes...), nil
}
