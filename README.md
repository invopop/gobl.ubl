# GOBL.UBL

GOBL conversion into UBL XML format and vice versa.

[![codecov](https://codecov.io/gh/invopop/gobl.ubl/graph/badge.svg?token=KWKFOSEEK7)](https://codecov.io/gh/invopop/gobl.ubl)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/invopop/gobl.ubl)

Copyright [Invopop Ltd.](https://invopop.com) 2025. Released publicly under the [Apache License Version 2.0](LICENSE). For commercial licenses, please contact the [dev team at invopop](mailto:dev@invopop.com). To accept contributions to this library, we require transferring copyrights to Invopop Ltd.

## Usage
### Go Package

Usage of the GOBL to UBL conversion library is straightforward and supports bidirectional conversion:

1. Convert GOBL to UBL XML:
   You must first have a GOBL Envelope, including an invoice, ready to convert. There are some samples in the `test/data` directory.

2. Parse UBL XML into GOBL:
   You need to have a valid UBL XML document that you want to convert to GOBL format.

Both conversion directions are supported, allowing you to seamlessly transform between GOBL and UBL XML formats as needed.

The package uses the same terms as GOBL's `convert` package: **export** maps a GOBL envelope into a UBL document and **import** maps it back, while **encode** and **decode** turn UBL documents into XML bytes and back.

#### Export GOBL to UBL

```go
package main

import (
    "os"

    "github.com/invopop/gobl"
    ubl "github.com/invopop/gobl.ubl"
)

func main() {
    data, _ := os.ReadFile("./test/data/invoice-sample.json")

    env := new(gobl.Envelope)
    if err := json.Unmarshal(data, env); err != nil {
        panic(err)
    }

    // Export the UBL document
    doc, err := ubl.Export(env)
    if err != nil {
        panic(err)
    }

    // Encode the XML output
    out, err := ubl.Encode(doc)
    if err != nil {
        panic(err)
    }

}
```

To export into a format other than the default EN 16931, add it as an option. `ExportInvoice` does the same when an invoice is expected:

```go
doc, err := ubl.ExportInvoice(env, ubl.WithFormat(ubl.FormatPeppol))
```

To embed files in an invoice, such as a PDF version of it, add them with `WithAttachments`. Each is written as an `AdditionalDocumentReference` with an `EmbeddedDocumentBinaryObject`. Other documents, such as Invoice Responses, cannot embed files and return `ErrUnsupportedAttachments`:

```go
doc, err := ubl.Export(env,
    ubl.WithFormat(ubl.FormatPeppol),
    ubl.WithAttachments(ubl.BinaryAttachment{
        ID:          "INV-001-PDF",
        Description: "Invoice INV-001",
        Data:        pdf,
        MimeCode:    "application/pdf",
        Filename:    "INV-001.pdf",
    }),
)
```

Through the GOBL `convert` register, pass them with `convert.WithAttachments`. Every UBL format embeds them, using the attachment's name as its ID when it has none:

```go
out, err := convert.Export(env, "ubl+peppol+bis-v3", convert.WithAttachments(&convert.Attachment{
    Name: "INV-001.pdf",
    MIME: "application/pdf",
    Data: pdf,
}))
```

#### Formats

This package provides the base UBL import and export, and the formats that apply in any country:

| Key | Format |
| --- | --- |
| `ubl+en16931-v2017` | `FormatEN16931` |
| `ubl+peppol+bis-v3` | `FormatPeppol` |

A format lists the GOBL schemas it imports into and exports from in `Import` and `Export`. Formats without `Import` schemas are export-only and never detected, as is needed for formats whose documents only differ from another format's by their container.

A format can write documents for more than one specification, choosing from the GOBL document. Its own `CustomizationID`, `ProfileID`, `Addons` and `VESIDs` define its main specification, and `Specifications` lists the others, each with an `Applies` function. On export, the first specification that applies to the document is used, or else the main one; `SpecificationFor` and `VESID` provide them for a document. Every specification's identifiers are recognised on import. The GOBL `convert` register only exports documents that include the addons of the specification they are exported with, and otherwise refuses them with a fault on `$addons`, such as `UBL-PEPPOL-BIS-V3-BILL-INVOICE-01`. `FormatPeppol` writes:

| GOBL document | Specification |
| --- | --- |
| `bill.Invoice` | Peppol BIS Billing 3.0 |
| `bill.Invoice` with the `self-billed` tag | Peppol BIS Self-Billing 3.0 (`SpecificationPeppolSelfBilling`) |
| `bill.Status` | Peppol Invoice Response 3 (`SpecificationPeppolInvoiceResponse`) |

The base import and export have no format-specific behavior. A format adds the rules of its specification with `ExportFuncs` and `ImportFuncs`, which adjust the finished document in order: an export function receives the GOBL envelope and the exported UBL document, and an import function the UBL document and the imported envelope, before it is calculated. `InvoiceParties` lists each GOBL party alongside its UBL party, for rules that apply to every party. Formats that allow sub-invoice lines, such as EXTENDED-CTC-FR, set `SubLines`: line breakdowns are then written as sub-invoice lines, and read back from the line billing references that name the invoice itself.

Regional formats live in their own modules, which register them with `ubl.RegisterFormats` when imported, making them available to `FindFormat`, `Import`, and the GOBL `convert` register:

| Key | Module |
| --- | --- |
| `ubl+fr-ctc-cius-v1`, `ubl+fr-ctc-extended-v1` | [gobl.fr.ctc](https://github.com/invopop/gobl.fr.ctc) (`_ "github.com/invopop/gobl.fr.ctc/ubl"`) |
| `ubl+sa-zatca-v1` | [gobl.sa.zatca](https://github.com/invopop/gobl.sa.zatca) (`_ "github.com/invopop/gobl.sa.zatca/ubl"`) |
| `ubl+de-xrechnung-v3` | [gobl.de.xinvoice](https://github.com/invopop/gobl.de.xinvoice) |

#### Import UBL to GOBL

```go
package main

import (
    "encoding/json"
    "os"

    ubl "github.com/invopop/gobl.ubl"
)

func main() {
    // Read the UBL XML file
    inData, err := os.ReadFile("path/to/ubl_invoice.xml")
    if err != nil {
        panic(err)
    }

    // Decode the UBL document
    doc, err := ubl.Decode(inData)
    if err != nil {
        panic(err)
    }

    // Import into a GOBL envelope
    env, err := ubl.Import(doc)
    if err != nil {
        panic(err)
    }

    // Extract binary attachments if needed
    if inv, ok := doc.(*ubl.Invoice); ok {
        attachments := inv.ExtractBinaryAttachments()
    }

    // Marshal to JSON
    outputData, err := json.MarshalIndent(env, "", "  ")
    if err != nil {
        panic(err)
    }
}
```

Through the GOBL `convert` register, `convert.Import` returns the files embedded in an invoice as `Attachments`, alongside the envelope:

```go
res, err := convert.Import(data)
for _, a := range res.Attachments {
    // store a.Data as a.Name
}
```

#### Tax Data Documents

A Peppol Tax Data Document (TDD) reports an invoice to a tax authority, next to the invoice itself. This package has the TDD structures (`TaxData`) and the base profile, the Peppol ViDA TDD 1.1.0 (`TaxDataProfileViDA`). National profiles, such as the Slovak TDD, set their own identifiers and invoice UUID inputs in a `TaxDataProfile`. The TDD is not a UBL format: the app that reports to the tax authority builds it.

There are two ways to build a TDD:

- **From GOBL.** `ExportTaxData` exports the invoice to Peppol BIS Billing 3.0 first (or the format of a `WithFormat` option), then builds the TDD from that UBL invoice. The report always matches the invoice that is sent.
- **From UBL.** `BuildTaxData` takes the UBL XML of the invoice as it was sent or received, for example when the receiver's provider (C3) reports.

```go
td, err := ubl.ExportTaxData(env, ubl.TaxDataProfileViDA, ubl.TaxDataParams{
    TypeCode:         ubl.TaxDataTypeSubmit,     // S, R or D
    Scope:            ubl.TaxDataScopeDomestic,  // D, IC or INTL
    ReporterRole:     ubl.TaxDataReporterSender, // C2 or C3
    TaxAuthorityID:   "...",                     // from the tax authority
    ReceiverEndpoint: "...",                     // the tax authority's 0242 participant (C5)
    Representative:   "...",                     // the provider's SPID
    TransmissionUUID: "...",                     // the transmission of the invoice
})
data, err := ubl.Encode(td)
```

The values in `TaxDataParams` do not come from the invoice. The tax authority endpoint and the provider's SPID change between the test and production networks. The type code depends on earlier answers from the tax authority, so the caller keeps that state. An invoice without the values its profile needs for the invoice UUID returns `ErrTaxDataNotReportable`.

## Command Line

This package no longer includes a command-line tool. The `gobl` CLI from [gobl.dev](https://github.com/invopop/gobl.dev) converts between GOBL and every format registered with the GOBL `convert` register, including UBL:

```bash
go install github.com/invopop/gobl.dev/cmd/gobl@latest

gobl convert invoice.json invoice.xml --to ubl+en16931-v2017   # export
gobl convert invoice.xml                                       # import, format detected
gobl convert --list                                            # registered formats
```

## Testing

### testify

The library uses testify for testing. To run the tests, you can use the following command:

```bash
go test ./...
```

### Schematron validation

Beyond the golden-file comparisons, the generated XML can be pushed through the
real EN 16931 / Peppol schematron rule sets.
Validation runs against [phorm](https://github.com/phax/phorm), the standalone
validation service that replaced the now-archived `invopop/phive` gRPC wrapper,
using the [`invopop/phorm`](https://github.com/invopop/phorm) HTTP client.

Start a service locally:

```bash
docker run -d --name phorm -p 8080:8080 phelger/phorm
```

Use `phelger/phorm-arm64` on Apple Silicon. Note the image is `phelger/phorm`,
**not** `phax/phorm` — the latter does not exist, despite what the
`invopop/phorm` README says.

It takes a few seconds to boot. It is ready once this returns HTTP 200:

```bash
curl -s -o /dev/null -w '%{http_code}\n' \
  -H 'X-Token: phorm-dev-token' \
  'http://localhost:8080/api/get/vesids?include-deprecated=true'
```

Then run the suite with `-validate`:

```bash
go test ./... -validate
```

Without `-validate` the validating tests are skipped, so the plain `go test
./...` never needs a service and CI stays offline.

#### Pointing somewhere else

`PHORM_URL` and `PHORM_TOKEN` default to `http://localhost:8080` and phorm's
stock development token. Override them for a shared instance, or when port 8080
is already taken locally:

```bash
docker run -d --name phorm -p 8085:8080 phelger/phorm
PHORM_URL=http://localhost:8085 go test ./... -validate
```

#### Known failures

These fixtures do not currently pass schematron. They are long-standing gaps in
the conversion rather than regressions, so a run is "clean" when only these
fail:

| Rule | Fixtures | Cause |
| --- | --- | --- |
| `PEPPOL-EN16931-R061` | `peppol/invoice-prices-include-vat.json` | The fixture pays by SEPA direct debit but carries no mandate reference (BT-89). |

#### Notes

- **A failed validation is not an error.** phorm answers a document that breaks
  a rule with an HTTP 400 carrying the report, which it also uses for a request
  it rejects outright, so `invopop/phorm` separates the two by whether the body
  is a validation report (fixed in v0.1.5). An error from `ValidateXml`
  therefore means the validation never ran — unreachable service, rejected
  token, unresolvable VESID, or a body that is not XML — and the tests treat it
  as fatal, since nothing was checked.
- **phorm normalises VESID versions**, so a spelling such as
  `fr.ctc:ubl-invoice:1.4.0-03` resolves to its published `fr.ctc:ubl-invoice:1.4-03`
  rule set. The resolved id comes back as `ves.vesid`, which is worth checking
  when a rule set behaves unexpectedly.
- **phive-rules keeps only a rolling window of releases.** VESIDs that pass
  today are dropped a few releases later, so the VESIDs in `format.go` need periodic
  updating; `GET /api/get/vesids?include-deprecated=true` lists what a given
  phorm build actually carries, along with a `deprecated` flag.

## Considerations

There are certain assumptions and lost information in the conversion from UBL to GOBL that should be considered:

1. GOBL does not currently support additional embedded documents, so the AdditionalReferencedDocument field (BG-24 in EN 16931) is not supported and lost in the conversion.
2. GOBL only supports a single period in the ordering, so only the first InvoicePeriod (BG-14) in the UBL is taken.
3. Fields ProfileID (BT-23) and CustomizationID (BT-24) in UBL are not supported and lost in the conversion.
4. The AccountingCost (BT-19, BT-133) fields are added as notes.
5. Payment advances do not include their own tax rate, they use the global tax rate of the invoice.

## Development

The main source of information for this project comes from the EN 16931 standard, developed by the EU for electronic invoicing. [Part 1](https://standards.iteh.ai/catalog/standards/cen/4f31d4a9-53eb-4f1a-835e-6f0583cad2bb/en-16931-1-2017) of the standard defines the semantic data model that forms an invoice, but does not provide a concrete implementation. [Part 3.2](https://standards.iteh.ai/catalog/standards/cen/07652211-da2d-4ad7-871f-36ee918e9a01/cen-ts-16931-3-2-2020) defines the mappings from the semantic data model to the UBL 2.1 XML format covered in this repository.

Useful links:

- [Official UBL 2.1 Specification](https://docs.oasis-open.org/ubl/UBL-2.1.html)
