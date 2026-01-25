package presentation

import (
	"net/http"

	"business/internal/library/logger"

	"github.com/gin-gonic/gin"
)

// Logout handles the POST /auth/logout endpoint
func (lc *AuthController) Logout(c *gin.Context) {
	secure, err := lc.secureCookieEnabled()
	if err != nil {
		lc.logger.Error("failed to determine cookie security", logger.Err(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	domain, err := lc.cookieDomain()
	if err != nil {
		lc.logger.Error("failed to determine cookie domain", logger.Err(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"access_token",
		"",
		-1, /* delete cookie */
		"/",
		domain,
		secure, /* Secure */
		true,   /* HttpOnly */
	)
	c.Status(http.StatusNoContent)
}
