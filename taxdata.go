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
	TaxAuthority            TaxDataTaxAuthority   `xml:"pxs:TaxAuthority"`
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

// TaxDataDocument is the extract of the reported invoice or credit note (TDG-02).
type TaxDataDocument struct {
	CustomizationID         string              `xml:"cbc:CustomizationID"`
	ProfileID               string              `xml:"cbc:ProfileID"`
	ID                      string              `xml:"cbc:ID"`
	UUID                    string              `xml:"cbc:UUID"`
	IssueDate               string              `xml:"cbc:IssueDate"`
	IssueTime               string              `xml:"cbc:IssueTime,omitempty"`
	DocumentTypeCode        string              `xml:"pxs:DocumentTypeCode"`
	Note                    string              `xml:"cbc:Note,omitempty"`
	TaxPointDate            string              `xml:"cbc:TaxPointDate,omitempty"`
	DocumentCurrencyCode    string              `xml:"cbc:DocumentCurrencyCode"`
	TaxCurrencyCode         string              `xml:"cbc:TaxCurrencyCode,omitempty"`
	InvoicePeriod           *Period             `xml:"cac:InvoicePeriod,omitempty"`
	BillingReference        []*BillingReference `xml:"cac:BillingReference,omitempty"`
	AccountingSupplierParty SupplierParty       `xml:"cac:AccountingSupplierParty"`
	AccountingCustomerParty CustomerParty       `xml:"cac:AccountingCustomerParty"`
	TaxRepresentativeParty  *Party              `xml:"cac:TaxRepresentativeParty,omitempty"`
	Delivery                *Delivery           `xml:"cac:Delivery,omitempty"`
	PaymentMeans            []PaymentMeans      `xml:"cac:PaymentMeans,omitempty"`
	AllowanceCharge         []AllowanceCharge   `xml:"cac:AllowanceCharge,omitempty"`
	TaxTotal                []TaxTotal          `xml:"cac:TaxTotal,omitempty"`
	MonetaryTotal           MonetaryTotal       `xml:"pxs:MonetaryTotal"`
	DocumentLines           []TaxDataLine       `xml:"pxs:DocumentLine"`
}

// TaxDataLine is one line of the reported document (BG-25).
type TaxDataLine struct {
	ID                  string             `xml:"cbc:ID"`
	Note                string             `xml:"cbc:Note,omitempty"`
	InvoicedQuantity    *Quantity          `xml:"cbc:InvoicedQuantity,omitempty"`
	LineExtensionAmount Amount             `xml:"cbc:LineExtensionAmount"`
	InvoicePeriod       *Period            `xml:"cac:InvoicePeriod,omitempty"`
	AllowanceCharge     []*AllowanceCharge `xml:"cac:AllowanceCharge,omitempty"`
	Item                *Item              `xml:"cac:Item,omitempty"`
	Price               *Price             `xml:"cac:Price,omitempty"`
}

// BuildTaxData builds the Tax Data Document that reports a UBL invoice or
// credit note with the profile. The data is the UBL XML as it was sent or
// received, for example by the receiver's provider (C3).
func BuildTaxData(data []byte, profile TaxDataProfile, params TaxDataParams) (*TaxData, error) {
	if err := checkTaxDataParams(profile, params); err != nil {
		return nil, err
	}
	doc, err := Decode(data)
	if err != nil {
		return nil, err
	}
	inv, ok := doc.(*Invoice)
	if !ok {
		return nil, ErrUnsupportedDocumentType
	}
	return NewTaxData(inv, profile, params)
}

// ExportTaxData builds the Tax Data Document that reports the GOBL invoice in
// the envelope with the profile. The invoice is first exported to UBL, Peppol
// BIS Billing 3.0 unless a WithFormat option says otherwise, so the report
// matches the invoice that is sent. Options are passed on to that export.
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
	return NewTaxData(inv, profile, params)
}

