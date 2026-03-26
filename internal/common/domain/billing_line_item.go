package domain

// BillingLineItemInput is raw input used to construct a BillingLineItem.
type BillingLineItemInput struct {
	ProductNameRaw     *string
	ProductNameDisplay *string
	Amount             *float64
	Currency           *string
}

// Normalize trims optional strings and canonicalizes currency for one input line-item.
func (i BillingLineItemInput) Normalize() BillingLineItemInput {
	i.ProductNameRaw = normalizeOptionalString(i.ProductNameRaw)
	i.ProductNameDisplay = normalizeOptionalString(i.ProductNameDisplay)
	i.Amount = cloneOptionalFloat64(i.Amount)
	i.Currency = normalizeOptionalUpperString(i.Currency)
	return i
}

// IsEmpty reports whether a line-item input has no extracted fields.
func (i BillingLineItemInput) IsEmpty() bool {
	return i.ProductNameRaw == nil &&
		i.ProductNameDisplay == nil &&
		i.Amount == nil &&
		i.Currency == nil
}

// ToBillingLineItem converts normalized raw input into an aggregate child entity.
func (i BillingLineItemInput) ToBillingLineItem() BillingLineItem {
	return BillingLineItem{
		ProductNameRaw:     i.ProductNameRaw,
		ProductNameDisplay: i.ProductNameDisplay,
		Amount:             i.Amount,
		Currency:           i.Currency,
	}
}

// BillingLineItem represents one detail row nested under a billing.
type BillingLineItem struct {
	ProductNameRaw     *string
	ProductNameDisplay *string
	Amount             *float64
	Currency           *string
}

// Normalize trims optional strings and canonicalizes currency for one line-item.
func (i BillingLineItem) Normalize() BillingLineItem {
	i.ProductNameRaw = normalizeOptionalString(i.ProductNameRaw)
	i.ProductNameDisplay = normalizeOptionalString(i.ProductNameDisplay)
	i.Amount = cloneOptionalFloat64(i.Amount)
	i.Currency = normalizeOptionalUpperString(i.Currency)
	return i
}

// IsEmpty reports whether a line-item has no extracted fields.
func (i BillingLineItem) IsEmpty() bool {
	return i.ProductNameRaw == nil &&
		i.ProductNameDisplay == nil &&
		i.Amount == nil &&
		i.Currency == nil
}

func normalizeBillingLineItemInputs(items []BillingLineItemInput) []BillingLineItemInput {
	if len(items) == 0 {
		return nil
	}

	normalized := make([]BillingLineItemInput, 0, len(items))
	for _, item := range items {
		item = item.Normalize()
		if item.IsEmpty() {
			continue
		}
		normalized = append(normalized, item)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeBillingLineItems(items []BillingLineItem) []BillingLineItem {
	if len(items) == 0 {
		return nil
	}

	normalized := make([]BillingLineItem, 0, len(items))
	for _, item := range items {
		item = item.Normalize()
		if item.IsEmpty() {
			continue
		}
		normalized = append(normalized, item)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func cloneOptionalFloat64(value *float64) *float64 {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}
