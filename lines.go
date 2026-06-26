package ubl

import (
	"strconv"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/iso"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

// InvoiceLine represents a line item in an invoice and credit note
type InvoiceLine struct {
	ID                  string              `xml:"cbc:ID"`
	Note                []string            `xml:"cbc:Note"`
	InvoicedQuantity    *Quantity           `xml:"cbc:InvoicedQuantity,omitempty"` // or CreditNoteQuantity
	CreditedQuantity    *Quantity           `xml:"cbc:CreditedQuantity,omitempty"`
	LineExtensionAmount Amount              `xml:"cbc:LineExtensionAmount"`
	AccountingCost      *string             `xml:"cbc:AccountingCost"`
	InvoicePeriod       *Period             `xml:"cac:InvoicePeriod"`
	OrderLineReference  *OrderLineReference `xml:"cac:OrderLineReference"`
	DocumentReference   *LineDocReference   `xml:"cac:DocumentReference,omitempty"`
	AllowanceCharge     []*AllowanceCharge  `xml:"cac:AllowanceCharge"`
	TaxTotal            []TaxTotal          `xml:"cac:TaxTotal,omitempty"`
	Item                *Item               `xml:"cac:Item"`
	Price               *Price              `xml:"cac:Price"`
}

// LineDocReference defines a document reference at line level (BT-128)
type LineDocReference struct {
	ID               IDType  `xml:"cbc:ID"`
	DocumentTypeCode *string `xml:"cbc:DocumentTypeCode,omitempty"`
}

func (ui *Invoice) addLines(inv *bill.Invoice, context Context) { //nolint:gocyclo
	if len(inv.Lines) == 0 {
		return
	}

	var lines []InvoiceLine
	invoiceType := ui.getInvoiceTypeBasedOnXMLName()

	for _, l := range inv.Lines {
		ccy := l.Item.Currency.String()
		if ccy == "" {
			ccy = inv.Currency.String()
		}
		invLine := InvoiceLine{
			ID: strconv.Itoa(l.Index),

			LineExtensionAmount: Amount{
				CurrencyID: &ccy,
				Value:      lineExtensionValue(l, context, ccy),
			},
		}

		// Always set quantity (mandatory field)
		iq := &Quantity{
			Value: l.Quantity.String(),
		}
		if l.Item != nil && l.Item.Unit != "" {
			iq.UnitCode = string(l.Item.Unit.UNECE())
		}
		if invoiceType.In(bill.InvoiceTypeCreditNote) {
			invLine.CreditedQuantity = iq
		} else {
			invLine.InvoicedQuantity = iq
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

		// BT-128: Invoice line object identifier
		if l.Identifier != nil {
			typeCode := "130"
			ref := &LineDocReference{
				ID:               IDType{Value: l.Identifier.Code.String()},
				DocumentTypeCode: &typeCode,
			}
			if l.Identifier.Ext.Has(untdid.ExtKeyReference) {
				s := l.Identifier.Ext.Get(untdid.ExtKeyReference).String()
				ref.ID.SchemeID = &s
			}
			invLine.DocumentReference = ref
		}

		if l.Period != nil {
			invLine.InvoicePeriod = &Period{
				StartDate: formatDate(l.Period.Start),
				EndDate:   formatDate(l.Period.End),
			}
		}

		if l.Order != "" {
			invLine.OrderLineReference = &OrderLineReference{
				LineID: l.Order.String(),
			}
		}

		// OIOUBL reconciles allowances/charges at the DOCUMENT level
		// (F-INV126/128/129 sum document-level AllowanceCharge, not line-level),
		// so for OIOUBL they're promoted in addTotals instead of set on the line.
		if (len(l.Charges) > 0 || len(l.Discounts) > 0) && !context.Is(ContextOIOUBL21) {
			invLine.AllowanceCharge = makeLineCharges(l.Charges, l.Discounts, ccy, l.Sum, context, l.Taxes)
		}

		// Line VAT amount (KSA-11) is mandatory for tax
		// invoice and associated credit notes and debit notes
		if context.Is(ContextZATCA) && l.Total != nil && len(l.Taxes) > 0 && l.Taxes[0].Percent != nil {
			taxAmount := l.Taxes[0].Percent.Of(*l.Total)
			roundingAmount := l.Total.Add(taxAmount)
			invLine.TaxTotal = []TaxTotal{
				{
					TaxAmount:      Amount{Value: taxAmount.String(), CurrencyID: &ccy},
					RoundingAmount: &Amount{Value: roundingAmount.String(), CurrencyID: &ccy},
				},
			}
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

			// OIOUBL forbids cac:OriginCountry on a line item (F-INV211 / F-CRN109).
			if l.Item.Origin != "" && !context.Is(ContextOIOUBL21) {
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
						ID: IDType{Value: l.Taxes[0].Category.String()},
					},
				}

				if s := oioubl21TaxCategoryID(l.Taxes[0].Ext); s != "" {
					it.ClassifiedTaxCategory.ID = &IDType{Value: s}
				}

				// Set percent: required unless category is "O" (outside scope)
				if l.Taxes[0].Percent != nil {
					p := l.Taxes[0].Percent.StringWithoutSymbol()
					it.ClassifiedTaxCategory.Percent = &p
				} else if it.ClassifiedTaxCategory.ID == nil || it.ClassifiedTaxCategory.ID.Value != "O" {
					// Default to 0% when not outside scope
					p := "0"
					it.ClassifiedTaxCategory.Percent = &p
				}
			}

			if len(l.Item.Identities) > 0 {
				for _, id := range l.Item.Identities {
					// BT-158/159: Item classification (Label holds the listID)
					if id.Label != "" && !id.Ext.Has(iso.ExtKeySchemeID) {
						listID := id.Label
						if it.CommodityClassification == nil {
							it.CommodityClassification = &[]CommodityClassification{}
						}
						*it.CommodityClassification = append(*it.CommodityClassification, CommodityClassification{
							ItemClassificationCode: &IDType{
								Value:  id.Code.String(),
								ListID: &listID,
							},
						})
						continue
					}

					if it.BuyersItemIdentification != nil && it.StandardItemIdentification != nil {
						break
					}

					s := id.Ext.Get(iso.ExtKeySchemeID).String()

					// Map first identity without extension to BuyersItemIdentification
					if s == "" {
						if it.BuyersItemIdentification == nil {
							it.BuyersItemIdentification = &ItemIdentification{
								ID: &IDType{
									Value: id.Code.String(),
								},
							}
						}
						continue
					}

					// Map first identity with extension to StandardItemIdentification
					if it.StandardItemIdentification == nil {
						it.StandardItemIdentification = &ItemIdentification{
							ID: &IDType{
								SchemeID: &s,
								Value:    id.Code.String(),
							},
						}
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

			if l.Item.Ref != "" {
				invLine.Item.SellersItemIdentification = &ItemIdentification{
					ID: &IDType{
						Value: l.Item.Ref.String(),
					},
				}
			}
		}

		if context.Is(ContextOIOUBL21) {
			invLine.TaxTotal = makeLineTaxTotals(l, ccy, context)
		}

		lines = append(lines, invLine)
	}
	if invoiceType.In(bill.InvoiceTypeCreditNote) {
		ui.CreditNoteLines = lines
	} else {
		ui.InvoiceLines = lines
	}

	if context.Is(ContextOIOUBL21) {
		applyOIOUBL21LineTaxCategories(ui.InvoiceLines)
		applyOIOUBL21LineTaxCategories(ui.CreditNoteLines)
	}
}

// applyOIOUBL21LineTaxCategories maps the tax categories on a set of lines: the
// item classified category, the line-level subtotals, and any promoted
// allowance/charges. Invoice and credit-note lines share the InvoiceLine type.
func applyOIOUBL21LineTaxCategories(lines []InvoiceLine) {
	for i := range lines {
		line := &lines[i]
		if line.Item != nil && line.Item.ClassifiedTaxCategory != nil {
			applyOIOUBL21ClassifiedTaxCategory(line.Item.ClassifiedTaxCategory)
		}
		for j := range line.TaxTotal {
			for k := range line.TaxTotal[j].TaxSubtotal {
				applyOIOUBL21TaxCategory(&line.TaxTotal[j].TaxSubtotal[k].TaxCategory)
			}
		}
		for _, ac := range line.AllowanceCharge {
			for _, tc := range ac.TaxCategory {
				applyOIOUBL21TaxCategory(tc)
			}
		}
	}
}

// rescaleToCurrency rounds the amount to the natural precision of the given
// currency code (e.g. 2 for EUR, 0 for JPY). Falls back to the amount's
// existing precision if the currency code is unknown.
func roundToCurrency(a num.Amount, ccy string) num.Amount {
	if def := currency.Code(ccy).Def(); def != nil {
		return def.Rescale(a)
	}
	return a
}

func rescaleToCurrency(a num.Amount, ccy string) string {
	return roundToCurrency(a, ccy).String()
}

// lineExtensionValue renders the line LineExtensionAmount. OIOUBL F-INV348
// requires the gross Price×Qty (line allowances are carried separately and
// netted at the document level); other profiles use the net line total.
func lineExtensionValue(l *bill.Line, ctx Context, ccy string) string {
	if ctx.Is(ContextOIOUBL21) && l.Sum != nil {
		// gross, rounded to currency precision (l.Sum is the raw qty×price)
		return rescaleToCurrency(*l.Sum, ccy)
	}
	return l.Total.String()
}

func makeLineTaxTotals(line *bill.Line, ccy string, ctx Context) []TaxTotal {
	if line == nil || len(line.Taxes) == 0 {
		return nil
	}

	var taxable num.Amount
	switch {
	case ctx.Is(ContextOIOUBL21) && line.Sum != nil:
		// OIOUBL line TaxableAmount is gross (Price×Qty), rounded to currency
		// precision (l.Sum is the raw product); the discount is subtracted once
		// at the document level (F-LIB402 sums gross line taxable amounts then
		// adjusts for the document AllowanceCharge).
		taxable = roundToCurrency(*line.Sum, ccy)
	case line.Total != nil:
		taxable = *line.Total
	case line.Sum != nil:
		taxable = *line.Sum
	default:
		return nil
	}

	taxTotal := TaxTotal{
		TaxAmount: Amount{Value: "0", CurrencyID: &ccy},
	}
	totalAmount := num.MakeAmount(0, taxable.Exp())

	for _, t := range line.Taxes {
		subtotal := TaxSubtotal{
			TaxableAmount: Amount{Value: taxable.String(), CurrencyID: &ccy},
		}
		taxCat := TaxCategory{}

		if k := oioubl21TaxCategoryID(t.Ext); k != "" {
			taxCat.ID = &IDType{Value: k}
		}

		if t.Percent != nil {
			p := t.Percent.StringWithoutSymbol()
			taxCat.Percent = &p
			amount := t.Percent.Of(taxable).Rescale(taxable.Exp())
			subtotal.TaxAmount = Amount{Value: amount.String(), CurrencyID: &ccy}
			totalAmount = totalAmount.Add(amount)
		} else {
			// No percent (e.g. exempt): still emit at currency precision
			// ("0.00"), or OIOUBL F-LIB263 rejects a bare "0".
			subtotal.TaxAmount = Amount{Value: num.MakeAmount(0, taxable.Exp()).String(), CurrencyID: &ccy}
		}

		if t.Category != "" {
			taxCat.TaxScheme = &TaxScheme{ID: IDType{Value: t.Category.String()}}
		}
		subtotal.TaxCategory = taxCat
		taxTotal.TaxSubtotal = append(taxTotal.TaxSubtotal, subtotal)
	}

	// OIOUBL requires a line TaxTotal even for 0% lines (F-INV138 / F-LIB404);
	// other profiles omit it when the line tax amount is zero.
	if totalAmount.IsZero() && !ctx.Is(ContextOIOUBL21) {
		return nil
	}
	taxTotal.TaxAmount = Amount{Value: totalAmount.String(), CurrencyID: &ccy}

	return []TaxTotal{taxTotal}
}

func makeLineCharges(charges []*bill.LineCharge, discounts []*bill.LineDiscount, ccy string, baseSum *num.Amount, ctx Context, taxes tax.Set) []*AllowanceCharge {
	var allowanceCharges []*AllowanceCharge
	// BR-DEC-24 / UBL-DT-01: line allowance and charge amounts (BT-136/BT-141)
	// and their base amounts must match the currency's natural precision.
	// GOBL keeps higher precision internally — notably after RemoveIncludedTaxes
	// strips VAT from prices_include invoices — so round here at the boundary.
	var base *Amount
	if baseSum != nil {
		base = &Amount{
			Value:      rescaleToCurrency(*baseSum, ccy),
			CurrencyID: &ccy,
		}
	}
	for _, ch := range charges {
		ac := &AllowanceCharge{
			ChargeIndicator: true,
			Amount: Amount{
				Value:      rescaleToCurrency(ch.Amount, ccy),
				CurrencyID: &ccy,
			},
		}
		if e := ch.Ext.Get(untdid.ExtKeyCharge).String(); e != "" {
			ac.AllowanceChargeReasonCode = &e
		}
		if ch.Reason != "" {
			ac.AllowanceChargeReason = &ch.Reason
		}
		if ch.Percent != nil {
			p := allowanceMultiplier(ch.Percent, ctx)
			ac.MultiplierFactorNumeric = &p
			if base != nil {
				ac.BaseAmount = base
			}
		}
		if ctx.Is(ContextOIOUBL21) {
			ac.TaxCategory = makeTaxCategory(taxes, ctx) // F-LIB226: line allowance needs a TaxCategory
		}
		allowanceCharges = append(allowanceCharges, ac)
	}
	for _, d := range discounts {
		ac := &AllowanceCharge{
			ChargeIndicator: false,
			Amount: Amount{
				Value:      rescaleToCurrency(d.Amount, ccy),
				CurrencyID: &ccy,
			},
		}
		if e := d.Ext.Get(untdid.ExtKeyAllowance).String(); e != "" {
			ac.AllowanceChargeReasonCode = &e
		}
		if d.Reason != "" {
			ac.AllowanceChargeReason = &d.Reason
		}
		if d.Percent != nil {
			p := allowanceMultiplier(d.Percent, ctx)
			ac.MultiplierFactorNumeric = &p
			if base != nil {
				ac.BaseAmount = base
			}
		}
		if ctx.Is(ContextOIOUBL21) {
			ac.TaxCategory = makeTaxCategory(taxes, ctx) // F-LIB226: line allowance needs a TaxCategory
		}
		allowanceCharges = append(allowanceCharges, ac)
	}
	return allowanceCharges
}
