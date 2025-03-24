# GOBL.UBL

GOBL conversion into UBL XML format and vice versa.

[![codecov](https://codecov.io/gh/invopop/gobl.ubl/graph/badge.svg?token=KWKFOSEEK7)](https://codecov.io/gh/invopop/gobl.ubl)

Copyright [Invopop Ltd.](https://invopop.com) 2023. Released publicly under the [Apache License Version 2.0](LICENSE). For commercial licenses, please contact the [dev team at invopop](mailto:dev@invopop.com). To accept contributions to this library, we require transferring copyrights to Invopop Ltd.

## Usage

### Go Package

Usage of the GOBL to UBL conversion library is straightforward and supports bidirectional conversion:

1. Convert GOBL to UBL XML:
   You must first have a GOBL Envelope, including an invoice, ready to convert. There are some samples in the `test/data` directory.

2. Parse UBL XML into GOBL:
   You need to have a valid UBL XML document that you want to convert to GOBL format.

Both conversion directions are supported, allowing you to seamlessly transform between GOBL and UBL XML formats as needed.

#### Convert GOBL to UBL

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

    // Prepare the UBL Invoice document
    doc, err := ubl.ConvertInvoice(env)
    if err != nil {
        panic(err)
    }

    // Create the XML output
    out, err := doc.Bytes()
    if err != nil {
        panic(err)
    }

}
```

#### UBL to GOBL

```go
package main

import (
    "io"

    ubl "github.com/invopop/gobl.ubl"
    )

func main() {
    // Read the UBL XML file
	inData, err := io.ReadAll("path/to/ubl_invoice.xml")
	if err != nil {
		panic(err)
	}

    env, err := ubl.ParseInvoice(inData)
    if err != nil {
        panic(err)
    }

    outputData, err = json.MarshalIndent(env, "", "  ")
    if err != nil {
        panic(err)
    }
}
```

## Command Line

The GOBL to UBL tool includes a command-line helper. You can install it manually in your Go environment with:

```bash
go install ./cmd/gobl.ubl
```

Once installed, usage is straightforward. The tool automatically detects the input file type and performs the appropriate conversion:

- If the input is a JSON file (GOBL format), it will convert it to UBL XML.
- If the input is an XML file (UBL format), it will convert it to GOBL JSON.

For example:

```bash
gobl.ubl convert ./test/data/invoice-sample.json
```

## Testing
### testify
The library uses testify for testing. To run the tests, you can use the following command:

```bash
go test ./...
```

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
