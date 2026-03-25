package router

import (
	"business/internal/app/middleware"
	authpresentation "business/internal/app/presentation/auth"
	billingpresentation "business/internal/app/presentation/billing"
	macpresentation "business/internal/app/presentation/mailaccountconnection"
	manualpresentation "business/internal/app/presentation/manualmailworkflow"
	"business/internal/library/logger"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/dig"
)

func Router(g *gin.Engine, container *dig.Container, log logger.Interface, allowedOrigin string) (*gin.Engine, error) {
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
		sessionGroup = group.Group("", middleware.CsrfOriginCheck(allowedOrigin))
		sessionGroup.POST("/refresh", authController.Refresh)
		sessionGroup.POST("/logout", authController.Logout)
	}
	registerAuthRoutes(g.Group("/api/v1/auth"))

	// MailAccountConnection関連
	var macController *macpresentation.Controller
	if err := container.Invoke(func(mc *macpresentation.Controller) {
		macController = mc
	}); err != nil {
		log.Error("failed to resolve mail account connection controller", logger.Err(err))
		return g, err
	}
	registerMailAccountConnectionRoutes := func(group *gin.RouterGroup) {
		group.GET("", authMiddleware.Authenticate(), macController.List)
		group.DELETE("/:connection_id", authMiddleware.Authenticate(), macController.Disconnect)
		group.POST("/gmail/authorize", authMiddleware.Authenticate(), macController.Authorize)
		group.POST("/gmail/callback", authMiddleware.Authenticate(), macController.Callback)
	}
	registerMailAccountConnectionRoutes(g.Group("/api/v1/mail-account-connections"))

	// ManualMailWorkflow関連
	var manualController *manualpresentation.Controller
	if err := container.Invoke(func(mc *manualpresentation.Controller) {
		manualController = mc
	}); err != nil {
		log.Error("failed to resolve manual mail workflow controller", logger.Err(err))
		return g, err
	}
	registerManualMailWorkflowRoutes := func(group *gin.RouterGroup) {
		group.GET("", authMiddleware.Authenticate(), manualController.List)
		group.POST("", authMiddleware.Authenticate(), manualController.Execute)
	}
	registerManualMailWorkflowRoutes(g.Group("/api/v1/manual-mail-workflows"))

	// Billing関連
	var billingController *billingpresentation.Controller
	if err := container.Invoke(func(bc *billingpresentation.Controller) {
		billingController = bc
	}); err != nil {
		log.Error("failed to resolve billing controller", logger.Err(err))
		return g, err
	}
	registerBillingRoutes := func(group *gin.RouterGroup) {
		group.GET("", authMiddleware.Authenticate(), billingController.List)
	}
	registerBillingRoutes(g.Group("/api/v1/billings"))

	return g, nil
}
