// Package gtou provides a conversor from GOBL to UBL.
package gtou

import (
	"encoding/xml"
	"fmt"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
)

// Conversor is a struct that contains the necessary elements to convert between GOBL and UBL
type Conversor struct {
	doc *Document
}

// NewConversor creates a new Conversor instance
func NewConversor() *Conversor {
	c := new(Conversor)
	c.doc = new(Document)
	return c
}

// GetDocument returns the document from the conversor
func (c *Conversor) GetDocument() *Document {
	return c.doc
}

// ConvertToUBL converts a GOBL envelope into a UBL document
func (c *Conversor) ConvertToUBL(env *gobl.Envelope) (*Document, error) {
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, fmt.Errorf("invalid type %T", env.Document)
	}

	// Create the UBL document
	doc := &Document{
		CACNamespace:         CAC,
		CBCNamespace:         CBC,
		UBLNamespace:         UBL,
		CustomizationID:      "urn:cen.eu:en16931:2017",
		ProfileID:            "Invoicing on purchase order",
		ID:                   inv.ID,
		IssueDate:            inv.IssueDate.Format("2006-01-02"),
		DueDate:              inv.DueDate.Format("2006-01-02"),
		InvoiceTypeCode:      "380",
		Note:                 []string{"Ordered in our booth at the convention"},
		DocumentCurrencyCode: inv.Currency,
		AccountingCost:       "Project cost code 123",
		InvoicePeriod:        []Period{createInvoicePeriod(inv)},
		OrderReference:       createOrderReference(inv),
		ContractDocumentReference: []DocumentReference{
			{ID: "Contract321"},
		},
		AdditionalDocumentReference: []DocumentReference{
			{ID: "Doc1", DocumentDescription: "Timesheet"},
			{ID: "Doc2", DocumentDescription: "EHF specification"},
		},
		AccountingSupplierParty: createSupplierParty(inv.Supplier),
		// AccountingCustomerParty: createCustomerParty(inv.Customer),
		PayeeParty:             createPayeeParty(inv.Payee),
		TaxRepresentativeParty: createTaxRepresentativeParty(inv.TaxRepresentative),
		// Delivery:                createDelivery(inv.Delivery),
		PaymentMeans:       createPaymentMeans(inv.PaymentMeans),
		PaymentTerms:       createPaymentTerms(inv.PaymentTerms),
		AllowanceCharge:    createAllowanceCharges(inv.AllowanceCharges),
		TaxTotal:           createTaxTotals(inv.TaxTotals),
		LegalMonetaryTotal: createMonetaryTotal(inv.MonetaryTotal),
		InvoiceLine:        createInvoiceLines(inv.Lines),
	}

	err := c.createCustomerParty(inv.Customer)
	if err != nil {
		return nil, err
	}

	err = c.createDelivery(inv.Delivery)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

func createInvoicePeriod(inv *bill.Invoice) Period {
	return Period{
		StartDate: inv.Period.StartDate.Format("2006-01-02"),
		EndDate:   inv.Period.EndDate.Format("2006-01-02"),
	}
}

func createOrderReference(inv *bill.Invoice) *OrderReference {
	return &OrderReference{
		ID: inv.OrderReference,
	}
}

func createPayeeParty(payee *bill.Payee) *Party {
	return &Party{
		PartyIdentification: []Identification{
			{ID: payee.ID},
		},
		PartyName: &PartyName{Name: payee.Name},
		PartyLegalEntity: &PartyLegalEntity{
			CompanyID: payee.CompanyID,
		},
	}
}

func createPaymentMeans(paymentMeans *bill.PaymentMeans) []PaymentMeans {
	return []PaymentMeans{
		{
			PaymentMeansCode: IDType{Value: paymentMeans.Code},
			PaymentID:        paymentMeans.ID,
			PayeeFinancialAccount: &FinancialAccount{
				ID: paymentMeans.AccountID,
				FinancialInstitutionBranch: &Branch{
					ID: paymentMeans.BranchID,
				},
			},
		},
	}
}

func createPaymentTerms(paymentTerms *bill.PaymentTerms) []PaymentTerms {
	return []PaymentTerms{
		{
			Note: []string{paymentTerms.Note},
		},
	}
}

func createAllowanceCharges(allowanceCharges []bill.AllowanceCharge) []AllowanceCharge {
	var charges []AllowanceCharge
	for _, ac := range allowanceCharges {
		charges = append(charges, AllowanceCharge{
			ChargeIndicator:           ac.ChargeIndicator,
			AllowanceChargeReasonCode: ac.ReasonCode,
			AllowanceChargeReason:     ac.Reason,
			Amount:                    Amount{CurrencyID: ac.Currency, Value: ac.Amount},
			TaxCategory: &TaxCategory{
				ID:      ac.TaxCategoryID,
				Percent: ac.TaxPercent,
				TaxScheme: &TaxScheme{
					ID: ac.TaxSchemeID,
				},
			},
		})
	}
	return charges
}

func createTaxTotals(taxTotals []bill.TaxTotal) []TaxTotal {
	var totals []TaxTotal
	for _, tt := range taxTotals {
		var subtotals []TaxSubtotal
		for _, st := range tt.Subtotals {
			subtotals = append(subtotals, TaxSubtotal{
				TaxableAmount: Amount{CurrencyID: st.Currency, Value: st.TaxableAmount},
				TaxAmount:     Amount{CurrencyID: st.Currency, Value: st.TaxAmount},
				TaxCategory: TaxCategory{
					ID:      st.TaxCategoryID,
					Percent: st.TaxPercent,
					TaxScheme: &TaxScheme{
						ID: st.TaxSchemeID,
					},
				},
			})
		}
		totals = append(totals, TaxTotal{
			TaxAmount:   Amount{CurrencyID: tt.Currency, Value: tt.TaxAmount},
			TaxSubtotal: subtotals,
		})
	}
	return totals
}

func createMonetaryTotal(monetaryTotal *bill.MonetaryTotal) MonetaryTotal {
	return MonetaryTotal{
		LineExtensionAmount:  Amount{CurrencyID: monetaryTotal.Currency, Value: monetaryTotal.LineExtensionAmount},
		TaxExclusiveAmount:   Amount{CurrencyID: monetaryTotal.Currency, Value: monetaryTotal.TaxExclusiveAmount},
		TaxInclusiveAmount:   Amount{CurrencyID: monetaryTotal.Currency, Value: monetaryTotal.TaxInclusiveAmount},
		AllowanceTotalAmount: Amount{CurrencyID: monetaryTotal.Currency, Value: monetaryTotal.AllowanceTotalAmount},
		ChargeTotalAmount:    Amount{CurrencyID: monetaryTotal.Currency, Value: monetaryTotal.ChargeTotalAmount},
		PrepaidAmount:        Amount{CurrencyID: monetaryTotal.Currency, Value: monetaryTotal.PrepaidAmount},
		PayableAmount:        Amount{CurrencyID: monetaryTotal.Currency, Value: monetaryTotal.PayableAmount},
	}
}

// Bytes returns the XML representation of the document in bytes
func (d *Document) Bytes() ([]byte, error) {
	bytes, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), bytes...), nil
}
