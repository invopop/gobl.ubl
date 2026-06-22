package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/spf13/cobra"
)

type convertOpts struct {
	*rootOpts
	contextName string
	profileID   string
}

// converter is implemented by every UBL document Parse can return (Invoice,
// ApplicationResponse, Reminder); each converts back to a GOBL envelope.
type converter interface {
	Convert() (*gobl.Envelope, error)
}

func convert(o *rootOpts) *convertOpts {
	return &convertOpts{rootOpts: o}
}

func (c *convertOpts) cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert <infile> [outfile]",
		Short: "Convert between GOBL and UBL documents",
		Long: `Convert between GOBL and UBL, auto-detecting the direction from the input:

  - JSON input is read as a GOBL envelope and converted to a UBL XML document.
    Use --context (and optionally --profile-id) to select the UBL customization.
  - XML input is read as a UBL document (Invoice, ApplicationResponse or Reminder)
    and converted back to a GOBL envelope.

<infile> is required; [outfile] defaults to stdout. Either may be "-" for stdin/stdout.`,
		RunE: c.runE,
	}

	flags := cmd.Flags()
	flags.StringVar(&c.contextName, "context", "",
		"UBL customization for JSON->XML conversion: en16931, peppol, peppol-self-billed, "+
			"xrechnung, peppol-france-cius, peppol-france-extended, zatca (default en16931)")
	flags.StringVar(&c.profileID, "profile-id", "",
		"Override the UBL ProfileID (JSON->XML conversion only)")

	return cmd
}

func (c *convertOpts) runE(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || len(args) > 2 {
		return fmt.Errorf("expected one or two arguments, the command usage is `gobl.ubl convert <infile> [outfile]`")
	}

	input, err := openInput(cmd, args)
	if err != nil {
		return err
	}
	defer input.Close() // nolint:errcheck

	out, err := c.openOutput(cmd, args)
	if err != nil {
		return err
	}
	defer out.Close() // nolint:errcheck

	inData, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	// Direction is auto-detected: JSON => UBL XML, otherwise UBL => GOBL.
	var outputData []byte
	if json.Valid(inData) {
		env := new(gobl.Envelope)
		if err := json.Unmarshal(inData, env); err != nil {
			return fmt.Errorf("parsing input as GOBL Envelope: %w", err)
		}
		opts, err := c.buildOptions()
		if err != nil {
			return err
		}
		doc, err := ubl.Convert(env, opts...)
		if err != nil {
			return fmt.Errorf("building UBL document: %w", err)
		}

		outputData, err = ubl.Bytes(doc)
		if err != nil {
			return fmt.Errorf("generating UBL xml: %w", err)
		}
	} else {
		// Assume XML if not JSON

		doc, err := ubl.Parse(inData)
		if err != nil {
			return fmt.Errorf("building GOBL envelope: %w", err)
		}

		// Every supported UBL document (Invoice, ApplicationResponse, Reminder)
		// converts back to a GOBL envelope through this method.
		conv, ok := doc.(converter)
		if !ok {
			return fmt.Errorf("building GOBL envelope: %w", ubl.ErrUnsupportedDocumentType)
		}

		env, err := conv.Convert()
		if err != nil {
			return fmt.Errorf("building GOBL envelope: %w", err)
		}

		outputData, err = json.MarshalIndent(env, "", "  ")
		if err != nil {
			return fmt.Errorf("generating JSON output: %w", err)
		}
	}

	if _, err = out.Write(outputData); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	return nil
}

func (c *convertOpts) buildOptions() ([]ubl.Option, error) {
	if c.contextName == "" && c.profileID == "" {
		return nil, nil
	}

	ctx := ubl.ContextEN16931
	if c.contextName != "" {
		found, ok := ubl.ContextByName(c.contextName)
		if !ok {
			return nil, fmt.Errorf("unknown context %q", c.contextName)
		}
		ctx = found
	}

	if c.profileID != "" {
		ctx.ProfileID = c.profileID
	}

	return []ubl.Option{ubl.WithContext(ctx)}, nil
}
