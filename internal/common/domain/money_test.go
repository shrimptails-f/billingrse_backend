package domain

import (
	"errors"
	"math"
	"testing"
)

func TestNewMoney(t *testing.T) {
	t.Parallel()

	money, err := NewMoney(12.345, " usd ")
	if err != nil {
		t.Fatalf("NewMoney returned error: %v", err)
	}
	if money.Amount != 12.345 {
		t.Fatalf("expected amount 12.345, got %v", money.Amount)
	}
	if money.Currency != "USD" {
		t.Fatalf("expected currency USD, got %s", money.Currency)
	}
}

func TestMoneyValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		money Money
		err   error
	}{
		{
			name:  "valid money",
			money: Money{Amount: 1.5, Currency: "JPY"},
			err:   nil,
		},
		{
			name:  "invalid amount",
			money: Money{Amount: 0, Currency: "JPY"},
			err:   ErrMoneyAmountInvalid,
		},
		{
			name:  "invalid amount scale",
			money: Money{Amount: 1.2345, Currency: "JPY"},
			err:   ErrMoneyAmountScaleInvalid,
		},
		{
			name:  "invalid currency empty",
			money: Money{Amount: 1, Currency: ""},
			err:   ErrMoneyCurrencyEmpty,
		},
		{
			name:  "invalid currency format",
			money: Money{Amount: 1, Currency: "JP"},
			err:   ErrMoneyCurrencyInvalid,
		},
		{
			name:  "invalid amount NaN",
			money: Money{Amount: math.NaN(), Currency: "USD"},
			err:   ErrMoneyAmountInvalid,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.money.Validate()
			if tc.err == nil && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tc.err != nil && !errors.Is(err, tc.err) {
				t.Fatalf("expected %v, got %v", tc.err, err)
			}
		})
	}
}
