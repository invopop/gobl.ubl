package ubl

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/uuid"
)

// Peppol Tax Data Document (TDD) identifiers shared by every profile.
const (
	// TaxDataProfileID is the business process of every TDD (TDT-002).
	TaxDataProfileID = "urn:peppol:taxreporting"
	// TaxDataInvoiceUUIDNamespace is the fixed UUID v5 namespace used to
	// calculate the invoice UUID (TDT-017) in ViDA and its national profiles.
	TaxDataInvoiceUUIDNamespace = "e0bc4ac8-b025-46e5-a76d-0c893fc3027e"
)

// TaxDataTypeCode values (TDT-007) of the ViDA family.
const (
	TaxDataTypeSubmit    = "S"
	TaxDataTypeResubmit  = "R"
	TaxDataTypeDisregard = "D"
)

// TaxDataScope values (TDT-006) of the ViDA family.
const (
	TaxDataScopeDomestic       = "D"
	TaxDataScopeIntraCommunity = "IC"
	TaxDataScopeInternational  = "INTL"
)

// TaxDataReporterRole values (TDT-012) of the ViDA family.
const (
	TaxDataReporterSender   = "C2"
	TaxDataReporterReceiver = "C3"
)

// Errors returned while building a Tax Data Document.
var (
	// ErrTaxDataNotReportable means the invoice cannot be reported, as
	// a value the invoice UUID depends on is missing.
	ErrTaxDataNotReportable = errors.New("invoice cannot be reported in a tax data document")
	// ErrTaxDataParams means the parameters of the report are incomplete or
	// invalid.
	ErrTaxDataParams = errors.New("invalid tax data parameters")
)

// TaxDataProfile describes one jurisdiction's Tax Data Document. The ViDA
// profile is the base, and national profiles such as Slovakia's change the
// identifiers, the invoice UUID inputs, and a few elements.
type TaxDataProfile struct {
	// Key identifies the profile, e.g. "tdd-vida-v1".
	Key cbc.Key
	// Name of the profile.
	Name i18n.String
	// Namespace of the pxs elements.
	Namespace string
	// CustomizationID of the TDD (TDT-001).
	CustomizationID string
	// VESID identifies the official validation rules of the profile.
	VESID string
	// TypeCodes, Scopes and the two reporter roles are the codes the profile
	// accepts in the parameters.
	TypeCodes    []string
	Scopes       []string
	SenderRole   string
	ReceiverRole string
	// TaxAuthority is true when the profile identifies the tax authority (TDG-04).
	TaxAuthority bool
	// TransmissionUUID is true when the profile carries the invoice
	// transmission UUID (TDT-018).
	TransmissionUUID bool
	// BuyerName is true when the profile carries the buyer's registration
	// name (BT-44).
	BuyerName bool
	// InvoiceUUID builds the name hashed into the invoice UUID (TDT-017) from
	// the invoice, or returns ErrTaxDataNotReportable.
	InvoiceUUID func(inv *Invoice) (string, error)
}

// TaxDataProfileViDA is the base profile, Peppol ViDA Tax Data Document 1.1.0.
var TaxDataProfileViDA = TaxDataProfile{
	Key:              "tdd-vida-v1",
	Name:             i18n.NewString("Peppol ViDA Tax Data Document"),
	Namespace:        "urn:peppol:schema:vida-taxdata:1.0",
	CustomizationID:  "urn:peppol:taxdata:vida-1",
	VESID:            "org.peppol.taxdata:vida:1.1.0",
	TypeCodes:        []string{TaxDataTypeSubmit, TaxDataTypeResubmit, TaxDataTypeDisregard},
	Scopes:           []string{TaxDataScopeDomestic, TaxDataScopeIntraCommunity, TaxDataScopeInternational},
	SenderRole:       TaxDataReporterSender,
	ReceiverRole:     TaxDataReporterReceiver,
	TaxAuthority:     true,
	TransmissionUUID: true,
	InvoiceUUID:      vidaInvoiceUUIDName,
}

