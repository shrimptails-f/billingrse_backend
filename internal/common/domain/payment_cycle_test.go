package domain

import (
	"errors"
	"testing"
)

func TestNewPaymentCycle(t *testing.T) {
	t.Parallel()

	cycle, err := NewPaymentCycle("recurring")
	if err != nil {
		t.Fatalf("NewPaymentCycle returned error: %v", err)
	}
	if cycle != PaymentCycleRecurring {
		t.Fatalf("expected recurring, got %s", cycle)
	}

	cycle, err = NewPaymentCycle(" One_Time ")
	if err != nil {
		t.Fatalf("NewPaymentCycle returned error: %v", err)
	}
	if cycle != PaymentCycleOneTime {
		t.Fatalf("expected one_time, got %s", cycle)
	}

	if _, err := NewPaymentCycle(""); !errors.Is(err, ErrPaymentCycleEmpty) {
		t.Fatalf("expected ErrPaymentCycleEmpty, got %v", err)
	}

	if _, err := NewPaymentCycle("monthly"); !errors.Is(err, ErrPaymentCycleInvalid) {
		t.Fatalf("expected ErrPaymentCycleInvalid, got %v", err)
	}
}
