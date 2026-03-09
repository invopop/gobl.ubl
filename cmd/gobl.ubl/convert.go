package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/invopop/gobl"
	ubl "github.com/invopop/gobl.ubl"
	"github.com/spf13/cobra"
)

type convertOpts struct {
	*rootOpts
	contextName string
	profileID   string
}

func convert(o *rootOpts) *convertOpts {
	return &convertOpts{rootOpts: o}
}

func (c *convertOpts) cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert <infile> <outfile>",
		Short: "Convert a GOBL JSON into a Universal Business Language (UBL) document and vice versa",
		RunE:  c.runE,
	}

	flags := cmd.Flags()
	flags.StringVar(&c.contextName, "context", "", "Context for UBL conversion (en16931, peppol, xrechnung, nemhandel, ...)")
	flags.StringVar(&c.profileID, "profile-id", "", "Override UBL ProfileID for JSON to XML conversion")

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

	// Check if input is JSON or XML
	isJSON := json.Valid(inData)

	var outputData []byte

	if isJSON {
		env := new(gobl.Envelope)
		if err := json.Unmarshal(inData, env); err != nil {
			return fmt.Errorf("parsing input as GOBL Envelope: %w", err)
		}
		opts, err := c.buildOptions()
		if err != nil {
			return err
		}
		doc, err := ubl.ConvertInvoice(env, opts...)
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

		inv, ok := doc.(*ubl.Invoice)
		if !ok {
			return fmt.Errorf("building GOBL envelope: %w", ubl.ErrUnsupportedDocumentType)
		}

		env, err := inv.Convert()
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
		switch strings.ToLower(c.contextName) {
		case "en16931", "en":
			ctx = ubl.ContextEN16931
		case "peppol":
			ctx = ubl.ContextPeppol
		case "peppol-self-billed", "peppol-selfbilled", "peppol-self":
			ctx = ubl.ContextPeppolSelfBilled
		case "xrechnung":
			ctx = ubl.ContextXRechnung
		case "peppol-france-cius", "france-cius", "fr-cius":
			ctx = ubl.ContextPeppolFranceCIUS
		case "peppol-france-extended", "france-extended", "fr-extended":
			ctx = ubl.ContextPeppolFranceExtended
		case "nemhandel", "oioubl":
			ctx = ubl.ContextOIOUBL
		case "nemhandel-2.1", "oioubl-2.1", "oioubl21":
			ctx = ubl.ContextOIOUBL21
		default:
			return nil, fmt.Errorf("unknown context %q", c.contextName)
		}
	}

	if c.profileID != "" {
		ctx.ProfileID = c.profileID
	}

	return []ubl.Option{ubl.WithContext(ctx)}, nil
}
