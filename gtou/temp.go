package gtou

import (
	"github.com/invopop/gobl/bill"
)

func (c *Conversor) createOrderReference(inv *bill.Invoice) error {
	orderReference := &OrderReference{
		ID: inv.OrderReference,
	}
	c.doc.OrderReference = orderReference
	return nil
}

func (c *Conversor) createPayeeParty(inv *bill.Invoice) error {
	party := &Party{
		PartyIdentification: []Identification{
			{ID: inv.Payee.ID},
		},
		PartyName: &PartyName{Name: inv.Payee.Name},
		PartyLegalEntity: &PartyLegalEntity{
			CompanyID: inv.Payee.CompanyID,
		},
	}
	c.doc.PayeeParty = party
	return nil
}

func (c *Conversor) createPaymentMeans(inv *bill.Invoice) error {
	paymentMeansDoc := []PaymentMeans{
		{
			PaymentMeansCode: IDType{Value: inv.PaymentMeans.Code},
			PaymentID:        inv.PaymentMeans.ID,
			PayeeFinancialAccount: &FinancialAccount{
				ID: inv.PaymentMeans.AccountID,
				FinancialInstitutionBranch: &Branch{
					ID: inv.PaymentMeans.BranchID,
				},
			},
		},
	}
	c.doc.PaymentMeans = paymentMeansDoc
	return nil
}

func (c *Conversor) createPaymentTerms(inv *bill.Invoice) error {
	paymentTermsDoc := []PaymentTerms{
		{
			Note: []string{inv.PaymentTerms.Note},
		},
	}
	c.doc.PaymentTerms = paymentTermsDoc
	return nil
}

func (c *Conversor) createAllowanceCharges(inv *bill.Invoice) error {
	var charges []AllowanceCharge
	for _, ac := range inv.AllowanceCharges {
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
	c.doc.AllowanceCharge = charges
	return nil
}

func (c *Conversor) createTaxTotals(inv *bill.Invoice) error {
	var totals []TaxTotal
	for _, tt := range inv.Tax.List {
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
	c.doc.TaxTotal = totals
	return nil
}

func (c *Conversor) createMonetaryTotal(inv *bill.Invoice) error {
	monetaryTotalDoc := MonetaryTotal{
		LineExtensionAmount:  Amount{CurrencyID: inv.Currency, Value: inv.MonetaryTotal.LineExtensionAmount},
		TaxExclusiveAmount:   Amount{CurrencyID: inv.Currency, Value: inv.MonetaryTotal.TaxExclusiveAmount},
		TaxInclusiveAmount:   Amount{CurrencyID: inv.Currency, Value: inv.MonetaryTotal.TaxInclusiveAmount},
		AllowanceTotalAmount: Amount{CurrencyID: inv.Currency, Value: inv.MonetaryTotal.AllowanceTotalAmount},
		ChargeTotalAmount:    Amount{CurrencyID: inv.Currency, Value: inv.MonetaryTotal.ChargeTotalAmount},
		PrepaidAmount:        Amount{CurrencyID: inv.Currency, Value: inv.MonetaryTotal.PrepaidAmount},
		PayableAmount:        Amount{CurrencyID: inv.Currency, Value: inv.MonetaryTotal.PayableAmount},
	}
	c.doc.LegalMonetaryTotal = monetaryTotalDoc
	return nil
}
