package ubl

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeTokens(t *testing.T) {
	t.Run("tokens the encoder rejects are an error", func(t *testing.T) {
		_, err := encodeTokens([]xml.Token{xml.EndElement{Name: xml.Name{Local: "a"}}})
		assert.ErrorContains(t, err, "end tag </a> without start tag")
	})
}

func TestDecodeExtensionData(t *testing.T) {
	t.Run("tokens the encoder rejects are an error", func(t *testing.T) {
		toks := []xml.Token{
			xml.StartElement{Name: xml.Name{Local: "a"}},
			xml.EndElement{Name: xml.Name{Local: "b"}},
		}
		err := decodeExtensionData(new(DocumentSignatures), toks)
		assert.ErrorContains(t, err, "end tag </b> does not match start tag <a>")
	})
}
