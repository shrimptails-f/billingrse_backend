package presentation

import (
	"business/internal/auth/application"
	"business/internal/auth/domain"
	"business/internal/library/logger"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login handles the POST /auth/login endpoint
func (lc *AuthController) Login(c *gin.Context) {
	var req loginRequest
	reqLog, err := lc.logger.WithContext(c.Request.Context())
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		reqLog.Info("login_failed",
			logger.String("reason", "invalid_request"),
			logger.HTTPStatusCode(http.StatusBadRequest),
		)
		c.Status(http.StatusBadRequest)
		return
	}

	token, err := lc.usecase.Login(c.Request.Context(), domain.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, application.ErrInvalidCredentials) {
			reqLog.Info("login_failed",
				logger.String("reason", "invalid_credentials"),
				logger.HTTPStatusCode(http.StatusUnauthorized),
			)
			c.Status(http.StatusUnauthorized)
			return
		}
		reqLog.Error("login_failed",
			logger.String("reason", "login_usecase_error"),
			logger.HTTPStatusCode(http.StatusInternalServerError),
			logger.Err(err),
		)
		c.Status(http.StatusInternalServerError)
		return
	}

	maxAge := 86400 // 1日の秒数
	secure, err := lc.secureCookieEnabled()
	if err != nil {
		reqLog.Error("login_failed",
			logger.String("reason", "cookie_security_resolution_failed"),
			logger.HTTPStatusCode(http.StatusInternalServerError),
			logger.Err(err),
		)
		c.Status(http.StatusInternalServerError)
		return
	}

	domain, err := lc.cookieDomain()
	if err != nil {
		reqLog.Error("login_failed",
			logger.String("reason", "cookie_domain_resolution_failed"),
			logger.HTTPStatusCode(http.StatusInternalServerError),
			logger.Err(err),
		)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"access_token",
		token,
		maxAge, /*有効期限*/
		"/",
		domain,
		secure, /*Secure*/
		true,   /*HttpOnly*/
	)
	reqLog.Info("login_succeeded", logger.HTTPStatusCode(http.StatusNoContent))
	c.Status(http.StatusNoContent)
}

// secureCookieEnabled returns true when cookies should be marked as Secure.
func (lc *AuthController) secureCookieEnabled() (bool, error) {
	if lc.osw == nil {
		return false, errors.New("os wrapper is nil")
	}

	app, err := lc.osw.GetEnv("APP")
	if err != nil {
		return false, err
	}

	app = strings.TrimSpace(app)
	if app == "" {
		return false, nil
	}

	return app != "local", nil
}

func (lc *AuthController) cookieDomain() (string, error) {
	if lc.osw == nil {
		return "", errors.New("os wrapper is nil")
	}

	domain, err := lc.osw.GetEnv("DOMAIN")
	if err != nil {
		return "", err
	}

	domain = strings.TrimSpace(domain)
	if domain == "" {
		return "", errors.New("DOMAIN is empty")
	}

	return domain, nil
}
