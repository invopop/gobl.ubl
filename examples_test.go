package ubl_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lestrrat-go/libxml2"
	"github.com/lestrrat-go/libxml2/xsd"
)

const (
	xmlPattern  = "*.xml"
	jsonPattern = "*.json"

	schemaInvoice    = "UBL-Invoice-2.1.xsd"
	schemaCreditNote = "UBL-CreditNote-2.1.xsd"

	staticUUID uuid.UUID = "0195ce71-dc9c-72c8-bf2c-9890a4a9f0a2"
)

// updateOut is a flag that can be set to update example files
var updateOut = flag.Bool("update", false, "Update the example files in test/data")

func TestConvertToInvoice(t *testing.T) {
	invoiceSchema := loadSchema(t, schemaInvoice)
	creditNoteSchema := loadSchema(t, schemaCreditNote)

	examples, err := getDataGlob(jsonPattern)
	require.NoError(t, err)

	for _, example := range examples {
		inName := filepath.Base(example)
		outName := strings.Replace(inName, ".json", ".xml", 1)

		t.Run(inName, func(t *testing.T) {
			doc, err := testInvoiceFrom(inName)
			require.NoError(t, err)

			data, err := doc.Bytes()
			require.NoError(t, err)

			if *updateOut {
				err = os.WriteFile(outputFilepath(outName), data, 0644)
				require.NoError(t, err)
				var schema *xsd.Schema
				switch doc.XMLName.Local {
				case "Invoice":
					schema = invoiceSchema
				case "CreditNote":
					schema = creditNoteSchema
				default:
					require.Fail(t, "unknown document schema")
				}
				err = ValidateXML(schema, data)
				require.NoError(t, err)
			}

			output, err := loadOutputFile(outName)
			assert.NoError(t, err)
			assert.Equal(t, string(output), string(data), "Output should match the expected XML. Update with --update flag.")
		})
	}
}

func TestParseInvoice(t *testing.T) {
	examples, err := getDataGlob(xmlPattern)
	require.NoError(t, err)

	for _, example := range examples {
		inName := filepath.Base(example)
		outName := strings.Replace(inName, ".xml", ".json", 1)

		t.Run(inName, func(t *testing.T) {
			// Load XML data
			xmlData, err := os.ReadFile(example)
			require.NoError(t, err)

			// Convert UBL XML to GOBL
			doc, err := ubl.Parse(xmlData)
			require.NoError(t, err)
			inv, ok := doc.(*ubl.Invoice)
			require.True(t, ok, "Document should be an invoice")
			env, err := inv.Convert()
			require.NoError(t, err)

			// Unfortunately, the sample UBL documents have lots of errors, including
			// missing exchange rate data and invalid Tax ID codes. We're disabling
			// validation here, but periodically it'd be good to re-enable and check
			// for any major issues.
			// require.NoError(t, env.Validate())

			env.Head.UUID = staticUUID
			if inv, ok := env.Extract().(*bill.Invoice); ok {
				inv.UUID = staticUUID
			}

			// Recalculate to ensure consistent digests
			if err = env.Calculate(); err != nil {
				require.NoError(t, err)
			}

			writeEnvelope(outputFilepath(outName), env)

			// Extract the invoice from the envelope
			invoice, ok := env.Extract().(*bill.Invoice)
			require.True(t, ok, "Document should be an invoice")

			// Marshal only the invoice
			data, err := json.MarshalIndent(invoice, "", "\t")
			require.NoError(t, err)

			// Load the expected output
			output, err := loadOutputFile(outName)
			assert.NoError(t, err)

			// Parse the expected output to extract the invoice
			var expectedEnv gobl.Envelope
			err = json.Unmarshal(output, &expectedEnv)
			require.NoError(t, err)

			expectedInvoice, ok := expectedEnv.Extract().(*bill.Invoice)
			require.True(t, ok, "Expected document should be an invoice")

			// Marshal the expected invoice
			expectedData, err := json.MarshalIndent(expectedInvoice, "", "\t")
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedData), string(data), "Invoice should match the expected JSON. Update with --update flag.")
		})
	}
}

// testInvoiceFrom creates a UBL Invoice from a GOBL file in the `test/data` folder
func testInvoiceFrom(name string) (*ubl.Invoice, error) {
	env, err := loadTestEnvelope(name)
	if err != nil {
		return nil, err
	}
	return ubl.ConvertInvoice(env)
}

