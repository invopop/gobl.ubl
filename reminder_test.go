package ubl_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const oioublReminderDir = "oioubl21-reminder"

func TestConvertToReminder(t *testing.T) {
	examples, err := filepath.Glob(filepath.Join(getConvertPath(), oioublReminderDir, jsonPattern))
	require.NoError(t, err)
	require.NotEmpty(t, examples, "no Reminder examples found")

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

			outPath := filepath.Join(getConvertPath(), oioublReminderDir, "out", outName)
			if *updateOut {
				require.NoError(t, os.WriteFile(outPath, data, 0644))
			}

			output, err := os.ReadFile(outPath)
			assert.NoError(t, err)
			assert.Equal(t, string(output), string(data), "Output should match the expected XML. Update with --update flag.")
		})
	}
}

func TestParseOIOUBL21Reminder(t *testing.T) {
	examples, err := filepath.Glob(filepath.Join(getParsePath(), oioublReminderDir, xmlPattern))
	require.NoError(t, err)
	require.NotEmpty(t, examples, "no Reminder parse examples found")

	for _, example := range examples {
		inName := filepath.Base(example)
		outName := strings.Replace(inName, ".xml", ".json", 1)

		t.Run(inName, func(t *testing.T) {
			xmlData, err := os.ReadFile(example)
			require.NoError(t, err)

			doc, err := ubl.Parse(xmlData)
			require.NoError(t, err)
			rem, ok := doc.(*ubl.Reminder)
			require.True(t, ok, "Document should be a Reminder")

			env, err := rem.Convert()
			require.NoError(t, err)

			env.Head.UUID = staticUUID
			if pmt, ok := env.Extract().(*bill.Payment); ok {
				pmt.UUID = staticUUID
			}
			require.NoError(t, env.Calculate())

			outPath := filepath.Join(getParsePath(), oioublReminderDir, "out", outName)
			if *updateOut {
				data, err := json.MarshalIndent(env, "", "\t")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(outPath, data, 0644))
			}

			payment, ok := env.Extract().(*bill.Payment)
			require.True(t, ok, "Document should be a payment")
			data, err := json.MarshalIndent(payment, "", "\t")
			require.NoError(t, err)

			output, err := os.ReadFile(outPath)
			assert.NoError(t, err)

			var expectedEnv gobl.Envelope
			require.NoError(t, json.Unmarshal(output, &expectedEnv))
			expectedPayment, ok := expectedEnv.Extract().(*bill.Payment)
			require.True(t, ok, "Expected document should be a payment")
			expectedData, err := json.MarshalIndent(expectedPayment, "", "\t")
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedData), string(data), "Payment should match the expected JSON. Update with --update flag.")
		})
	}
}
