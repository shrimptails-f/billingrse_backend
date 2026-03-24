package domain

const (
	// FailureStageNormalizeInput identifies an input normalization failure.
	FailureStageNormalizeInput = "normalize_input"
	// FailureStageAnalyze identifies an analyzer call failure.
	FailureStageAnalyze = "analyze"
	// FailureStageResponseParse identifies a response parsing failure.
	FailureStageResponseParse = "response_parse"
	// FailureStageSave identifies a persistence failure.
	FailureStageSave = "save"
)

const (
	// FailureCodeInvalidEmailInput identifies invalid email input data.
	FailureCodeInvalidEmailInput = "invalid_email_input"
	// FailureCodeAnalysisFailed identifies analyzer execution failures.
	FailureCodeAnalysisFailed = "analysis_failed"
	// FailureCodeAnalysisResponseInvalid identifies malformed analyzer responses.
	FailureCodeAnalysisResponseInvalid = "analysis_response_invalid"
	// FailureCodeAnalysisResponseEmpty identifies empty analyzer responses.
	FailureCodeAnalysisResponseEmpty = "analysis_response_empty"
	// FailureCodeParsedEmailSaveFailed identifies ParsedEmail save failures.
	FailureCodeParsedEmailSaveFailed = "parsed_email_save_failed"
)

// MessageFailure describes a partial failure for a single email.
type MessageFailure struct {
	EmailID           uint   `json:"email_id"`
	ExternalMessageID string `json:"external_message_id"`
	Stage             string `json:"stage"`
	Code              string `json:"code"`
}
