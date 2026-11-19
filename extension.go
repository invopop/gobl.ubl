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

// ExtensionData is the content of a UBL extension: one element of any
// namespace other than the extension namespace. An implementation must name
// its element through an XMLName field, or it encodes as <Data>, and must
// declare every prefix it writes through xmlns:p,attr fields, or those
// prefixes come out unbound.
type ExtensionData interface {
	// ExtensionNamespaces maps each prefix the type's xml tags use to its
	// namespace, so the content can be decoded into the type.
	ExtensionNamespaces() map[string]string
}

// ExtensionContent wraps the content of a UBL extension.
type ExtensionContent struct {
	Data ExtensionData `xml:",any"`
}

// NewExtension creates a new extension holding the given data. data must not
// be nil or empty, as UBL requires exactly one element in ext:ExtensionContent.
func NewExtension(data ExtensionData) *Extension {
	return &Extension{
		ExtensionContent: &ExtensionContent{Data: data},
	}
}

// FindExtensionData returns the first extension content of type T, if any.
func FindExtensionData[T ExtensionData](exts *Extensions) (T, bool) {
	var zero T
	if exts == nil {
		return zero, false
	}
	for _, ext := range exts.Extension {
		if ext.ExtensionContent == nil {
			continue
		}
		if d, ok := ext.ExtensionContent.Data.(T); ok {
			return d, true
		}
	}
	return zero, false
}

// AddExtension adds a new extension to the ubl invoice
func (ui *Invoice) AddExtension(extension *Extension) {
	if ui.Extensions == nil {
		ui.Extensions = &Extensions{}
	}
	ui.Extensions.Extension = append(ui.Extensions.Extension, *extension)
}
