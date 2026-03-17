package middleware

import (
	"business/internal/auth/domain"
	"business/internal/library/logger"
	"business/internal/library/oswrapper"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// UserProvider provides user information for authentication checks
type UserProvider interface {
	GetUserByID(ctx context.Context, id uint) (domain.User, error)
}

// AuthMiddleware validates JWTs from incoming HTTP requests.
type AuthMiddleware struct {
	osw   oswrapper.OsWapperInterface
	users UserProvider
	log   logger.Interface
}

// NewAuthMiddleware wires a middleware instance.
func NewAuthMiddleware(osw oswrapper.OsWapperInterface, users UserProvider, log logger.Interface) *AuthMiddleware {
	if log == nil {
		log = logger.NewNop()
	}

	return &AuthMiddleware{
		osw:   osw,
		users: users,
		log:   log.With(logger.Component("auth_middleware")),
	}
}

// Authenticate returns a Gin middleware validating JWT tokens from the access_token cookie.
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqLog := m.log
		if requestLog, err := m.log.WithContext(c.Request.Context()); err == nil {
			reqLog = requestLog
		}

		cookieToken, err := c.Cookie("access_token")
		if err != nil || strings.TrimSpace(cookieToken) == "" {
			reqLog.Info("permission_denied", permissionDeniedFields(c, "missing_token")...)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(cookieToken)

		jwtSecret, err := m.osw.GetEnv("JWT_SECRET_KEY")
		if err != nil {
			reqLog.Error("failed to load jwt secret", logger.Err(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			c.Abort()
			return
		}

		token, err := jwt.ParseWithClaims(tokenString, &domain.AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			reqLog.Info("permission_denied", permissionDeniedFields(c, "invalid_token")...)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*domain.AuthClaims)
		if !ok {
			reqLog.Info("permission_denied", permissionDeniedFields(c, "invalid_token_claims")...)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		requestCtx, err := logger.ContextWithUserID(c.Request.Context(), claims.UserID)
		if err != nil {
			reqLog.Error("failed to attach user context", logger.Err(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			c.Abort()
			return
		}

		c.Request = c.Request.WithContext(requestCtx)
		if requestLog, err := m.log.WithContext(c.Request.Context()); err == nil {
			reqLog = requestLog
		}

		// Check if email verification is required for this path
		if m.requiresEmailVerification(c.Request.URL.Path) {
			// Get user to check email verification status
			user, err := m.users.GetUserByID(c.Request.Context(), claims.UserID)
			if err != nil {
				reqLog.Info("permission_denied", permissionDeniedFields(c, "user_not_found")...)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
				c.Abort()
				return
			}

			// Check if email is verified
			if !user.IsEmailVerified() {
				reqLog.Info("permission_denied", permissionDeniedFields(c, "email_verification_required")...)
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": gin.H{
						"code":    "email_verification_required",
						"message": "メールアドレスの認証が完了していません。確認メールのリンクから認証を完了してください。",
					},
				})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func permissionDeniedFields(c *gin.Context, reason string) []logger.Field {
	return []logger.Field{
		logger.String("reason", reason),
		logger.String("method", c.Request.Method),
		logger.String("path", resolvedPath(c)),
		logger.HTTPStatusCode(http.StatusUnauthorized),
	}
}

// requiresEmailVerification checks if the given path requires email verification
func (m *AuthMiddleware) requiresEmailVerification(path string) bool {
	// Skip email verification check for these paths
	skipPaths := []string{
		"/auth/register",
		"/auth/login",
		"/auth/email/verify",
		"/auth/email/resend",
	}

	for _, skipPath := range skipPaths {
		if path == skipPath {
			return false
		}
	}

	return true
}
