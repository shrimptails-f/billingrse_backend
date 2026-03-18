package auth

import (
	"business/internal/auth/application"
	"business/internal/auth/domain"
	"business/internal/library/logger"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type verifyEmailResponse struct {
	Message string       `json:"message"`
	User    userResponse `json:"user"`
}

type resendVerificationRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type resendVerificationResponse struct {
	Message string `json:"message"`
}

// VerifyEmail handles the GET /auth/email/verify endpoint.
func (lc *Controller) VerifyEmail(c *gin.Context) {
	reqLog, err := lc.logger.WithContext(c.Request.Context())
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	token := c.Query("token")

	if token == "" {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: errorDetail{
				Code:    "missing_token",
				Message: "トークンが指定されていません。",
			},
		})
		return
	}

	user, err := lc.usecase.VerifyEmail(c.Request.Context(), domain.VerifyEmailRequest{
		Token: token,
	})
	if err != nil {
		if errors.Is(err, application.ErrInvalidToken) {
			c.JSON(http.StatusBadRequest, errorResponse{
				Error: errorDetail{
					Code:    "invalid_token",
					Message: "無効なトークンです。",
				},
			})
			return
		}
		if errors.Is(err, application.ErrTokenExpired) {
			c.JSON(http.StatusBadRequest, errorResponse{
				Error: errorDetail{
					Code:    "token_expired",
					Message: "トークンの有効期限が切れています。再送信をお試しください。",
				},
			})
			return
		}
		if errors.Is(err, application.ErrTokenAlreadyUsed) {
			c.JSON(http.StatusBadRequest, errorResponse{
				Error: errorDetail{
					Code:    "token_already_used",
					Message: "このトークンは既に使用済みです。",
				},
			})
			return
		}
		reqLog.Error("VerifyEmail error", logger.Err(err))
		c.Status(http.StatusInternalServerError)
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

// ResendVerificationEmail handles the POST /auth/email/resend endpoint.
func (lc *Controller) ResendVerificationEmail(c *gin.Context) {
	var req resendVerificationRequest
	reqLog, err := lc.logger.WithContext(c.Request.Context())
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	err = lc.usecase.ResendVerificationEmail(c.Request.Context(), domain.ResendVerificationRequest{
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
		reqLog.Error("ResendVerificationEmail error", logger.Err(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, resendVerificationResponse{
		Message: "確認メールを再送信しました。メールボックスをご確認ください。",
	})
}
