package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestDoWithCondition_Success tests that the operation succeeds on the first try.
func TestDoWithCondition_Success(t *testing.T) {
	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		return nil
	}

	err := DoWithCondition(context.Background(), []time.Duration{0, 0, 0}, nil, operation)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call, got: %d", callCount)
	}
}

// TestDoWithCondition_RetryUntilSuccess tests that the operation retries and eventually succeeds.
func TestDoWithCondition_RetryUntilSuccess(t *testing.T) {
	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	shouldRetry := func(err error) bool {
		return true // Always retry
	}

	err := DoWithCondition(context.Background(), []time.Duration{0, 0, 0}, shouldRetry, operation)
	if err != nil {
		t.Fatalf("expected no error after retries, got: %v", err)
	}
	if callCount != 3 {
		t.Fatalf("expected 3 calls, got: %d", callCount)
	}
}

// TestDoWithCondition_NoRetryOnConditionFalse tests that retry stops immediately when shouldRetry returns false.
func TestDoWithCondition_NoRetryOnConditionFalse(t *testing.T) {
	callCount := 0
	expectedErr := errors.New("non-retryable error")
	operation := func(ctx context.Context) error {
		callCount++
		return expectedErr
	}

	shouldRetry := func(err error) bool {
		return false // Never retry
	}

	err := DoWithCondition(context.Background(), []time.Duration{0, 0, 0}, shouldRetry, operation)
	if err != expectedErr {
		t.Fatalf("expected error %v, got: %v", expectedErr, err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call (no retries), got: %d", callCount)
	}
}

// TestDoWithCondition_ContextCancellation tests that the operation stops when context is canceled.
func TestDoWithCondition_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		if callCount == 1 {
			cancel() // Cancel context after first attempt
		}
		return errors.New("always fails")
	}

	shouldRetry := func(err error) bool {
		return true // Always retry if not canceled
	}

	err := DoWithCondition(ctx, []time.Duration{1 * time.Second, 1 * time.Second}, shouldRetry, operation)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
	// Should be 1 call + context cancellation during wait
	if callCount != 1 {
		t.Fatalf("expected 1 call before cancellation, got: %d", callCount)
	}
}

// TestDoWithCondition_ContextDeadline tests that the operation stops when context deadline is exceeded.
func TestDoWithCondition_ContextDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		return errors.New("always fails")
	}

	shouldRetry := func(err error) bool {
		return true
	}

	// Use longer backoff so deadline is hit during wait
	err := DoWithCondition(ctx, []time.Duration{100 * time.Millisecond, 100 * time.Millisecond}, shouldRetry, operation)
	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}
	if callCount < 1 {
		t.Fatalf("expected at least 1 call, got: %d", callCount)
	}
}

// TestDoWithCondition_NilShouldRetry tests that nil shouldRetry retries all errors (backward compatibility).
func TestDoWithCondition_NilShouldRetry(t *testing.T) {
	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		if callCount < 4 {
			return errors.New("temporary error")
		}
		return nil
	}

	err := DoWithCondition(context.Background(), []time.Duration{0, 0, 0}, nil, operation)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if callCount != 4 {
		t.Fatalf("expected 4 calls (initial + 3 retries), got: %d", callCount)
	}
}

// TestDoWithCondition_MaxRetriesExhausted tests that the last error is returned after exhausting retries.
func TestDoWithCondition_MaxRetriesExhausted(t *testing.T) {
	expectedErr := errors.New("persistent error")
	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		return expectedErr
	}

	shouldRetry := func(err error) bool {
		return true
	}

	err := DoWithCondition(context.Background(), []time.Duration{0, 0, 0}, shouldRetry, operation)
	if err != expectedErr {
		t.Fatalf("expected error %v, got: %v", expectedErr, err)
	}
	if callCount != 4 {
		t.Fatalf("expected 4 calls (initial + 3 retries), got: %d", callCount)
	}
}

// TestDo_BackwardCompatibility tests that the old Do function still works as expected.
func TestDo_BackwardCompatibility(t *testing.T) {
	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		if callCount < 2 {
			return errors.New("temporary error")
		}
		return nil
	}

	err := Do(context.Background(), []time.Duration{0, 0}, operation)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 calls, got: %d", callCount)
	}
}

// TestDoWithCondition_ConditionalRetry tests a realistic scenario with conditional retry.
func TestDoWithCondition_ConditionalRetry(t *testing.T) {
	retryableErr := errors.New("429 rate limit")
	nonRetryableErr := errors.New("400 bad request")

	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		if callCount == 1 {
			return retryableErr // Should retry
		} else if callCount == 2 {
			return nonRetryableErr // Should NOT retry
		}
		return nil
	}

	shouldRetry := func(err error) bool {
		return errors.Is(err, retryableErr)
	}

	err := DoWithCondition(context.Background(), []time.Duration{0, 0, 0}, shouldRetry, operation)
	if err != nonRetryableErr {
		t.Fatalf("expected non-retryable error %v, got: %v", nonRetryableErr, err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 calls (initial + 1 retry, then stop), got: %d", callCount)
	}
}

// TestDoWithCondition_NilBackoff tests that nil backoff executes exactly once (no retries).
func TestDoWithCondition_NilBackoff(t *testing.T) {
	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		return errors.New("always fails")
	}

	err := DoWithCondition(context.Background(), nil, nil, operation)
	if err == nil {
		t.Fatal("expected error from operation")
	}
	if callCount != 1 {
		t.Fatalf("expected exactly 1 call with nil backoff, got: %d", callCount)
	}
}

// TestDoWithCondition_EmptyBackoff tests that empty backoff executes exactly once (no retries).
func TestDoWithCondition_EmptyBackoff(t *testing.T) {
	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		return errors.New("always fails")
	}

	err := DoWithCondition(context.Background(), []time.Duration{}, nil, operation)
	if err == nil {
		t.Fatal("expected error from operation")
	}
	if callCount != 1 {
		t.Fatalf("expected exactly 1 call with empty backoff, got: %d", callCount)
	}
}

// TestDo_NilBackoff tests backward compatibility with nil backoff.
func TestDo_NilBackoff(t *testing.T) {
	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		if callCount == 1 {
			return errors.New("temporary error")
		}
		return nil
	}

	err := Do(context.Background(), nil, operation)
	if err == nil {
		t.Fatal("expected error since no retries with nil backoff")
	}
	if callCount != 1 {
		t.Fatalf("expected exactly 1 call with nil backoff, got: %d", callCount)
	}
}
