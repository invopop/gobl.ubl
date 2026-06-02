package ubl_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/phive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const oioublResponseDir = "oioubl21-response"

func TestConvertToApplicationResponse(t *testing.T) {
	var pc phive.ValidationServiceClient
	if *validate {
		conn, err := grpc.NewClient(
			"127.0.0.1:9090",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer conn.Close() //nolint:errcheck
		pc = phive.NewValidationServiceClient(conn)
	}

	examples, err := filepath.Glob(filepath.Join(getConvertPath(), oioublResponseDir, jsonPattern))
	require.NoError(t, err)
	require.NotEmpty(t, examples, "no ApplicationResponse examples found")

	for _, example := range examples {
		inName := filepath.Base(example)
		outName := strings.Replace(inName, ".json", ".xml", 1)

		t.Run(inName, func(t *testing.T) {
			env, err := loadTestEnvelopeFromPath(example)
			require.NoError(t, err)

			doc, err := ubl.Convert(env, ubl.WithContext(ubl.ContextOIOUBL21))
			require.NoError(t, err)

			data, err := ubl.Bytes(doc)
			require.NoError(t, err)

			outPath := filepath.Join(getConvertPath(), oioublResponseDir, "out", outName)
			if *updateOut {
				require.NoError(t, os.WriteFile(outPath, data, 0644))
			}

			if *validate {
				vesid := ubl.ContextOIOUBL21.VESIDs.ApplicationResponse
				resp, err := pc.ValidateXml(context.Background(), &phive.ValidateXmlRequest{
					Vesid:      vesid,
					XmlContent: data,
				})
				require.NoError(t, err)
				results, err := json.MarshalIndent(resp.Results, "", "  ")
				require.NoError(t, err)
				require.True(t, resp.Success, "Generated XML should be valid for %s: %s", vesid, string(results))
			}

			output, err := os.ReadFile(outPath)
			assert.NoError(t, err)
			assert.Equal(t, string(output), string(data), "Output should match the expected XML. Update with --update flag.")
		})
	}
}

func TestParseApplicationResponse(t *testing.T) {
	examples, err := filepath.Glob(filepath.Join(getParsePath(), oioublResponseDir, xmlPattern))
	require.NoError(t, err)
	require.NotEmpty(t, examples, "no ApplicationResponse parse examples found")

	for _, example := range examples {
		inName := filepath.Base(example)
		outName := strings.Replace(inName, ".xml", ".json", 1)

		t.Run(inName, func(t *testing.T) {
			xmlData, err := os.ReadFile(example)
			require.NoError(t, err)

			doc, err := ubl.Parse(xmlData)
			require.NoError(t, err)
			ar, ok := doc.(*ubl.ApplicationResponse)
			require.True(t, ok, "Document should be an ApplicationResponse")

			env, err := ar.Convert()
			require.NoError(t, err)

			env.Head.UUID = staticUUID
			if st, ok := env.Extract().(*bill.Status); ok {
				st.UUID = staticUUID
			}
			require.NoError(t, env.Calculate())

			outPath := filepath.Join(getParsePath(), oioublResponseDir, "out", outName)
			if *updateOut {
				data, err := json.MarshalIndent(env, "", "\t")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(outPath, data, 0644))
			}

			status, ok := env.Extract().(*bill.Status)
			require.True(t, ok, "Document should be a status")
			data, err := json.MarshalIndent(status, "", "\t")
			require.NoError(t, err)

			output, err := os.ReadFile(outPath)
			assert.NoError(t, err)

			var expectedEnv gobl.Envelope
			require.NoError(t, json.Unmarshal(output, &expectedEnv))
			expectedStatus, ok := expectedEnv.Extract().(*bill.Status)
			require.True(t, ok, "Expected document should be a status")
			expectedData, err := json.MarshalIndent(expectedStatus, "", "\t")
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedData), string(data), "Status should match the expected JSON. Update with --update flag.")
		})
	}
}
