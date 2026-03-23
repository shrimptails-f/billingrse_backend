package domain

import (
	"fmt"
	"strings"
	"time"
)

// FetchCondition represents the required mail fetch condition.
type FetchCondition struct {
	LabelName string
	Since     time.Time
	Until     time.Time
}

// Normalize trims free-form fields while preserving the provided timestamps.
func (c FetchCondition) Normalize() FetchCondition {
	c.LabelName = strings.TrimSpace(c.LabelName)
	return c
}

// Validate enforces the v1 fetch-condition invariants.
func (c FetchCondition) Validate() error {
	normalized := c.Normalize()
	if normalized.LabelName == "" {
		return fmt.Errorf("%w: label_name is required", ErrFetchConditionInvalid)
	}
	if normalized.Since.IsZero() {
		return fmt.Errorf("%w: since is required", ErrFetchConditionInvalid)
	}
	if normalized.Until.IsZero() {
		return fmt.Errorf("%w: until is required", ErrFetchConditionInvalid)
	}
	if !normalized.Since.Before(normalized.Until) {
		return fmt.Errorf("%w: since must be before until", ErrFetchConditionInvalid)
	}
	return nil
}