// NewTaxData builds the ViDA Tax Data Document, or the document of a profile
// based on ViDA, that reports a decoded UBL invoice or credit note.
func NewTaxData(inv *Invoice, profile TaxDataProfile, params TaxDataParams) (*TaxData, error) {
	if err := checkTaxDataParams(profile, params); err != nil {
		return nil, err
	}
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
		TaxAuthority: TaxDataTaxAuthority{
			ID:   params.TaxAuthorityID,
			Name: params.TaxAuthorityName,
		},
		ReportingParty: TaxDataEndpointParty{EndpointID: *reporter},
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

// taxDataDocument copies the parts of the invoice that a TDD carries, and
// nothing else, as the TDD Schematron rejects any other element.
func taxDataDocument(inv *Invoice, profile TaxDataProfile) TaxDataDocument {
	doc := TaxDataDocument{
		CustomizationID:      inv.CustomizationID,
		ID:                   inv.ID,
		IssueDate:            inv.IssueDate,
		IssueTime:            inv.IssueTime,
		TaxPointDate:         inv.TaxPointDate,
		DocumentCurrencyCode: inv.DocumentCurrencyCode,
		TaxCurrencyCode:      inv.TaxCurrencyCode,
		AccountingSupplierParty: SupplierParty{
			Party: taxDataParty(inv.AccountingSupplierParty.Party, false),
		},
		AccountingCustomerParty: CustomerParty{
			Party: taxDataParty(inv.AccountingCustomerParty.Party, profile.BuyerName),
		},
		MonetaryTotal: inv.LegalMonetaryTotal,
	}
	if inv.ProfileID != nil {
		doc.ProfileID = inv.ProfileID.Value
	}
	if tc := typeCodeOf(inv); tc != "" {
		doc.DocumentTypeCode = tc
	}
	// The TDD allows a single note.
	if len(inv.Note) > 0 {
		doc.Note = strings.Join(inv.Note, "\n\n")
	}
	if len(inv.InvoicePeriod) > 0 {
		p := inv.InvoicePeriod[0]
		doc.InvoicePeriod = &p
	}
	for _, br := range inv.BillingReference {
		if br == nil || br.InvoiceDocumentReference == nil {
			continue
		}
		doc.BillingReference = append(doc.BillingReference, &BillingReference{
			InvoiceDocumentReference: &Reference{
				ID:        IDType{Value: br.InvoiceDocumentReference.ID.Value},
				IssueDate: br.InvoiceDocumentReference.IssueDate,
			},
		})
	}
	if inv.TaxRepresentativeParty != nil {
		doc.TaxRepresentativeParty = taxDataParty(inv.TaxRepresentativeParty, false)
	}
	for _, d := range inv.Delivery {
		if d != nil && d.ActualDeliveryDate != nil {
			doc.Delivery = &Delivery{ActualDeliveryDate: d.ActualDeliveryDate}
			break
		}
	}
	for _, pm := range inv.PaymentMeans {
		doc.PaymentMeans = append(doc.PaymentMeans, taxDataPaymentMeans(pm))
	}
	for _, ac := range inv.AllowanceCharge {
		ac := ac
		ac.TaxCategory = taxDataCategories(ac.TaxCategory)
		doc.AllowanceCharge = append(doc.AllowanceCharge, ac)
	}
	for _, tt := range inv.TaxTotal {
		doc.TaxTotal = append(doc.TaxTotal, taxDataTaxTotal(tt))
	}
	lines := inv.InvoiceLines
	if len(inv.CreditNoteLines) > 0 {
		lines = inv.CreditNoteLines
	}
	for _, l := range lines {
		doc.DocumentLines = append(doc.DocumentLines, taxDataLine(l))
	}
	return doc
}

func typeCodeOf(inv *Invoice) string {
	if inv.InvoiceTypeCode != nil {
		return inv.InvoiceTypeCode.Value
	}
	if inv.CreditNoteTypeCode != nil {
		return inv.CreditNoteTypeCode.Value
	}
	return ""
}

func partyEndpoint(p *Party) *EndpointID {
	if p == nil || p.EndpointID == nil || p.EndpointID.Value == "" {
		return nil
	}
	e := *p.EndpointID
	return &e
}

// taxDataParty keeps the country, the VAT identifier and, when asked, the
// registration name.
func taxDataParty(p *Party, withName bool) *Party {
	out := new(Party)
	if p == nil {
		return out
	}
	if p.PostalAddress != nil && p.PostalAddress.Country != nil {
		out.PostalAddress = &PostalAddress{Country: &Country{IdentificationCode: p.PostalAddress.Country.IdentificationCode}}
	}
	if pts := vatTaxScheme(p); pts != nil {
		out.PartyTaxScheme = []PartyTaxScheme{{
			CompanyID: &IDType{Value: pts.CompanyID.Value},
			TaxScheme: &TaxScheme{ID: IDType{Value: "VAT"}},
		}}
	}
	if withName && p.PartyLegalEntity != nil && p.PartyLegalEntity.RegistrationName != nil {
		out.PartyLegalEntity = &PartyLegalEntity{RegistrationName: p.PartyLegalEntity.RegistrationName}
	}
	return out
}

// vatTaxScheme provides the party's VAT registration, if any.
func vatTaxScheme(p *Party) *PartyTaxScheme {
	if p == nil {
		return nil
	}
	for i := range p.PartyTaxScheme {
		pts := &p.PartyTaxScheme[i]
		if pts.CompanyID != nil && pts.TaxScheme != nil && pts.TaxScheme.ID.Value == "VAT" {
			return pts
		}
	}
	return nil
}

func taxDataPaymentMeans(pm PaymentMeans) PaymentMeans {
	out := PaymentMeans{
		PaymentMeansCode: pm.PaymentMeansCode,
		PaymentID:        pm.PaymentID,
	}
	if pm.CardAccount != nil {
		out.CardAccount = &CardAccount{
			PrimaryAccountNumberID: pm.CardAccount.PrimaryAccountNumberID,
			NetworkID:              pm.CardAccount.NetworkID,
			HolderName:             pm.CardAccount.HolderName,
		}
	}
	if fa := pm.PayeeFinancialAccount; fa != nil {
		out.PayeeFinancialAccount = &FinancialAccount{ID: fa.ID}
		if fa.FinancialInstitutionBranch != nil && fa.FinancialInstitutionBranch.ID != nil {
			out.PayeeFinancialAccount.FinancialInstitutionBranch = &Branch{ID: fa.FinancialInstitutionBranch.ID}
		}
	}
	if m := pm.PaymentMandate; m != nil {
		out.PaymentMandate = &PaymentMandate{ID: m.ID}
		if m.PayerFinancialAccount != nil {
			out.PaymentMandate.PayerFinancialAccount = &FinancialAccount{ID: m.PayerFinancialAccount.ID}
		}
	}
	return out
}

// taxDataCategory keeps the identifier, the rate and the scheme, and the
// exemption reason when asked.
func taxDataCategory(c *TaxCategory, withExemption bool) *TaxCategory {
	if c == nil {
		return nil
	}
	out := &TaxCategory{ID: c.ID, Percent: c.Percent}
	if withExemption {
		out.TaxExemptionReasonCode = c.TaxExemptionReasonCode
		out.TaxExemptionReason = c.TaxExemptionReason
	}
	if c.TaxScheme != nil {
		out.TaxScheme = &TaxScheme{ID: IDType{Value: c.TaxScheme.ID.Value}}
	}
	return out
}

func taxDataCategories(list []*TaxCategory) []*TaxCategory {
	var out []*TaxCategory
	for _, c := range list {
		if c != nil {
			out = append(out, taxDataCategory(c, false))
		}
	}
	return out
}

func taxDataTaxTotal(tt TaxTotal) TaxTotal {
	out := TaxTotal{TaxAmount: tt.TaxAmount}
	for _, st := range tt.TaxSubtotal {
		out.TaxSubtotal = append(out.TaxSubtotal, TaxSubtotal{
			TaxableAmount: st.TaxableAmount,
			TaxAmount:     st.TaxAmount,
			TaxCategory:   *taxDataCategory(&st.TaxCategory, true),
		})
	}
	return out
}

func taxDataLine(l InvoiceLine) TaxDataLine {
	out := TaxDataLine{
		ID:                  l.ID,
		InvoicedQuantity:    l.InvoicedQuantity,
		LineExtensionAmount: l.LineExtensionAmount,
	}
	if out.InvoicedQuantity == nil {
		// Credit notes carry the credited quantity, reported as invoiced.
		out.InvoicedQuantity = l.CreditedQuantity
	}
	if len(l.Note) > 0 {
		out.Note = strings.Join(l.Note, "\n\n")
	}
	if l.InvoicePeriod != nil {
		out.InvoicePeriod = &Period{StartDate: l.InvoicePeriod.StartDate, EndDate: l.InvoicePeriod.EndDate}
	}
	for _, ac := range l.AllowanceCharge {
		if ac == nil {
			continue
		}
		c := *ac
		// Line allowances and charges carry no tax category in the TDD.
		c.TaxCategory = nil
		out.AllowanceCharge = append(out.AllowanceCharge, &c)
	}
	if it := l.Item; it != nil {
		out.Item = &Item{
			Description:             it.Description,
			Name:                    it.Name,
			CommodityClassification: it.CommodityClassification,
		}
		if ct := it.ClassifiedTaxCategory; ct != nil {
			out.Item.ClassifiedTaxCategory = &ClassifiedTaxCategory{ID: ct.ID, Percent: ct.Percent}
			if ct.TaxScheme != nil {
				out.Item.ClassifiedTaxCategory.TaxScheme = &TaxScheme{ID: IDType{Value: ct.TaxScheme.ID.Value}}
			}
		}
	}
	if p := l.Price; p != nil {
		out.Price = &Price{PriceAmount: p.PriceAmount, BaseQuantity: p.BaseQuantity}
	}
	return out
}

// TaxDataInvoiceUUIDName joins the values that identify an invoice into the
// name hashed into the invoice UUID: each value is trimmed of leading and
// trailing whitespace (space, tab, CR, LF) and kept as it is otherwise, and the
// values are joined with one space. Any empty value makes the invoice not
// reportable.
func TaxDataInvoiceUUIDName(values ...string) (string, error) {
	out := make([]string, len(values))
	for i, v := range values {
		v = strings.Trim(v, " \t\r\n")
		if v == "" {
			return "", ErrTaxDataNotReportable
		}
		out[i] = v
	}
	return strings.Join(out, " "), nil
}

// TaxDataInvoiceUUID calculates the invoice UUID (TDT-017) from the name.
func TaxDataInvoiceUUID(name string) string {
	return uuid.V5(uuid.MustParse(TaxDataInvoiceUUIDNamespace), []byte(name)).String()
}

// vidaInvoiceUUIDName uses the seller VAT identifier (BT-31), the type code
// (BT-3), the invoice number (BT-1) and the issue date (BT-2). An invoice
// without a seller VAT identifier is not reportable (ViDA TDD BIS 2.2.6).
func vidaInvoiceUUIDName(inv *Invoice) (string, error) {
	vat := ""
	if pts := vatTaxScheme(inv.AccountingSupplierParty.Party); pts != nil {
		vat = pts.CompanyID.Value
	}
	return TaxDataInvoiceUUIDName(vat, typeCodeOf(inv), inv.ID, inv.IssueDate)
}
