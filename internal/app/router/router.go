package v1

import (
	"business/internal/app/middleware"
	"business/internal/app/presentation"
	"business/internal/library/logger"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/dig"
)

func NewRouter(g *gin.Engine, container *dig.Container, log logger.Interface) *gin.Engine {
	g.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
		// c.Status(http.StatusNoContent)

		// c.Status(http.StatusBadRequest)
		// c.JSON(http.StatusOK, gin.H{
		// 	"message": "hello world",
		// })
	})

	// AuthController と AuthMiddleware を先に解決
	var authController *presentation.AuthController
	var authMiddleware *middleware.AuthMiddleware
	if err := container.Invoke(func(lc *presentation.AuthController, am *middleware.AuthMiddleware) {
		authController = lc
		authMiddleware = am
	}); err != nil {
		log.Error("failed to resolve AuthController or AuthMiddleware", logger.Err(err))
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
	// // AccountLinkController を解決
	// var accountLinkController *presentation.AccountLinkController
	// if err := container.Invoke(func(alc *presentation.AccountLinkController) {
	// 	accountLinkController = alc
	// }); err != nil {
	// 	log.Error("failed to resolve AccountLinkController", logger.Err(err))
	// }

	// // /account-links エンドポイント (requires authentication)
	// g.GET("/account-links", authMiddleware.Authenticate(), accountLinkController.List)

	return g
}
