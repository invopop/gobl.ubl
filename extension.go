package ubl

// Extensions wraps a list of UBL extensions.
type Extensions struct {
	Extension []Extension `xml:"ext:UBLExtension"`
}

// Extension represents a single UBL extension.
type Extension struct {
	ExtensionURI     *string           `xml:"ext:ExtensionURI"`
	ExtensionContent *ExtensionContent `xml:"ext:ExtensionContent"`
}

// ExtensionContent wraps the content of a UBL extension.
type ExtensionContent struct {
	UBLDocumentSignatures *DocumentSignatures `xml:"sig:UBLDocumentSignatures"`
	// Raw holds content not modelled here, such as the data of a regional
	// format. Set it instead of UBLDocumentSignatures, as an extension holds
	// one element, and declare the namespaces it uses: it is written as is.
	// Decoding fills only Raw, with the element re-encoded so that each
	// element declares its own namespace: fit for reading, not a copy of the
	// original bytes.
	Raw []byte `xml:",innerxml"`
}

// NewExtension creates a new extension
func NewExtension() *Extension {
	return &Extension{
		ExtensionContent: &ExtensionContent{
			UBLDocumentSignatures: &DocumentSignatures{},
		},
	}
}

// AddExtension adds a new extension to the ubl invoice
func (ui *Invoice) AddExtension(extension *Extension) {
	if ui.Extensions == nil {
		ui.Extensions = &Extensions{}
	}
	ui.Extensions.Extension = append(ui.Extensions.Extension, *extension)
}
