package presentation

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

// VerifyEmail handles the GET /auth/email/verify endpoint
func (lc *AuthController) VerifyEmail(c *gin.Context) {
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
		lc.logger.Error("VerifyEmail error", logger.Err(err))
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