// testLoadXML provides the raw data of a test XML file
func testLoadXML(name string) ([]byte, error) {
	src, err := os.Open(filepath.Join(getConversionTypePath(xmlPattern), name))
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := src.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	return io.ReadAll(src)
}

// testParseInvoice takes the provided file and converts to a
// GOBL
func testParseInvoice(name string) (*gobl.Envelope, error) {
	data, err := testLoadXML(name)
	if err != nil {
		return nil, err
	}

	doc, err := ubl.Parse(data)
	if err != nil {
		return nil, err
	}

	inv, ok := doc.(*ubl.Invoice)
	if !ok {
		return nil, fmt.Errorf("document is not an invoice")
	}
	return inv.Convert()
}

// loadTestEnvelope returns a GOBL Envelope from a file in the `test/data` folder
func loadTestEnvelope(name string) (*gobl.Envelope, error) {
	path := filepath.Join(getConversionTypePath(jsonPattern), name)
	src, _ := os.Open(path)
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(src); err != nil {
		return nil, err
	}
	env := new(gobl.Envelope)
	if err := json.Unmarshal(buf.Bytes(), env); err != nil {
		return nil, err
	}

	// Clear the IDs
	env.Head.UUID = staticUUID
	if inv, ok := env.Extract().(*bill.Invoice); ok {
		inv.UUID = staticUUID
	}

	if err := env.Calculate(); err != nil {
		panic(err)
	}

	if err := env.Validate(); err != nil {
		panic(err)
	}

	// Make an update if requested
	writeEnvelope(path, env)

	return env, nil
}

// loadOutputFile returns byte data from a file in the `test/data/out` folder
func loadOutputFile(name string) ([]byte, error) {
	src, _ := os.Open(outputFilepath(name))
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(src); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeEnvelope(path string, env *gobl.Envelope) {
	if !*updateOut {
		return
	}
	data, err := json.MarshalIndent(env, "", "\t")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		panic(err)
	}
}

func outputFilepath(name string) string {
	var pattern string
	if strings.HasSuffix(name, ".json") {
		pattern = xmlPattern
	} else {
		pattern = jsonPattern
	}
	return filepath.Join(getOutPath(pattern), name)
}

func loadSchema(t *testing.T, name string) *xsd.Schema {
	t.Helper()
	schema, err := xsd.ParseFromFile(filepath.Join(getSchemaPath(), name))
	require.NoError(t, err)
	return schema
}

// ValidateXML validates a XML document against a XSD Schema
func ValidateXML(schema *xsd.Schema, data []byte) error {
	xmlDoc, err := libxml2.Parse(data)
	if err != nil {
		return err
	}

	err = schema.Validate(xmlDoc)
	if err != nil {
		return err.(xsd.SchemaValidationError).Errors()[0]
	}

	return nil
}

func getDataGlob(pattern string) ([]string, error) {
	return filepath.Glob(filepath.Join(getConversionTypePath(pattern), pattern))
}

func getSchemaPath() string {
	return filepath.Join(getDataPath(), "schema", "maindoc")
}

func getOutPath(pattern string) string {
	return filepath.Join(getConversionTypePath(pattern), "out")
}

func getDataPath() string {
	return filepath.Join(getTestPath(), "data")
}

func getConversionTypePath(pattern string) string {
	if pattern == xmlPattern {
		return filepath.Join(getDataPath(), "parse")
	}
	return filepath.Join(getDataPath(), "convert")
}

func getTestPath() string {
	return filepath.Join(getRootFolder(), "test")
}

// TODO: adapt to new folder structure
func getRootFolder() string {
	cwd, _ := os.Getwd()

	for !isRootFolder(cwd) {
		cwd = removeLastEntry(cwd)
	}
	return cwd
}

func isRootFolder(dir string) bool {
	files, _ := os.ReadDir(dir)

	for _, file := range files {
		if file.Name() == "go.mod" {
			return true
		}
	}

	return false
}

func removeLastEntry(dir string) string {
	lastEntry := "/" + filepath.Base(dir)
	i := strings.LastIndex(dir, lastEntry)
	return dir[:i]
}
