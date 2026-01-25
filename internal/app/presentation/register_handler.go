package presentation

import (
	"business/internal/auth/application"
	"business/internal/auth/domain"
	"business/internal/library/logger"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type registerResponse struct {
	Message string       `json:"message"`
	User    userResponse `json:"user"`
}

// Register handles the POST /auth/register endpoint
func (lc *AuthController) Register(c *gin.Context) {
	var req registerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	user, err := lc.usecase.Register(c.Request.Context(), domain.RegisterRequest{
		Email:    req.Email,
		Name:     req.Name,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, application.ErrEmailAlreadyExists) {
			c.JSON(http.StatusBadRequest, errorResponse{
				Error: errorDetail{
					Code:    "email_already_exists",
					Message: "このメールアドレスは既に登録されています。",
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
		lc.logger.Error("Register error", logger.Err(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, registerResponse{
		Message: "登録が完了しました。確認メールを送信しましたので、メールアドレスの認証を完了してください。",
		User: userResponse{
			ID:              user.ID,
			Name:            user.Name,
			Email:           user.Email,
			EmailVerified:   user.EmailVerified,
			EmailVerifiedAt: user.EmailVerifiedAt,
			CreatedAt:       user.CreatedAt,
		},
	})
}
