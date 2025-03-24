package ubl

import (
	"strconv"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/num"
)

// InvoiceLine represents a line item in an invoice
type InvoiceLine struct {
	ID                  string              `xml:"cbc:ID"`
	Note                []string            `xml:"cbc:Note"`
	InvoicedQuantity    *Quantity           `xml:"cbc:InvoicedQuantity"`
	LineExtensionAmount Amount              `xml:"cbc:LineExtensionAmount"`
	AccountingCost      *string             `xml:"cbc:AccountingCost"`
	InvoicePeriod       *Period             `xml:"cac:InvoicePeriod"`
	OrderLineReference  *OrderLineReference `xml:"cac:OrderLineReference"`
	AllowanceCharge     []*AllowanceCharge  `xml:"cac:AllowanceCharge"`
	Item                *Item               `xml:"cac:Item"`
	Price               *Price              `xml:"cac:Price"`
}

// Quantity represents a quantity with a unit code
type Quantity struct {
	UnitCode string `xml:"unitCode,attr"`
	Value    string `xml:",chardata"`
}

// OrderLineReference represents a reference to an order line
type OrderLineReference struct {
	LineID string `xml:"cbc:LineID"`
}

// Item represents an item in an invoice line
type Item struct {
	Description                *string                    `xml:"cbc:Description"`
	Name                       string                     `xml:"cbc:Name"`
	BuyersItemIdentification   *ItemIdentification        `xml:"cac:BuyersItemIdentification"`
	SellersItemIdentification  *ItemIdentification        `xml:"cac:SellersItemIdentification"`
	StandardItemIdentification *ItemIdentification        `xml:"cac:StandardItemIdentification"`
	OriginCountry              *Country                   `xml:"cac:OriginCountry"`
	CommodityClassification    *[]CommodityClassification `xml:"cac:CommodityClassification"`
	ClassifiedTaxCategory      *ClassifiedTaxCategory     `xml:"cac:ClassifiedTaxCategory"`
	AdditionalItemProperty     *[]AdditionalItemProperty  `xml:"cac:AdditionalItemProperty"`
}

// ItemIdentification represents an item identification
type ItemIdentification struct {
	ID *IDType `xml:"cbc:ID"`
}

// CommodityClassification represents a commodity classification
type CommodityClassification struct {
	ItemClassificationCode *IDType `xml:"cbc:ItemClassificationCode"`
}

// ClassifiedTaxCategory represents a classified tax category
type ClassifiedTaxCategory struct {
	ID        *string    `xml:"cbc:ID,omitempty"`
	Percent   *string    `xml:"cbc:Percent,omitempty"`
	TaxScheme *TaxScheme `xml:"cac:TaxScheme,omitempty"`
}

// AdditionalItemProperty represents an additional property of an item
type AdditionalItemProperty struct {
	Name  string `xml:"cbc:Name"`
	Value string `xml:"cbc:Value"`
}

// Price represents the price of an item
type Price struct {
	PriceAmount     Amount           `xml:"cbc:PriceAmount"`
	BaseAmount      *Amount          `xml:"cbc:BaseAmount,omitempty"`
	AllowanceCharge *AllowanceCharge `xml:"cac:AllowanceCharge,omitempty"`
}

