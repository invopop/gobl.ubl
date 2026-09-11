package ubl_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/invopop/phorm"
	"github.com/stretchr/testify/require"
)

// Schematron validation runs against phorm (https://github.com/phax/phorm), the
// standalone validation service that replaced the archived invopop/phive gRPC
// wrapper. Run one locally with:
//
//	docker run -d --name phorm -p 8080:8080 phelger/phorm
//
// then run the suite with -validate. Point PHORM_URL / PHORM_TOKEN elsewhere to
// use a shared instance; the defaults match a stock local container.
const (
	defaultPhormURL = "http://localhost:8080"

	phormURLEnv   = "PHORM_URL"
	phormTokenEnv = "PHORM_TOKEN"
)

func phormURL() string {
	if url := os.Getenv(phormURLEnv); url != "" {
		return strings.TrimRight(url, "/")
	}
	return defaultPhormURL
}

func phormToken() string {
	if token := os.Getenv(phormTokenEnv); token != "" {
		return token
	}
	return phorm.DefaultToken
}

// phormClient returns a client for the validation service, skipping the test
// when -validate was not requested. An empty token makes phorm fall back to its
// stock development token.
func phormClient(t *testing.T) *phorm.Client {
	t.Helper()

	if !*validate {
		t.Skip("schematron validation not requested (use -validate)")
	}
	return phorm.New(phormURL(), os.Getenv(phormTokenEnv))
}

// finding is a single schematron error or warning.
type finding struct {
	Level string
	Rule  string
	Text  string
	Field string
}

func (f finding) String() string {
	out := f.Level + ": " + f.Text
	if f.Rule != "" {
		out += " [" + f.Rule + "]"
	}
	if f.Field != "" {
		out += " at " + f.Field
	}
	return out
}

// validateXML pushes the document through phorm and fails the test with every
// error the rule set raises.
func validateXML(t *testing.T, pc *phorm.Client, vesid string, data []byte) {
	t.Helper()

	problems, err := phormValidate(t, pc, vesid, data)
	require.NoError(t, err)

	var errs []string
	for _, p := range problems {
		if p.Level == "ERROR" || p.Level == "FATAL_ERROR" {
			errs = append(errs, p.String())
		}
	}
	if len(errs) > 0 {
		t.Errorf("%s: %d schematron error(s):\n%s", vesid, len(errs), strings.Join(errs, "\n"))
	}
}

// phormValidate validates the document and returns every finding, errors and
// warnings alike.
//
// phorm answers a failed validation with HTTP 400, and its Go client turns any
// non-2xx status into an error after truncating the body, so a document that
// breaks a rule arrives as a transport error with the findings cut out of it.
// The report is therefore read straight off the HTTP API when the client
// reports an error, and only a response that is not a validation report at all
// is surfaced as a genuine failure.
func phormValidate(t *testing.T, pc *phorm.Client, vesid string, data []byte) ([]finding, error) {
	t.Helper()

	resp, clientErr := pc.ValidateXml(context.Background(), &phorm.ValidateXmlRequest{
		Vesid:      vesid,
		XmlContent: data,
	})
	if clientErr == nil {
		var out []finding
		for _, r := range resp.Results {
			for _, e := range r.Errors {
				out = append(out, finding{Level: "ERROR", Rule: e.TestId, Text: e.Message, Field: e.Xpath})
			}
			for _, w := range r.Warnings {
				out = append(out, finding{Level: "WARN", Rule: w.TestId, Text: w.Message, Field: w.Xpath})
			}
		}
		return out, nil
	}

	return phormReport(vesid, data)
}

// phormReport posts the document to phorm and decodes the validation report
// whatever status code it comes back with.
func phormReport(vesid string, data []byte) ([]finding, error) {
	url := phormURL() + "/api/validate/" + vesid
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Token", phormToken())

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("phorm unreachable at %s: %w", phormURL(), err)
	}
	defer res.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var report struct {
		Results []struct {
			Items []struct {
				ErrorLevel     string `json:"errorLevel"`
				ErrorText      string `json:"errorText"`
				ErrorFieldName string `json:"errorFieldName"`
				Test           string `json:"test"`
			} `json:"items"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &report); err != nil {
		return nil, fmt.Errorf("phorm returned %s for %s, not a validation report: %s",
			res.Status, vesid, truncate(body))
	}
	if len(report.Results) == 0 {
		return nil, fmt.Errorf("phorm returned %s for %s with no results: %s",
			res.Status, vesid, truncate(body))
	}

	var out []finding
	for _, r := range report.Results {
		for _, it := range r.Items {
			out = append(out, finding{
				Level: it.ErrorLevel,
				Rule:  it.Test,
				Text:  it.ErrorText,
				Field: it.ErrorFieldName,
			})
		}
	}
	return out, nil
}

func truncate(b []byte) string {
	const limit = 400
	if len(b) <= limit {
		return string(b)
	}
	return string(b[:limit]) + "…"
}
