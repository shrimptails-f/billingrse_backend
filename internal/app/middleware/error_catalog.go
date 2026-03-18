package middleware

const (
	errorCodeMissingToken              = "missing_token"
	errorCodeInvalidToken              = "invalid_token"
	errorCodeInvalidTokenClaims        = "invalid_token_claims"
	errorCodeUserNotFound              = "user_not_found"
	errorCodeEmailVerificationRequired = "email_verification_required"
	errorCodeCSRFOriginNotAllowed      = "csrf_origin_not_allowed"
	errorCodeInternalServerError       = "internal_server_error"
)

const (
	errorMessageMissingToken              = "認証トークンがありません。"
	errorMessageInvalidToken              = "認証トークンが無効です。"
	errorMessageInvalidTokenClaims        = "認証トークンの内容が不正です。"
	errorMessageUserNotFound              = "ユーザーが見つかりません。"
	errorMessageEmailVerificationRequired = "メールアドレスの認証が完了していません。確認メールのリンクから認証を完了してください。"
	errorMessageCSRFOriginNotAllowed      = "オリジンまたはリファラが許可されていません。"
	errorMessageInternalServerError       = "サーバー内部でエラーが発生しました。しばらくしてから再度お試しください。"
)
