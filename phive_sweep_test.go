package ubl_test

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/invopop/phive"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var sweep = flag.Bool("sweep", false, "validate every stored OIOUBL XML against phive")

func TestPhiveSweepAllStoredOIOUBL(t *testing.T) {
	if !*sweep {
		t.Skip("run with -sweep")
	}
	conn, err := grpc.NewClient("127.0.0.1:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	pc := phive.NewValidationServiceClient(conn)

	dirs := []string{
		"test/data/convert/oioubl21/out",
		"test/data/convert/oioubl21-response/out",
		"test/data/parse/oioubl21",
		"test/data/parse/oioubl21-response",
	}
	for _, dir := range dirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.xml"))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			t.Run(f, func(t *testing.T) {
				data, err := os.ReadFile(f)
				if err != nil {
					t.Fatal(err)
				}
				vesid := "dk.oioubl:invoice:1.17.2"
				if bytes.Contains(data[:min(400, len(data))], []byte("CreditNote")) {
					vesid = "dk.oioubl:credit-note:1.17.2"
				}
				if bytes.Contains(data[:min(400, len(data))], []byte("ApplicationResponse")) {
					vesid = "dk.oioubl:application-response:1.17.2"
				}
				resp, err := pc.ValidateXml(context.Background(), &phive.ValidateXmlRequest{
					Vesid:      vesid,
					XmlContent: data,
				})
				if err != nil {
					t.Fatal(err)
				}
				if !resp.Success {
					t.Errorf("INVALID (%s): %s %v", vesid, resp.ErrorMessage, resp.Results)
				}
			})
		}
	}
}
