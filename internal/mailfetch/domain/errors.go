package domain

import "errors"

var (
	// ErrInvalidCommand is returned when required command fields are missing.
	ErrInvalidCommand = errors.New("manual mail fetch command is invalid")
	// ErrFetchConditionInvalid is returned when the fetch condition is malformed.
	ErrFetchConditionInvalid = errors.New("fetch condition is invalid")
	// ErrConnectionNotFound is returned when the requested connection does not exist for the user.
	ErrConnectionNotFound = errors.New("mail account connection not found")
	// ErrConnectionUnavailable is returned when the requested connection cannot be used for fetching.
	ErrConnectionUnavailable = errors.New("mail account connection is unavailable")
	// ErrProviderUnsupported is returned when a provider has no fetcher implementation.
	ErrProviderUnsupported = errors.New("mail provider is unsupported")
	// ErrProviderLabelNotFound is returned when the requested provider label does not exist.
	ErrProviderLabelNotFound = errors.New("mail provider label not found")
	// ErrProviderSessionBuildFailed is returned when a provider session cannot be initialized.
	ErrProviderSessionBuildFailed = errors.New("mail provider session build failed")
	// ErrProviderListFailed is returned when the provider list API call fails.
	ErrProviderListFailed = errors.New("mail provider list failed")
	// ErrEmailSourceInvalid is returned when provider/account source metadata is missing.
	ErrEmailSourceInvalid = errors.New("email source is invalid")
)
