package document

import "encoding/xml"

// Bytes returns the XML representation of the document in bytes
func (d *Invoice) Bytes() ([]byte, error) {
	bytes, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), bytes...), nil
}
