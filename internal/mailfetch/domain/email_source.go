package domain

import (
	"fmt"
	"strings"
)

// EmailSource identifies which mailbox a fetched email came from.
type EmailSource struct {
	Provider          string
	AccountIdentifier string
}

// Validate ensures source metadata required for idempotent persistence exists.
func (s EmailSource) Validate() error {
	if strings.TrimSpace(s.Provider) == "" {
		return fmt.Errorf("%w: provider is required", ErrEmailSourceInvalid)
	}
	if strings.TrimSpace(s.AccountIdentifier) == "" {
		return fmt.Errorf("%w: account_identifier is required", ErrEmailSourceInvalid)
	}
	return nil
}
