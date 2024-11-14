package document

// PaymentMeans represents the means of payment
type PaymentMeans struct {
	PaymentMeansCode      IDType            `xml:"cbc:PaymentMeansCode"`
	PaymentID             *string           `xml:"cbc:PaymentID"`
	PayeeFinancialAccount *FinancialAccount `xml:"cac:PayeeFinancialAccount"`
	PayerFinancialAccount *FinancialAccount `xml:"cac:PayerFinancialAccount"`
	CardAccount           *CardAccount      `xml:"cac:CardAccount"`
	InstructionID         *string           `xml:"cbc:InstructionID"`
	InstructionNote       []string          `xml:"cbc:InstructionNote"`
	PaymentMandate        *PaymentMandate   `xml:"cac:PaymentMandate"`
}

// PaymentMandate represents a payment mandate
type PaymentMandate struct {
	ID                    IDType            `xml:"cbc:ID"`
	PayerFinancialAccount *FinancialAccount `xml:"cac:PayerFinancialAccount"`
}

// CardAccount represents a card account
type CardAccount struct {
	PrimaryAccountNumberID *string `xml:"cbc:PrimaryAccountNumberID"`
	NetworkID              *string `xml:"cbc:NetworkID"`
	HolderName             *string `xml:"cbc:HolderName"`
}

// FinancialAccount represents a financial account
type FinancialAccount struct {
	ID                         *string `xml:"cbc:ID"`
	Name                       *string `xml:"cbc:Name"`
	FinancialInstitutionBranch *Branch `xml:"cac:FinancialInstitutionBranch"`
	AccountTypeCode            *string `xml:"cbc:AccountTypeCode"`
}

// Branch represents a branch of a financial institution
type Branch struct {
	ID   *string `xml:"cbc:ID"`
	Name *string `xml:"cbc:Name"`
}

// PaymentTerms represents the terms of payment
type PaymentTerms struct {
	Note           []string `xml:"cbc:Note"`
	Amount         *Amount  `xml:"cbc:Amount"`
	PaymentPercent *string  `xml:"cbc:PaymentPercent"`
	PaymentDueDate *string  `xml:"cbc:PaymentDueDate"`
}

// PrepaidPayment represents a prepaid payment
type PrepaidPayment struct {
	ID            string  `xml:"cbc:ID"`
	PaidAmount    *Amount `xml:"cbc:PaidAmount"`
	ReceivedDate  *string `xml:"cbc:ReceivedDate"`
	InstructionID *string `xml:"cbc:InstructionID"`
}