// TaxDataParams holds the values of a report that do not come from the
// invoice: the state of the report and the parties of the reporting flow.
type TaxDataParams struct {
	// TypeCode is one of the profile's type codes, e.g. S (submit).
	TypeCode string
	// Scope is one of the profile's scopes, e.g. D (domestic).
	Scope string
	// ReporterRole is the profile's sender or receiver role, e.g. C2 or C3.
	ReporterRole string
	// TaxAuthorityID and TaxAuthorityName identify the tax authority (TDT-010,
	// TDT-011), required by profiles that carry it.
	TaxAuthorityID   string
	TaxAuthorityName string
	// ReceiverEndpoint is the tax authority's access point, scheme 0242 (TDT-014).
	ReceiverEndpoint string
	// Representative is the reporting provider's SPID, scheme 0242 (TDT-015).
	Representative string
	// ReportingParty replaces the endpoint of the reporting party, which is
	// otherwise the seller's or the buyer's endpoint in the invoice.
	ReportingParty *EndpointID
	// TransmissionUUID identifies the transmission of the invoice (TDT-018),
	// required by profiles that carry it.
	TransmissionUUID string
	// UUID of the tax data document (TDT-003). A new UUID v4 when empty.
	UUID string
	// IssuedAt is the issue date and time of the report. Now when zero.
	IssuedAt time.Time
}

// TaxData is the root of a Peppol Tax Data Document.
type TaxData struct {
	XMLName      xml.Name
	PXSNamespace string `xml:"xmlns:pxs,attr"`
	CBCNamespace string `xml:"xmlns:cbc,attr"`
	CACNamespace string `xml:"xmlns:cac,attr"`

	CustomizationID         string                `xml:"cbc:CustomizationID"`
	ProfileID               string                `xml:"cbc:ProfileID"`
	UUID                    string                `xml:"cbc:UUID"`
	IssueDate               string                `xml:"cbc:IssueDate"`
	IssueTime               string                `xml:"cbc:IssueTime"`
	TaxDataTypeCode         string                `xml:"pxs:TaxDataTypeCode"`
	DocumentScope           string                `xml:"pxs:DocumentScope"`
	ReporterRole            string                `xml:"pxs:ReporterRole"`
	TaxAuthority            *TaxDataTaxAuthority  `xml:"pxs:TaxAuthority,omitempty"`
	ReportingParty          TaxDataEndpointParty  `xml:"pxs:ReportingParty"`
	ReceivingParty          TaxDataEndpointParty  `xml:"pxs:ReceivingParty"`
	ReportersRepresentative TaxDataRepresentative `xml:"pxs:ReportersRepresentative"`
	ReportedTransaction     TaxDataTransaction    `xml:"pxs:ReportedTransaction"`
}

func (*TaxData) ublDocument() {}

// TaxDataTaxAuthority identifies the tax authority (TDG-04).
type TaxDataTaxAuthority struct {
	ID   string `xml:"cbc:ID"`
	Name string `xml:"cbc:Name,omitempty"`
}

// TaxDataEndpointParty is a party identified only by its endpoint.
type TaxDataEndpointParty struct {
	EndpointID EndpointID `xml:"cbc:EndpointID"`
}

// TaxDataRepresentative is the provider that reports for the party.
type TaxDataRepresentative struct {
	PartyIdentification Identification `xml:"cac:PartyIdentification"`
}

// TaxDataTransaction holds the reported invoice (TDG-01).
type TaxDataTransaction struct {
	TransmissionUUID string          `xml:"pxs:TransmissionUUID,omitempty"`
	ReportedDocument TaxDataDocument `xml:"pxs:ReportedDocument"`
}

// ExportTaxData builds the Tax Data Document that reports the GOBL invoice in
// the envelope with the profile. The source is always GOBL, also for the
// receiver's provider (C3), which imports the invoice it receives first. The
// invoice is exported to Peppol BIS Billing 3.0 (or the format of a WithFormat
// option), and the TDD copies its values from that export, so the report
// carries the same values as the invoice that is sent.
func ExportTaxData(env *gobl.Envelope, profile TaxDataProfile, params TaxDataParams, opts ...Option) (*TaxData, error) {
	if err := checkTaxDataParams(profile, params); err != nil {
		return nil, err
	}
	doc, err := Export(env, append([]Option{WithFormat(FormatPeppol)}, opts...)...)
	if err != nil {
		return nil, err
	}
	inv, ok := doc.(*Invoice)
	if !ok {
		return nil, ErrUnsupportedDocumentType
	}
	return newTaxData(inv, profile, params)
}

