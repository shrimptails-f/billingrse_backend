package auth

import (
	"business/internal/app/httpresponse"
	"business/internal/auth/application"
	"business/internal/auth/domain"
	"business/internal/library/logger"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	authCookiePath         = "/api/v1/auth"
	refreshTokenCookieName = "refresh_token"
)

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type authTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type checkResponse struct {
	UserID uint `json:"user_id"`
}

// Login handles the POST /api/v1/auth/login endpoint.
func (lc *Controller) Login(c *gin.Context) {
	var req loginRequest
	reqLog, err := lc.logger.WithContext(c.Request.Context())
	if err != nil {
		httpresponse.WriteInternalServerError(c)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		reqLog.Info("login_failed",
			logger.String("reason", "invalid_request"),
			logger.HTTPStatusCode(http.StatusBadRequest),
		)
		httpresponse.WriteInvalidRequest(c)
		return
	}

	tokens, err := lc.usecase.LoginTokens(c.Request.Context(), domain.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, application.ErrInvalidCredentials) {
			reqLog.Info("login_failed",
				logger.String("reason", "invalid_credentials"),
				logger.HTTPStatusCode(http.StatusUnauthorized),
			)
			httpresponse.WriteError(c, http.StatusUnauthorized, "invalid_credentials", "メールアドレスまたはパスワードが正しくありません。")
			return
		}
		reqLog.Error("login_failed",
			logger.String("reason", "login_usecase_error"),
			logger.HTTPStatusCode(http.StatusInternalServerError),
			logger.Err(err),
		)
		httpresponse.WriteInternalServerError(c)
		return
	}

	secure, err := lc.secureCookieEnabled()
	if err != nil {
		reqLog.Error("login_failed",
			logger.String("reason", "cookie_security_resolution_failed"),
			logger.HTTPStatusCode(http.StatusInternalServerError),
			logger.Err(err),
		)
		httpresponse.WriteInternalServerError(c)
		return
	}

	domain, err := lc.cookieDomain()
	if err != nil {
		reqLog.Error("login_failed",
			logger.String("reason", "cookie_domain_resolution_failed"),
			logger.HTTPStatusCode(http.StatusInternalServerError),
			logger.Err(err),
		)
		httpresponse.WriteInternalServerError(c)
		return
	}

	if err := lc.setRefreshTokenCookie(c, tokens.RefreshToken, int(tokens.RefreshTokenExpiresIn), secure, domain); err != nil {
		reqLog.Error("login_failed",
			logger.String("reason", "refresh_cookie_write_failed"),
			logger.HTTPStatusCode(http.StatusInternalServerError),
			logger.Err(err),
		)
		httpresponse.WriteInternalServerError(c)
		return
	}

	reqLog.Info("login_succeeded", logger.HTTPStatusCode(http.StatusOK))
	c.JSON(http.StatusOK, authTokenResponse{
		AccessToken: tokens.AccessToken,
		TokenType:   tokens.TokenType,
		ExpiresIn:   tokens.ExpiresIn,
	})
}

// Refresh handles the POST /api/v1/auth/refresh endpoint.
func (lc *Controller) Refresh(c *gin.Context) {
	reqLog, err := lc.logger.WithContext(c.Request.Context())
	if err != nil {
		httpresponse.WriteInternalServerError(c)
		return
	}

	refreshToken, err := c.Cookie(refreshTokenCookieName)
	if err != nil || strings.TrimSpace(refreshToken) == "" {
		reqLog.Info("refresh_failed",
			logger.String("reason", "missing_refresh_token"),
			logger.HTTPStatusCode(http.StatusUnauthorized),
		)
		httpresponse.AbortUnauthorized(c, "missing_refresh_token", "リフレッシュトークンがありません。")
		return
	}

	tokens, err := lc.usecase.Refresh(c.Request.Context(), domain.RefreshRequest{
		RefreshToken: strings.TrimSpace(refreshToken),
	})
	if err != nil {
		if clearErr := lc.clearRefreshTokenCookie(c); clearErr != nil {
			reqLog.Error("refresh_failed", logger.String("reason", "refresh_cookie_clear_failed"), logger.Err(clearErr))
			httpresponse.WriteInternalServerError(c)
			return
		}
		if errors.Is(err, application.ErrRefreshTokenInvalid) {
			reqLog.Info("refresh_failed",
				logger.String("reason", "invalid_refresh_token"),
				logger.HTTPStatusCode(http.StatusUnauthorized),
			)
			httpresponse.AbortUnauthorized(c, "invalid_refresh_token", "リフレッシュトークンが無効です。")
			return
		}
		reqLog.Error("refresh_failed",
			logger.String("reason", "refresh_usecase_error"),
			logger.HTTPStatusCode(http.StatusInternalServerError),
			logger.Err(err),
		)
		httpresponse.WriteInternalServerError(c)
		return
	}

	secure, err := lc.secureCookieEnabled()
	if err != nil {
		reqLog.Error("refresh_failed",
			logger.String("reason", "cookie_security_resolution_failed"),
			logger.HTTPStatusCode(http.StatusInternalServerError),
			logger.Err(err),
		)
		httpresponse.WriteInternalServerError(c)
		return
	}

	domain, err := lc.cookieDomain()
	if err != nil {
		reqLog.Error("refresh_failed",
			logger.String("reason", "cookie_domain_resolution_failed"),
			logger.HTTPStatusCode(http.StatusInternalServerError),
			logger.Err(err),
		)
		httpresponse.WriteInternalServerError(c)
		return
	}

	if err := lc.setRefreshTokenCookie(c, tokens.RefreshToken, int(tokens.RefreshTokenExpiresIn), secure, domain); err != nil {
		reqLog.Error("refresh_failed",
			logger.String("reason", "refresh_cookie_write_failed"),
			logger.HTTPStatusCode(http.StatusInternalServerError),
			logger.Err(err),
		)
		httpresponse.WriteInternalServerError(c)
		return
	}

	reqLog.Info("refresh_succeeded", logger.HTTPStatusCode(http.StatusOK))
	c.JSON(http.StatusOK, authTokenResponse{
		AccessToken: tokens.AccessToken,
		TokenType:   tokens.TokenType,
		ExpiresIn:   tokens.ExpiresIn,
	})
}

