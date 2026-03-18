package v1

import (
	"business/internal/app/middleware"
	authpresentation "business/internal/app/presentation/auth"
	"business/internal/library/logger"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/dig"
)

func NewRouter(g *gin.Engine, container *dig.Container, log logger.Interface) (*gin.Engine, error) {
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
	authGroup := g.Group("/auth")
	{
		authGroup.POST("/login", authController.Login)
		authGroup.POST("/logout", authController.Logout)
		authGroup.POST("/register", authController.Register)
		authGroup.GET("/email/verify", authController.VerifyEmail)
		authGroup.POST("/email/resend", authController.ResendVerificationEmail)
		authGroup.GET("/check", authMiddleware.Authenticate(), authController.Check)
	}

	return g, nil
}
