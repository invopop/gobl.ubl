package utog

import (
	"fmt"
	"strings"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

func (c *Conversor) getLines(doc *Document) error {
	items := doc.InvoiceLine

	lines := make([]*bill.Line, 0, len(items))

	for _, item := range items {
		price, err := num.AmountFromString(item.Price.PriceAmount.Value)
		if err != nil {
			return err
		}
		line := &bill.Line{
			Quantity: num.MakeAmount(1, 0),
			Item: &org.Item{
				Name:  *item.Item.Name,
				Price: price,
			},
			Taxes: tax.Set{
				{
					Rate:     FindTaxKey(item.Item.ClassifiedTaxCategory.ID),
					Category: cbc.Code(*item.Item.ClassifiedTaxCategory.TaxScheme.ID),
				},
			},
		}

		if item.InvoicedQuantity != nil {
			line.Quantity, err = num.AmountFromString(item.InvoicedQuantity.Value)
			if err != nil {
				return err
			}
		}

		if item.InvoicedQuantity.UnitCode != "" {
			line.Item.Unit = UnitFromUNECE(cbc.Code(item.InvoicedQuantity.UnitCode))
		}

		if item.Item.SellersItemIdentification != nil && item.Item.SellersItemIdentification.ID != nil {
			line.Item.Ref = *item.Item.SellersItemIdentification.ID
		}

		if item.Item.BuyersItemIdentification != nil && item.Item.BuyersItemIdentification.ID != nil {
			if line.Item.Identities == nil {
				line.Item.Identities = make([]*org.Identity, 0)
			}
			line.Item.Identities = append(line.Item.Identities, &org.Identity{
				Code: cbc.Code(*item.Item.BuyersItemIdentification.ID),
			})
		}

		if item.Item.StandardItemIdentification != nil && item.Item.StandardItemIdentification.ID != nil {
			if line.Item.Identities == nil {
				line.Item.Identities = make([]*org.Identity, 0)
			}
			line.Item.Identities = append(line.Item.Identities, &org.Identity{
				Code: cbc.Code(*item.Item.StandardItemIdentification.ID),
			})
		}

		if item.Item.Description != nil {
			line.Item.Description = *item.Item.Description
		}

		if item.Item.OriginCountry != nil {
			line.Item.Origin = l10n.ISOCountryCode(item.Item.OriginCountry.IdentificationCode)
		}

		// if len(item.AssociatedDocumentLineDocument.IncludedNote) > 0 {
		// 	line.Notes = make([]*cbc.Note, 0, len(item.AssociatedDocumentLineDocument.IncludedNote))
		// 	for _, note := range item.AssociatedDocumentLineDocument.IncludedNote {
		// 		n := &cbc.Note{}
		// 		if note.Content != "" {
		// 			n.Text = note.Content
		// 		}
		// 		if note.ContentCode != "" {
		// 			n.Code = note.ContentCode
		// 		}
		// 		line.Notes = append(line.Notes, n)
		// 	}
		// }

		if item.Item.ClassifiedTaxCategory != nil && item.Item.ClassifiedTaxCategory.Percent != "" {
			percentStr := item.Item.ClassifiedTaxCategory.Percent
			if !strings.HasSuffix(percentStr, "%") {
				percentStr += "%"
			}
			percent, _ := num.PercentageFromString(percentStr)
			if line.Taxes == nil {
				line.Taxes = make([]*tax.Combo, 1)
				line.Taxes[0] = &tax.Combo{}
			}
			line.Taxes[0].Percent = &percent
		}

		if item.AllowanceCharge != nil {
			line, err = parseLineCharges(*item.AllowanceCharge, line)
			if err != nil {
				return err
			}
		}

		lines = append(lines, line)
	}
	c.inv.Lines = lines
	fmt.Println(c.inv.Lines)
	return nil
}

func parseLineCharges(allowances []AllowanceCharge, line *bill.Line) (*bill.Line, error) {
	for _, allowanceCharge := range allowances {
		amount, err := num.AmountFromString(allowanceCharge.Amount.Value)
		if err != nil {
			return nil, err
		}
		if allowanceCharge.ChargeIndicator {
			charge := &bill.LineCharge{
				Amount: amount,
			}
			if allowanceCharge.AllowanceChargeReasonCode != nil {
				charge.Code = *allowanceCharge.AllowanceChargeReasonCode
			}
			if allowanceCharge.AllowanceChargeReason != nil {
				charge.Reason = *allowanceCharge.AllowanceChargeReason
			}
			if allowanceCharge.MultiplierFactorNumeric != nil {
				if !strings.HasSuffix(*allowanceCharge.MultiplierFactorNumeric, "%") {
					*allowanceCharge.MultiplierFactorNumeric += "%"
				}
				percent, err := num.PercentageFromString(*allowanceCharge.MultiplierFactorNumeric)
				if err != nil {
					return nil, err
				}
				charge.Percent = &percent
			}
			if line.Charges == nil {
				line.Charges = make([]*bill.LineCharge, 0)
			}
			line.Charges = append(line.Charges, charge)
		} else {
			discount := &bill.LineDiscount{
				Amount: amount,
			}
			if allowanceCharge.AllowanceChargeReasonCode != nil {
				discount.Code = *allowanceCharge.AllowanceChargeReasonCode
			}
			if allowanceCharge.AllowanceChargeReason != nil {
				discount.Reason = *allowanceCharge.AllowanceChargeReason
			}
			if allowanceCharge.MultiplierFactorNumeric != nil {
				if !strings.HasSuffix(*allowanceCharge.MultiplierFactorNumeric, "%") {
					*allowanceCharge.MultiplierFactorNumeric += "%"
				}
				percent, err := num.PercentageFromString(*allowanceCharge.MultiplierFactorNumeric)
				if err != nil {
					return nil, err
				}
				discount.Percent = &percent
			}
			if line.Discounts == nil {
				line.Discounts = make([]*bill.LineDiscount, 0)
			}
			line.Discounts = append(line.Discounts, discount)
		}
	}
	return line, nil
}
