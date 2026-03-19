package middleware

import (
	"business/internal/app/httpresponse"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

func CsrfOriginCheck(allowedOrigin string) gin.HandlerFunc {
	allow := make(map[string]struct{}, len(allowedOrigin))
	if allowedOrigin != "" {
		allow[allowedOrigin] = struct{}{}
	}
	ok := func(v string) bool {
		if v == "" {
			return false
		}
		u, err := url.Parse(v)
		if err != nil {
			return false
		}
		host := u.Scheme + "://" + u.Host
		_, allowed := allow[host]
		return allowed
	}

	return func(c *gin.Context) {
		// Only enforce CSRF checks for state-changing requests.
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			c.Next()
			return
		}

		origin := c.Request.Header.Get("Origin")
		referer := c.Request.Header.Get("Referer")

		if ok(origin) || ok(referer) {
			c.Next()
			return
		}

		httpresponse.AbortForbidden(c, errorCodeCSRFOriginNotAllowed, errorMessageCSRFOriginNotAllowed)
	}
}
