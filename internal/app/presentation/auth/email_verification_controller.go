package auth

import (
	"business/internal/app/httpresponse"
	"business/internal/auth/application"
	"business/internal/auth/domain"
	"business/internal/library/logger"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type verifyEmailResponse struct {
	Message string       `json:"message"`
	User    userResponse `json:"user"`
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

type resendVerificationRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type resendVerificationResponse struct {
	Message string `json:"message"`
}

// VerifyEmail handles the POST /api/v1/auth/email/verify endpoint.
func (lc *Controller) VerifyEmail(c *gin.Context) {
	reqLog, err := lc.logger.WithContext(c.Request.Context())
	if err != nil {
		httpresponse.WriteInternalServerError(c)
		return
	}

	var req verifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.WriteInvalidRequest(c)
		return
	}

	token := strings.TrimSpace(req.Token)
	if token == "" {
		httpresponse.WriteError(c, http.StatusBadRequest, "missing_token", "トークンが指定されていません。")
		return
	}

	user, err := lc.usecase.VerifyEmail(c.Request.Context(), domain.VerifyEmailRequest{
		Token: token,
	})
	if err != nil {
		if errors.Is(err, application.ErrInvalidToken) {
			httpresponse.WriteError(c, http.StatusBadRequest, "invalid_token", "無効なトークンです。")
			return
		}
		if errors.Is(err, application.ErrTokenExpired) {
			httpresponse.WriteError(c, http.StatusConflict, "token_expired", "トークンの有効期限が切れています。再送信をお試しください。")
			return
		}
		if errors.Is(err, application.ErrTokenAlreadyUsed) {
			httpresponse.WriteError(c, http.StatusConflict, "token_already_used", "このトークンは既に使用済みです。")
			return
		}
		reqLog.Error("VerifyEmail error", logger.Err(err))
		httpresponse.WriteInternalServerError(c)
		return
	}

	c.JSON(http.StatusOK, verifyEmailResponse{
		Message: "メールアドレスの認証が完了しました。",
		User: userResponse{
			ID:              user.ID,
			Name:            user.Name.String(),
			Email:           user.Email.String(),
			EmailVerified:   user.IsEmailVerified(),
			EmailVerifiedAt: user.EmailVerifiedAt,
			CreatedAt:       user.CreatedAt,
		},
	})
}

// ResendVerificationEmail handles the POST /api/v1/auth/email/resend endpoint.
func (lc *Controller) ResendVerificationEmail(c *gin.Context) {
	var req resendVerificationRequest
	reqLog, err := lc.logger.WithContext(c.Request.Context())
	if err != nil {
		httpresponse.WriteInternalServerError(c)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.WriteInvalidRequest(c)
		return
	}

	err = lc.usecase.ResendVerificationEmail(c.Request.Context(), domain.ResendVerificationRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, application.ErrInvalidCredentials) {
			httpresponse.WriteError(c, http.StatusUnauthorized, "invalid_credentials", "メールアドレスまたはパスワードが正しくありません。")
			return
		}
		if errors.Is(err, application.ErrAlreadyVerified) {
			httpresponse.WriteError(c, http.StatusForbidden, "already_verified", "このメールアドレスは既に認証済みです。")
			return
		}
		if errors.Is(err, application.ErrRateLimitExceeded) {
			httpresponse.WriteError(c, http.StatusTooManyRequests, "rate_limit_exceeded", "再送信の回数制限に達しました。15分後に再度お試しください。")
			return
		}
		if errors.Is(err, application.ErrMailSendFailed) {
			httpresponse.WriteServiceUnavailable(c, "mail_send_failed", "メール送信に失敗しました。しばらくしてから再度お試しください。")
			return
		}
		reqLog.Error("ResendVerificationEmail error", logger.Err(err))
		httpresponse.WriteInternalServerError(c)
		return
	}

	c.JSON(http.StatusOK, resendVerificationResponse{
		Message: "確認メールを再送信しました。メールボックスをご確認ください。",
	})
}
