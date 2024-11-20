package document

// SupplierParty represents the supplier party in a transaction
type SupplierParty struct {
	Party Party `xml:"cac:Party"`
}

// CustomerParty represents the customer party in a transaction
type CustomerParty struct {
	Party Party `xml:"cac:Party"`
}

// Party represents a party involved in a transaction
type Party struct {
	EndpointID          *EndpointID       `xml:"cbc:EndpointID"`
	PartyIdentification *Identification   `xml:"cac:PartyIdentification"`
	PartyName           *PartyName        `xml:"cac:PartyName"`
	PostalAddress       *PostalAddress    `xml:"cac:PostalAddress"`
	PartyTaxScheme      []PartyTaxScheme  `xml:"cac:PartyTaxScheme"`
	PartyLegalEntity    *PartyLegalEntity `xml:"cac:PartyLegalEntity"`
	Contact             *Contact          `xml:"cac:Contact"`
}

// EndpointID represents an endpoint identifier
type EndpointID struct {
	SchemeID string `xml:"schemeID,attr"`
	Value    string `xml:",chardata"`
}

// Identification represents an identification
type Identification struct {
	ID *IDType `xml:"cbc:ID"`
}

// PartyName represents the name of a party
type PartyName struct {
	Name string `xml:"cbc:Name"`
}

// PostalAddress represents a postal address
type PostalAddress struct {
	StreetName           *string             `xml:"cbc:StreetName"`
	AdditionalStreetName *string             `xml:"cbc:AdditionalStreetName"`
	CityName             *string             `xml:"cbc:CityName"`
	PostalZone           *string             `xml:"cbc:PostalZone"`
	CountrySubentity     *string             `xml:"cbc:CountrySubentity"`
	AddressLine          []AddressLine       `xml:"cac:AddressLine"`
	Country              *Country            `xml:"cac:Country"`
	LocationCoordinate   *LocationCoordinate `xml:"cac:LocationCoordinate"`
}

// LocationCoordinate represents a location coordinate
type LocationCoordinate struct {
	LatitudeDegreesMeasure  *string `xml:"cbc:LatitudeDegreesMeasure"`
	LatitudeMinutesMeasure  *string `xml:"cbc:LatitudeMinutesMeasure"`
	LongitudeDegreesMeasure *string `xml:"cbc:LongitudeDegreesMeasure"`
	LongitudeMinutesMeasure *string `xml:"cbc:LongitudeMinutesMeasure"`
}

// AddressLine represents a line in an address
type AddressLine struct {
	Line string `xml:"cbc:Line"`
}

// Country represents a country
type Country struct {
	IdentificationCode string `xml:"cbc:IdentificationCode"`
}

// PartyTaxScheme represents a party's tax scheme
type PartyTaxScheme struct {
	CompanyID *string    `xml:"cbc:CompanyID"`
	TaxScheme *TaxScheme `xml:"cac:TaxScheme"`
}

// TaxScheme represents a tax scheme
type TaxScheme struct {
	ID string `xml:"cbc:ID"`
}

// PartyLegalEntity represents the legal entity of a party
type PartyLegalEntity struct {
	RegistrationName *string `xml:"cbc:RegistrationName"`
	CompanyID        *IDType `xml:"cbc:CompanyID"`
	CompanyLegalForm *string `xml:"cbc:CompanyLegalForm"`
}

// Contact represents contact information
type Contact struct {
	Name           *string `xml:"cbc:Name"`
	Telephone      *string `xml:"cbc:Telephone"`
	ElectronicMail *string `xml:"cbc:ElectronicMail"`
}
