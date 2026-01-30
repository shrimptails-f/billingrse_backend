package domain

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

var (
	// ErrMoneyAmountInvalid is returned when the amount is missing or invalid.
	ErrMoneyAmountInvalid = errors.New("money amount is invalid")
	// ErrMoneyAmountScaleInvalid is returned when the amount has too many decimal places.
	ErrMoneyAmountScaleInvalid = errors.New("money amount has too many decimal places")
	// ErrMoneyCurrencyEmpty is returned when the currency is empty.
	ErrMoneyCurrencyEmpty = errors.New("money currency is empty")
	// ErrMoneyCurrencyInvalid is returned when the currency is not a valid ISO 4217 code.
	ErrMoneyCurrencyInvalid = errors.New("money currency is invalid")
)

// Money represents an amount of money with currency.
type Money struct {
	Amount   decimal.Decimal
	Currency string
}

// NewMoney creates a Money value from amount and currency.
func NewMoney(amount float64, currency string) (Money, error) {
	normalizedAmount, err := NormalizeAmount(amount)
	if err != nil {
		return Money{}, err
	}
	normalizedCurrency, err := NormalizeCurrency(currency)
	if err != nil {
		return Money{}, err
	}
	return Money{Amount: normalizedAmount, Currency: normalizedCurrency}, nil
}

// Validate enforces invariants for Money.
func (m Money) Validate() error {
	if _, err := NormalizeAmountDecimal(m.Amount); err != nil {
		return err
	}
	if _, err := NormalizeCurrency(m.Currency); err != nil {
		return err
	}
	return nil
}

// NormalizeAmount validates and normalizes the amount to 3 decimal places.
func NormalizeAmount(amount float64) (decimal.Decimal, error) {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return decimal.Zero, ErrMoneyAmountInvalid
	}
	if amount <= 0 {
		return decimal.Zero, ErrMoneyAmountInvalid
	}
	raw := strconv.FormatFloat(amount, 'f', -1, 64)
	parsed, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, ErrMoneyAmountInvalid
	}
	return NormalizeAmountDecimal(parsed)
}

// NormalizeAmountDecimal validates a decimal amount with up to 3 fractional digits.
func NormalizeAmountDecimal(amount decimal.Decimal) (decimal.Decimal, error) {
	if amount.Cmp(decimal.Zero) <= 0 {
		return decimal.Zero, ErrMoneyAmountInvalid
	}
	if !amount.Equal(amount.Round(3)) {
		return decimal.Zero, ErrMoneyAmountScaleInvalid
	}
	return amount, nil
}

// NormalizeCurrency validates and normalizes currency to uppercase ISO 4217
// (currently allowlisted to JPY and USD).
func NormalizeCurrency(currency string) (string, error) {
	trimmed := strings.TrimSpace(currency)
	if trimmed == "" {
		return "", ErrMoneyCurrencyEmpty
	}
	upper := strings.ToUpper(trimmed)
	if len(upper) != 3 {
		return "", ErrMoneyCurrencyInvalid
	}
	for _, r := range upper {
		if r < 'A' || r > 'Z' {
			return "", ErrMoneyCurrencyInvalid
		}
	}
	if upper != "JPY" && upper != "USD" {
		return "", ErrMoneyCurrencyInvalid
	}
	return upper, nil
}
