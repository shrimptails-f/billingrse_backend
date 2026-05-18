package domain

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

func TestNewMoney(t *testing.T) {
	t.Parallel()

	money, err := NewMoneyFromString("12.345", " usd ")
	if err != nil {
		t.Fatalf("NewMoney returned error: %v", err)
	}
	if !money.Amount.Equal(decimal.RequireFromString("12.345")) {
		t.Fatalf("expected amount 12.345, got %s", money.Amount.String())
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
			money: Money{Amount: decimal.RequireFromString("1.5"), Currency: "JPY"},
			err:   nil,
		},
		{
			name:  "invalid amount",
			money: Money{Amount: decimal.Zero, Currency: "JPY"},
			err:   ErrMoneyAmountInvalid,
		},
		{
			name:  "invalid amount scale",
			money: Money{Amount: decimal.RequireFromString("1.2345"), Currency: "JPY"},
			err:   ErrMoneyAmountScaleInvalid,
		},
		{
			name:  "invalid currency empty",
			money: Money{Amount: decimal.RequireFromString("1"), Currency: ""},
			err:   ErrMoneyCurrencyEmpty,
		},
		{
			name:  "invalid currency unsupported",
			money: Money{Amount: decimal.RequireFromString("1"), Currency: "EUR"},
			err:   ErrMoneyCurrencyInvalid,
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

func TestNormalizeAmountString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		input  string
		want   string
		err    error
	}{
		{name: "integer", input: "1200", want: "1200"},
		{name: "trimmed decimal", input: " 1200.500 ", want: "1200.5"},
		{name: "max scale", input: "999999999999999.999", want: "999999999999999.999"},
		{name: "empty", input: "", err: ErrMoneyAmountInvalid},
		{name: "alpha", input: "abc", err: ErrMoneyAmountInvalid},
		{name: "comma separated", input: "1,200", err: ErrMoneyAmountInvalid},
		{name: "currency prefixed", input: "¥1200", err: ErrMoneyAmountInvalid},
		{name: "zero", input: "0", err: ErrMoneyAmountInvalid},
		{name: "negative", input: "-1", err: ErrMoneyAmountInvalid},
		{name: "too many decimals", input: "12.3456", err: ErrMoneyAmountScaleInvalid},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeAmountString(tc.input)
			if tc.err != nil {
				if !errors.Is(err, tc.err) {
					t.Fatalf("expected %v, got %v", tc.err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if !got.Equal(decimal.RequireFromString(tc.want)) {
				t.Fatalf("expected %s, got %s", tc.want, got.String())
			}
		})
	}
}
