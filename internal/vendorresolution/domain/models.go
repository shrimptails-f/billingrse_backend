package domain

import commondomain "business/internal/common/domain"

const (
	// MatchedBy* は common/domain に寄せた判定結果コードの再エクスポート。
	MatchedByNameExact      = commondomain.MatchedByNameExact
	MatchedBySenderDomain   = commondomain.MatchedBySenderDomain
	MatchedBySenderName     = commondomain.MatchedBySenderName
	MatchedBySubjectKeyword = commondomain.MatchedBySubjectKeyword

	// FailureStage* はどの段階で処理できなかったかを表す。
	FailureStageNormalizeInput = "normalize_input"
	FailureStageResolveVendor  = "resolve_vendor"
	FailureStageRegisterVendor = "register_vendor"

	// FailureCode* は失敗種別を表す安定コード。
	FailureCodeInvalidResolutionTarget = "invalid_resolution_target"
	FailureCodeVendorResolveFail       = "vendor_resolution_failed"
	FailureCodeVendorRegisterFail      = "vendor_registration_failed"
)

// VendorResolutionInput は common/domain の vendor 判定入力を再エクスポートする。
type VendorResolutionInput = commondomain.VendorResolutionInput

// VendorResolutionFetchPlan は repository が集める検索条件を表す。
type VendorResolutionFetchPlan = commondomain.VendorResolutionFetchPlan

// VendorResolutionFacts は repository が集めた判定材料を表す。
type VendorResolutionFacts = commondomain.VendorResolutionFacts

// ResolutionDecision は common/domain の判定結果を再エクスポートする。
type ResolutionDecision = commondomain.VendorResolutionDecision

// VendorRegistrationPlan は自動登録時の保存計画を表す。
type VendorRegistrationPlan = commondomain.VendorRegistrationPlan

// ResolvedItem は usecase が返す解決済み結果。
type ResolvedItem struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	BodyDigest        string
	VendorID          uint
	VendorName        string
	MatchedBy         string
}

// Failure は usecase が返す 1 件単位の失敗結果。
type Failure struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	Stage             string
	Code              string
}