func (out *Invoice) addLines(inv *bill.Invoice) {
	if len(inv.Lines) == 0 {
		return
	}

	var lines []InvoiceLine

	for _, l := range inv.Lines {
		ccy := l.Item.Currency.String()
		if ccy == "" {
			ccy = inv.Currency.String()
		}
		invLine := InvoiceLine{
			ID: strconv.Itoa(l.Index),

			LineExtensionAmount: Amount{
				CurrencyID: &ccy,
				Value:      l.Total.String(),
			},
		}

		if l.Quantity != (num.Amount{}) {
			invLine.InvoicedQuantity = &Quantity{
				Value: l.Quantity.String(),
			}
			if l.Item != nil && l.Item.Unit != "" {
				invLine.InvoicedQuantity.UnitCode = string(l.Item.Unit.UNECE())
			}
		}

		if len(l.Notes) > 0 {
			var notes []string
			for _, note := range l.Notes {
				if note.Key == "buyer-accounting-ref" {
					invLine.AccountingCost = &note.Text
				} else {
					notes = append(notes, note.Text)
				}
			}
			if len(notes) > 0 {
				invLine.Note = notes
			}
		}

		if len(l.Charges) > 0 || len(l.Discounts) > 0 {
			invLine.AllowanceCharge = makeLineCharges(l.Charges, l.Discounts, ccy)
		}

		if l.Item != nil {
			it := &Item{}

			if l.Item.Description != "" {
				d := l.Item.Description
				it.Description = &d
			}

			if l.Item.Name != "" {
				it.Name = l.Item.Name
			}

			if l.Item.Origin != "" {
				it.OriginCountry = &Country{
					IdentificationCode: l.Item.Origin.String(),
				}
			}

			if l.Item.Meta != nil {
				var properties []AdditionalItemProperty
				for key, value := range l.Item.Meta {
					properties = append(properties, AdditionalItemProperty{Name: key.String(), Value: value})
				}
				it.AdditionalItemProperty = &properties
			}

			if len(l.Taxes) > 0 && l.Taxes[0].Category != "" {
				it.ClassifiedTaxCategory = &ClassifiedTaxCategory{
					TaxScheme: &TaxScheme{
						ID: l.Taxes[0].Category.String(),
					},
				}
				if l.Taxes[0].Percent != nil {
					p := l.Taxes[0].Percent.StringWithoutSymbol()
					it.ClassifiedTaxCategory.Percent = &p
				}
				if l.Taxes[0].Ext != nil && l.Taxes[0].Ext[untdid.ExtKeyTaxCategory].String() != "" {
					rate := l.Taxes[0].Ext[untdid.ExtKeyTaxCategory].String()
					it.ClassifiedTaxCategory.ID = &rate
				}
			}

			if len(l.Item.Identities) > 0 {
				for _, id := range l.Item.Identities {
					if id.Ext == nil || id.Ext[iso.ExtKeySchemeID].String() == "" {
						continue
					}
					s := id.Ext[iso.ExtKeySchemeID].String()
					it.StandardItemIdentification = &ItemIdentification{
						ID: &IDType{
							SchemeID: &s,
							Value:    id.Code.String(),
						},
					}
				}
			}

			invLine.Item = it

			if l.Item.Price != nil {
				invLine.Price = &Price{
					PriceAmount: Amount{
						CurrencyID: &ccy,
						Value:      l.Item.Price.String(),
					},
				}
			}
		}

		lines = append(lines, invLine)
	}
	out.InvoiceLine = lines
}

func makeLineCharges(charges []*bill.LineCharge, discounts []*bill.LineDiscount, ccy string) []*AllowanceCharge {
	var allowanceCharges []*AllowanceCharge
	for _, ch := range charges {
		ac := &AllowanceCharge{
			ChargeIndicator: true,
			Amount: Amount{
				Value:      ch.Amount.String(),
				CurrencyID: &ccy,
			},
		}
		if ch.Ext != nil && ch.Ext[untdid.ExtKeyCharge].String() != "" {
			e := ch.Ext[untdid.ExtKeyCharge].String()
			ac.AllowanceChargeReasonCode = &e
		}
		if ch.Reason != "" {
			ac.AllowanceChargeReason = &ch.Reason
		}
		if ch.Percent != nil {
			p := ch.Percent.String()
			ac.MultiplierFactorNumeric = &p
		}
		allowanceCharges = append(allowanceCharges, ac)
	}
	for _, d := range discounts {
		ac := &AllowanceCharge{
			ChargeIndicator: false,
			Amount: Amount{
				Value:      d.Amount.String(),
				CurrencyID: &ccy,
			},
		}
		if d.Ext != nil && d.Ext[untdid.ExtKeyAllowance].String() != "" {
			e := d.Ext[untdid.ExtKeyAllowance].String()
			ac.AllowanceChargeReasonCode = &e
		}
		if d.Reason != "" {
			ac.AllowanceChargeReason = &d.Reason
		}
		if d.Percent != nil {
			p := d.Percent.String()
			ac.MultiplierFactorNumeric = &p
		}
		allowanceCharges = append(allowanceCharges, ac)
	}
	return allowanceCharges
}
