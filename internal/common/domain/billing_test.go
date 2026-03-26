package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestBillingValidate(t *testing.T) {
	t.Parallel()

	money, err := NewMoney(1200, "JPY")
	if err != nil {
		t.Fatalf("NewMoney returned error: %v", err)
	}

	base := Billing{
		UserID:        1,
		VendorID:      2,
		EmailID:       9,
		BillingNumber: BillingNumber("INV-001"),
		InvoiceNumber: "",
		Money:         money,
		BillingDate:   timePtr(time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)),
		PaymentCycle:  PaymentCycleRecurring,
		LineItems: []BillingLineItem{
			{
				ProductNameDisplay: stringPtr("Example Product"),
				Amount:             float64Ptr(1200),
				Currency:           stringPtr("JPY"),
			},
		},
	}

	if err := base.Validate(); err != nil {
		t.Fatalf("expected valid billing, got error: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(Billing) Billing
		err    error
	}{
		{
			name: "missing user id",
			mutate: func(b Billing) Billing {
				b.UserID = 0
				return b
			},
			err: ErrBillingUserIDEmpty,
		},
		{
			name: "missing vendor id",
			mutate: func(b Billing) Billing {
				b.VendorID = 0
				return b
			},
			err: ErrBillingVendorIDEmpty,
		},
		{
			name: "missing email id",
			mutate: func(b Billing) Billing {
				b.EmailID = 0
				return b
			},
			err: ErrBillingEmailIDEmpty,
		},
		{
			name: "missing billing number",
			mutate: func(b Billing) Billing {
				b.BillingNumber = ""
				return b
			},
			err: ErrBillingNumberEmpty,
		},
		{
			name: "invalid amount",
			mutate: func(b Billing) Billing {
				b.Money.Amount = decimal.Zero
				return b
			},
			err: ErrMoneyAmountInvalid,
		},
		{
			name: "missing currency",
			mutate: func(b Billing) Billing {
				b.Money.Currency = ""
				return b
			},
			err: ErrMoneyCurrencyEmpty,
		},
		{
			name: "missing payment cycle",
			mutate: func(b Billing) Billing {
				b.PaymentCycle = ""
				return b
			},
			err: ErrPaymentCycleEmpty,
		},
		{
			name: "zero billing date is invalid",
			mutate: func(b Billing) Billing {
				b.BillingDate = timePtr(time.Time{})
				return b
			},
			err: ErrBillingDateInvalid,
		},
		{
			name: "invalid invoice number format",
			mutate: func(b Billing) Billing {
				b.InvoiceNumber = InvoiceNumber("INV-002")
				return b
			},
			err: ErrInvoiceNumberInvalidFormat,
		},
		{
			name: "missing line items",
			mutate: func(b Billing) Billing {
				b.LineItems = nil
				return b
			},
			err: ErrBillingLineItemsEmpty,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mutated := tc.mutate(base)
			if err := mutated.Validate(); !errors.Is(err, tc.err) {
				t.Fatalf("expected %v, got %v", tc.err, err)
			}
		})
	}
}

func TestNewBilling(t *testing.T) {
	t.Parallel()

	invoice := "t1234567890123"
	billing, err := NewBilling(
		1,
		2,
		3,
		"",
		&invoice,
		1200.5,
		"usd",
		timePtr(time.Date(2025, 1, 5, 12, 30, 15, 0, time.UTC)),
		"recurring",
		nil,
		nil,
	)
	assert.ErrorIs(t, err, ErrBillingNumberEmpty)

	billingNumber := " INV-100 "
	invoice = "T1234567890123"
	productNameDisplay := " Example Product "
	billing, err = NewBilling(
		1,
		2,
		3,
		billingNumber,
		&invoice,
		10,
		"JPY",
		timePtr(time.Date(2025, 1, 5, 12, 30, 0, 0, time.UTC)),
		"one_time",
		&productNameDisplay,
		[]BillingLineItemInput{
			{
				ProductNameRaw:     stringPtr(" Example Product Full Name "),
				ProductNameDisplay: stringPtr(" Example Product "),
				Amount:             float64Ptr(10),
				Currency:           stringPtr(" jpy "),
			},
		},
	)
	if err != nil {
		t.Fatalf("NewBilling returned error: %v", err)
	}
	if billing.BillingNumber.String() != "INV-100" {
		t.Fatalf("expected normalized billing number, got %v", billing.BillingNumber)
	}
	if billing.BillingDate == nil || !billing.BillingDate.Equal(time.Date(2025, 1, 5, 12, 30, 0, 0, time.UTC)) {
		t.Fatalf("expected billing date to be preserved, got %+v", billing.BillingDate)
	}
	if billing.ProductNameDisplay == nil || *billing.ProductNameDisplay != "Example Product" {
		t.Fatalf("expected product name display to be normalized, got %+v", billing.ProductNameDisplay)
	}
	if len(billing.LineItems) != 1 {
		t.Fatalf("expected explicit line item to be preserved, got %+v", billing.LineItems)
	}
	if billing.LineItems[0].ProductNameRaw == nil || *billing.LineItems[0].ProductNameRaw != "Example Product Full Name" {
		t.Fatalf("expected explicit line item raw name to be normalized, got %+v", billing.LineItems[0])
	}
	if billing.LineItems[0].Currency == nil || *billing.LineItems[0].Currency != "JPY" {
		t.Fatalf("expected explicit line item currency to be normalized, got %+v", billing.LineItems[0])
	}

	empty := "  "
	billing, err = NewBilling(
		1,
		2,
		3,
		billingNumber,
		&empty,
		10,
		"JPY",
		nil,
		"one_time",
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewBilling returned error: %v", err)
	}
	if !billing.InvoiceNumber.IsEmpty() {
		t.Fatalf("expected invoice number to be empty")
	}
	if billing.BillingDate != nil {
		t.Fatalf("expected billing date to be optional, got %+v", billing.BillingDate)
	}
	if len(billing.LineItems) != 1 {
		t.Fatalf("expected fallback line item to be created, got %+v", billing.LineItems)
	}
	if billing.LineItems[0].Amount == nil || *billing.LineItems[0].Amount != 10 {
		t.Fatalf("expected fallback line item amount, got %+v", billing.LineItems[0])
	}
	if billing.LineItems[0].Currency == nil || *billing.LineItems[0].Currency != "JPY" {
		t.Fatalf("expected fallback line item currency, got %+v", billing.LineItems[0])
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