// newTaxData builds the document of the profile from the exported invoice.
func newTaxData(inv *Invoice, profile TaxDataProfile, params TaxDataParams) (*TaxData, error) {
	if profile.InvoiceUUID == nil {
		return nil, fmt.Errorf("%w: profile %s has no invoice UUID method", ErrTaxDataParams, profile.Key)
	}
	name, err := profile.InvoiceUUID(inv)
	if err != nil {
		return nil, err
	}

	reporter, err := taxDataReporter(inv, profile, params)
	if err != nil {
		return nil, err
	}

	issued := params.IssuedAt
	if issued.IsZero() {
		issued = time.Now()
	}
	id := params.UUID
	if id == "" {
		id = uuid.V4().String()
	}

	spid := schemeIDSPID
	td := &TaxData{
		XMLName:         xml.Name{Local: "pxs:TaxData"},
		PXSNamespace:    profile.Namespace,
		CBCNamespace:    NamespaceCBC,
		CACNamespace:    NamespaceCAC,
		CustomizationID: profile.CustomizationID,
		ProfileID:       TaxDataProfileID,
		UUID:            id,
		IssueDate:       issued.UTC().Format("2006-01-02"),
		IssueTime:       issued.UTC().Format("15:04:05Z"),
		TaxDataTypeCode: params.TypeCode,
		DocumentScope:   params.Scope,
		ReporterRole:    params.ReporterRole,
		ReportingParty:  TaxDataEndpointParty{EndpointID: *reporter},
		ReceivingParty: TaxDataEndpointParty{
			EndpointID: EndpointID{SchemeID: schemeIDSPID, Value: params.ReceiverEndpoint},
		},
		ReportersRepresentative: TaxDataRepresentative{
			PartyIdentification: Identification{
				ID: &IDType{SchemeID: &spid, Value: params.Representative},
			},
		},
		ReportedTransaction: TaxDataTransaction{
			ReportedDocument: taxDataDocument(inv, profile),
		},
	}
	if profile.TaxAuthority {
		td.TaxAuthority = &TaxDataTaxAuthority{ID: params.TaxAuthorityID, Name: params.TaxAuthorityName}
	}
	td.ReportedTransaction.ReportedDocument.UUID = TaxDataInvoiceUUID(name)
	if profile.TransmissionUUID {
		td.ReportedTransaction.TransmissionUUID = params.TransmissionUUID
	}
	return td, nil
}

// taxDataReporter provides the endpoint of the reporting party: the one in
// the parameters, else the seller's for the sender role and the buyer's for
// the receiver role.
func taxDataReporter(inv *Invoice, profile TaxDataProfile, params TaxDataParams) (*EndpointID, error) {
	if params.ReportingParty != nil && params.ReportingParty.Value != "" {
		e := *params.ReportingParty
		return &e, nil
	}
	reporter := partyEndpoint(inv.AccountingSupplierParty.Party)
	if params.ReporterRole == profile.ReceiverRole {
		reporter = partyEndpoint(inv.AccountingCustomerParty.Party)
	}
	if reporter == nil {
		return nil, fmt.Errorf("%w: the reporting party has no electronic address", ErrTaxDataParams)
	}
	return reporter, nil
}

// TaxDataTypeCodeOf provides the invoice or credit note type code (BT-3).
func TaxDataTypeCodeOf(inv *Invoice) string {
	return typeCodeOf(inv)
}

// schemeIDSPID is the ICD of the Peppol Service Provider Identifier Scheme.
const schemeIDSPID = "0242"

func checkTaxDataParams(profile TaxDataProfile, p TaxDataParams) error {
	var missing []string
	if !inList(p.TypeCode, profile.TypeCodes...) {
		missing = append(missing, "type code ("+strings.Join(profile.TypeCodes, ", ")+")")
	}
	if !inList(p.Scope, profile.Scopes...) {
		missing = append(missing, "scope ("+strings.Join(profile.Scopes, ", ")+")")
	}
	if p.ReporterRole == "" || !inList(p.ReporterRole, profile.SenderRole, profile.ReceiverRole) {
		missing = append(missing, "reporter role ("+profile.SenderRole+", "+profile.ReceiverRole+")")
	}
	if profile.TaxAuthority && p.TaxAuthorityID == "" {
		missing = append(missing, "tax authority ID")
	}
	if p.ReceiverEndpoint == "" {
		missing = append(missing, "receiver endpoint")
	}
	if p.Representative == "" {
		missing = append(missing, "representative SPID")
	}
	if profile.TransmissionUUID && p.TransmissionUUID == "" {
		missing = append(missing, "transmission UUID")
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", ErrTaxDataParams, strings.Join(missing, ", "))
	}
	return nil
}

func inList(v string, list ...string) bool {
	for _, s := range list {
		if v == s {
			return true
		}
	}
	return false
}
