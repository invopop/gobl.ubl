package ubl

import (
	"encoding/xml"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// tokenList is an xml.TokenReader that gives its tokens in order.
type tokenList []xml.Token

func (l *tokenList) Token() (xml.Token, error) {
	if len(*l) == 0 {
		return nil, io.EOF
	}
	tok := (*l)[0]
	*l = (*l)[1:]
	return tok, nil
}

func TestExtensionContentUnmarshalXML(t *testing.T) {
	t.Run("kept tokens the encoder rejects are an error", func(t *testing.T) {
		// the decoder accepts an element with no name, the encoder does not
		toks := tokenList{
			xml.StartElement{Name: xml.Name{Local: "ExtensionContent"}},
			xml.StartElement{},
			xml.EndElement{},
			xml.EndElement{Name: xml.Name{Local: "ExtensionContent"}},
		}
		err := xml.NewTokenDecoder(&toks).Decode(new(ExtensionContent))
		assert.ErrorContains(t, err, "start tag with no name")
	})
}

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
