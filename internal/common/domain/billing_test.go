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
		BillingDate:   time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC),
		PaymentCycle:  PaymentCycleRecurring,
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
			name: "missing billing date",
			mutate: func(b Billing) Billing {
				b.BillingDate = time.Time{}
				return b
			},
			err: ErrBillingDateEmpty,
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
			name: "invalid invoice number format",
			mutate: func(b Billing) Billing {
				b.InvoiceNumber = InvoiceNumber("INV-002")
				return b
			},
			err: ErrInvoiceNumberInvalidFormat,
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
		time.Date(2025, 1, 5, 12, 30, 15, 0, time.UTC),
		"recurring",
	)
	assert.ErrorIs(t, err, ErrBillingNumberEmpty)

	billingNumber := " INV-100 "
	invoice = "T1234567890123"
	billing, err = NewBilling(
		1,
		2,
		3,
		billingNumber,
		&invoice,
		10,
		"JPY",
		time.Date(2025, 1, 5, 12, 30, 0, 0, time.UTC),
		"one_time",
	)
	if err != nil {
		t.Fatalf("NewBilling returned error: %v", err)
	}
	if billing.BillingNumber.String() != "INV-100" {
		t.Fatalf("expected normalized billing number, got %v", billing.BillingNumber)
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
		time.Date(2025, 1, 5, 12, 30, 0, 0, time.UTC),
		"one_time",
	)
	if err != nil {
		t.Fatalf("NewBilling returned error: %v", err)
	}
	if !billing.InvoiceNumber.IsEmpty() {
		t.Fatalf("expected invoice number to be empty")
	}
}
