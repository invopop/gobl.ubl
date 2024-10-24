package gtou

import (
	"github.com/invopop/gobl/bill"
)

func (c *Conversor) createInvoiceLines(inv *bill.Invoice) error {
	var invoiceLines []InvoiceLine

	for _, line := range inv.Lines {
		invoiceLines = append(invoiceLines, InvoiceLine{
			ID:               line.ID,
			Note:             line.Notes,
			InvoicedQuantity: &Quantity{UnitCode: line.UnitCode, Value: line.Quantity},
			LineExtensionAmount: Amount{
				CurrencyID: line.Currency,
				Value:      line.LineExtensionAmount,
			},
			AccountingCost: line.AccountingCost,
			InvoicePeriod: &Period{
				StartDate: line.Period.StartDate.Format("2006-01-02"),
				EndDate:   line.Period.EndDate.Format("2006-01-02"),
			},
			OrderLineReference: &OrderLineReference{
				LineID: line.OrderLineID,
			},
			AllowanceCharge: createAllowanceCharges(line.AllowanceCharges),
			Item: &Item{
				Description: line.ItemDescription,
				Name:        line.ItemName,
				SellersItemIdentification: &ItemIdentification{
					ID: &IDType{Value: line.SellersItemID},
				},
				StandardItemIdentification: &ItemIdentification{
					ID: &IDType{Value: line.StandardItemID},
				},
				OriginCountry: &Country{
					IdentificationCode: line.OriginCountryCode,
				},
				CommodityClassification: &[]CommodityClassification{
					{ItemClassificationCode: &IDType{Value: line.CommodityCode}},
				},
				ClassifiedTaxCategory: &ClassifiedTaxCategory{
					ID:      line.TaxCategoryID,
					Percent: line.TaxPercent,
					TaxScheme: TaxScheme{
						ID: line.TaxSchemeID,
					},
				},
				AdditionalItemProperty: &[]AdditionalItemProperty{
					{Name: "Color", Value: line.ItemColor},
				},
			},
			Price: &Price{
				PriceAmount: Amount{
					CurrencyID: line.Currency,
					Value:      line.PriceAmount,
				},
				BaseAmount: &Amount{
					CurrencyID: line.Currency,
					Value:      line.BaseAmount,
				},
				AllowanceCharge: &AllowanceCharge{
					ChargeIndicator: line.PriceAllowanceCharge.ChargeIndicator,
					Amount: Amount{
						CurrencyID: line.Currency,
						Value:      line.PriceAllowanceCharge.Amount,
					},
				},
			},
		})
	}
	return nil
}
