package presentation

import (
	"business/internal/auth/application"
	"business/internal/auth/domain"
	"business/internal/library/logger"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type resendVerificationRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type resendVerificationResponse struct {
	Message string `json:"message"`
}

// ResendVerificationEmail handles the POST /auth/email/resend endpoint
func (lc *AuthController) ResendVerificationEmail(c *gin.Context) {
	var req resendVerificationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	err := lc.usecase.ResendVerificationEmail(c.Request.Context(), domain.ResendVerificationRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, application.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, errorResponse{
				Error: errorDetail{
					Code:    "invalid_credentials",
					Message: "メールアドレスまたはパスワードが正しくありません。",
				},
			})
			return
		}
		if errors.Is(err, application.ErrAlreadyVerified) {
			c.JSON(http.StatusBadRequest, errorResponse{
				Error: errorDetail{
					Code:    "already_verified",
					Message: "このメールアドレスは既に認証済みです。",
				},
			})
			return
		}
		if errors.Is(err, application.ErrRateLimitExceeded) {
			c.JSON(http.StatusTooManyRequests, errorResponse{
				Error: errorDetail{
					Code:    "rate_limit_exceeded",
					Message: "再送信の回数制限に達しました。15分後に再度お試しください。",
				},
			})
			return
		}
		if errors.Is(err, application.ErrMailSendFailed) {
			c.JSON(http.StatusInternalServerError, errorResponse{
				Error: errorDetail{
					Code:    "mail_send_failed",
					Message: "メール送信に失敗しました。しばらくしてから再度お試しください。",
				},
			})
			return
		}
		lc.logger.Error("ResendVerificationEmail error", logger.Err(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, resendVerificationResponse{
		Message: "確認メールを再送信しました。メールボックスをご確認ください。",
	})
}
