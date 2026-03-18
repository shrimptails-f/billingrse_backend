package v1

import (
	"business/internal/app/middleware"
	authpresentation "business/internal/app/presentation/auth"
	"business/internal/library/logger"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/dig"
)

func NewRouter(g *gin.Engine, container *dig.Container, log logger.Interface, allowedOrigins ...string) (*gin.Engine, error) {
	g.GET("/api/v1", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	g.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Auth関連
	var authController *authpresentation.Controller
	var authMiddleware *middleware.AuthMiddleware
	if err := container.Invoke(func(lc *authpresentation.Controller, am *middleware.AuthMiddleware) {
		authController = lc
		authMiddleware = am
	}); err != nil {
		log.Error("failed to resolve auth controller or auth middleware", logger.Err(err))
		return g, err
	}
	registerAuthRoutes := func(group *gin.RouterGroup) {
		group.POST("/login", authController.Login)
		group.POST("/register", authController.Register)
		group.POST("/email/verify", authController.VerifyEmail)
		group.POST("/email/resend", authController.ResendVerificationEmail)
		group.GET("/check", authMiddleware.Authenticate(), authController.Check)

		sessionGroup := group
		if len(allowedOrigins) > 0 {
			sessionGroup = group.Group("", middleware.CsrfOriginCheck(allowedOrigins...))
		}
		sessionGroup.POST("/refresh", authController.Refresh)
		sessionGroup.POST("/logout", authController.Logout)
	}
	registerAuthRoutes(g.Group("/api/v1/auth"))

	return g, nil
}
