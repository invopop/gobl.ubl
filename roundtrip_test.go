package ubl_test

import (
	"bytes"
	"encoding/xml"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ubl "github.com/invopop/gobl.ubl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// applicationResponseXML declares cbc on the root and cac on the element
// that uses it, so the root has no cac declaration.
const applicationResponseXML = `<?xml version="1.0" encoding="UTF-8"?>
<ApplicationResponse xmlns="urn:oasis:names:specification:ubl:schema:xsd:ApplicationResponse-2"
  xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">
  <cbc:ID>AR-1</cbc:ID>
  <cbc:IssueDate>2026-11-23</cbc:IssueDate>
  <cac:SenderParty xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2">
    <cac:PartyName><cbc:Name>Sender</cbc:Name></cac:PartyName>
  </cac:SenderParty>
</ApplicationResponse>`

func TestDecodeEncodeRoundTrip(t *testing.T) {
	docs := map[string][]byte{"inline/application-response.xml": []byte(applicationResponseXML)}
	err := filepath.WalkDir(getDataPath(), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".xml" {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(getDataPath(), path)
		docs[rel] = data
		return nil
	})
	require.NoError(t, err)

	for name, data := range docs {
		t.Run(name, func(t *testing.T) {
			first, err := ubl.Decode(data)
			require.NoError(t, err)
			out, err := ubl.Encode(first)
			require.NoError(t, err)

			assertValidRoot(t, out)
			assertPrefixesBound(t, out)

			second, err := ubl.Decode(out)
			require.NoError(t, err)
			assert.Equal(t, first, second, "decoding the output gives the same document")

			again, err := ubl.Encode(second)
			require.NoError(t, err)
			assert.Equal(t, string(out), string(again), "encoding is stable")
		})
	}
}

// assertValidRoot checks that the root element repeats no attribute and
// declares no prefix as empty, which XML 1.0 forbids.
func assertValidRoot(t *testing.T, data []byte) {
	t.Helper()
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := d.RawToken()
		require.NoError(t, err)
		root, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		seen := make(map[xml.Name]bool)
		for _, a := range root.Attr {
			assert.False(t, seen[a.Name], "attribute %s:%s is repeated", a.Name.Space, a.Name.Local)
			seen[a.Name] = true
			if a.Name.Space == "xmlns" {
				assert.NotEmpty(t, a.Value, "xmlns:%s is declared empty", a.Name.Local)
			}
		}
		return
	}
}

// assertPrefixesBound checks that every element prefix is declared. The
// decoder leaves an undeclared prefix as the name's space instead of a URI.
func assertPrefixesBound(t *testing.T, data []byte) {
	t.Helper()
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return
		}
		require.NoError(t, err)
		if el, ok := tok.(xml.StartElement); ok {
			require.True(t, strings.Contains(el.Name.Space, ":"), "prefix %q of <%s> is not declared", el.Name.Space, el.Name.Local)
		}
	}
}
