package domain

import "strings"

// ConnectionRef is the minimal provider-agnostic connection information required for fetching.
type ConnectionRef struct {
	ConnectionID      uint
	UserID            uint
	Provider          string
	AccountIdentifier string
}

// Source returns the email source metadata used for persistence.
func (r ConnectionRef) Source() EmailSource {
	return EmailSource{
		Provider:          strings.TrimSpace(r.Provider),
		AccountIdentifier: strings.TrimSpace(r.AccountIdentifier),
	}
}
