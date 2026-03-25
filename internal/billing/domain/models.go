package domain

const (
	// FailureStageNormalizeInput indicates target normalization or validation failed.
	FailureStageNormalizeInput = "normalize_input"
	// FailureStageBuildBilling indicates billing aggregate construction failed.
	FailureStageBuildBilling = "build_billing"
	// FailureStageSaveBilling indicates persistence failed unexpectedly.
	FailureStageSaveBilling = "save_billing"

	// ReasonCodeDuplicateBilling indicates the target was already persisted.
	ReasonCodeDuplicateBilling = "duplicate_billing"

	// FailureCodeInvalidCreationTarget indicates the workflow passed an invalid target.
	FailureCodeInvalidCreationTarget = "invalid_creation_target"
	// FailureCodeBillingConstructFailed indicates aggregate construction failed.
	FailureCodeBillingConstructFailed = "billing_construct_failed"
	// FailureCodeBillingPersistFailed indicates persistence failed.
	FailureCodeBillingPersistFailed = "billing_persist_failed"
)

// CreatedItem is a successfully created billing result.
type CreatedItem struct {
	BillingID         uint
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	VendorID          uint
	VendorName        string
	BillingNumber     string
}

// DuplicateItem is a duplicate billing result mapped to an existing billing row.
type DuplicateItem struct {
	ExistingBillingID uint
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	VendorID          uint
	VendorName        string
	BillingNumber     string
	ReasonCode        string
	Message           string
}

// Failure is a technical or contract failure for a single target.
type Failure struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	Stage             string
	Code              string
	Message           string
}
