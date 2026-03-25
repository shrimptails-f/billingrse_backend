package domain

const (
	// FailureStageFetchDetail identifies a failure while loading a provider message detail.
	FailureStageFetchDetail = "fetch_detail"
	// FailureStageNormalize identifies a failure while validating/normalizing message metadata.
	FailureStageNormalize = "normalize"
	// FailureStageSave identifies a failure while persisting a fetched email.
	FailureStageSave = "save"
)

const (
	// FailureCodeFetchDetailFailed identifies a provider detail-read failure.
	FailureCodeFetchDetailFailed = "fetch_detail_failed"
	// FailureCodeInvalidFetchedEmail identifies invalid normalized email metadata.
	FailureCodeInvalidFetchedEmail = "invalid_fetched_email"
	// FailureCodeDuplicateExternalMessageID identifies a duplicate external message ID in one fetch batch.
	FailureCodeDuplicateExternalMessageID = "duplicate_external_message_id"
	// FailureCodeEmailSaveFailed identifies a persistence failure for one email.
	FailureCodeEmailSaveFailed = "email_save_failed"
)

// MessageFailure describes a per-message partial failure.
type MessageFailure struct {
	ExternalMessageID string `json:"external_message_id"`
	Stage             string `json:"stage"`
	Code              string `json:"code"`
	Message           string `json:"message"`
}