// Logout handles the POST /api/v1/auth/logout endpoint.
func (lc *Controller) Logout(c *gin.Context) {
	reqLog, err := lc.logger.WithContext(c.Request.Context())
	if err != nil {
		httpresponse.WriteInternalServerError(c)
		return
	}

	refreshToken, err := c.Cookie(refreshTokenCookieName)
	if err == nil && strings.TrimSpace(refreshToken) != "" {
		invokeErr := lc.usecase.Logout(c.Request.Context(), domain.LogoutRequest{
			RefreshToken: strings.TrimSpace(refreshToken),
		})
		if invokeErr != nil {
			reqLog.Error("logout_failed",
				logger.String("reason", "logout_usecase_error"),
				logger.HTTPStatusCode(http.StatusInternalServerError),
				logger.Err(invokeErr),
			)
			if clearErr := lc.clearRefreshTokenCookie(c); clearErr != nil {
				reqLog.Error("logout_failed", logger.String("reason", "refresh_cookie_clear_failed"), logger.Err(clearErr))
				httpresponse.WriteInternalServerError(c)
				return
			}
			httpresponse.WriteInternalServerError(c)
			return
		}
	}

	if clearErr := lc.clearRefreshTokenCookie(c); clearErr != nil {
		reqLog.Error("logout_failed", logger.String("reason", "refresh_cookie_clear_failed"), logger.Err(clearErr))
		httpresponse.WriteInternalServerError(c)
		return
	}

	reqLog.Info("logout_succeeded", logger.HTTPStatusCode(http.StatusNoContent))
	c.Status(http.StatusNoContent)
}

// Check handles the GET /api/v1/auth/check endpoint.
func (lc *Controller) Check(c *gin.Context) {
	reqLog, err := lc.logger.WithContext(c.Request.Context())
	if err != nil {
		httpresponse.WriteInternalServerError(c)
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		reqLog.Error("Check error: userID not found in context")
		httpresponse.WriteInternalServerError(c)
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		reqLog.Error("Check error: userID type assertion failed")
		httpresponse.WriteInternalServerError(c)
		return
	}

	c.JSON(http.StatusOK, checkResponse{UserID: uid})
}

func (lc *Controller) secureCookieEnabled() (bool, error) {
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

func (lc *Controller) cookieDomain() (string, error) {
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

func (lc *Controller) setRefreshTokenCookie(c *gin.Context, token string, maxAge int, secure bool, domain string) error {
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("refresh token is empty")
	}
	if maxAge <= 0 {
		return fmt.Errorf("refresh token max age is invalid: %d", maxAge)
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		refreshTokenCookieName,
		strings.TrimSpace(token),
		maxAge,
		authCookiePath,
		domain,
		secure,
		true,
	)
	return nil
}

func (lc *Controller) clearRefreshTokenCookie(c *gin.Context) error {
	secure, err := lc.secureCookieEnabled()
	if err != nil {
		return err
	}

	domain, err := lc.cookieDomain()
	if err != nil {
		return err
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		refreshTokenCookieName,
		"",
		-1,
		authCookiePath,
		domain,
		secure,
		true,
	)
	return nil
}
