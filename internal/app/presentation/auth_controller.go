package presentation

import (
	"business/internal/auth/application"
	"business/internal/library/logger"
	"business/internal/library/oswrapper"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// AuthController handles login-related HTTP requests
type AuthController struct {
	usecase application.AuthUseCaseInterface
	logger  logger.Interface
	osw     oswrapper.OsWapperInterface
}

// NewAuthController creates a new AuthController instance
func NewAuthController(usecase application.AuthUseCaseInterface, log logger.Interface, osw oswrapper.OsWapperInterface) *AuthController {
	return &AuthController{
		usecase: usecase,
		logger:  log.With(logger.String("component", "auth_controller")),
		osw:     osw,
	}
}

// LoggerWithUserID returns a child logger with the user_id field attached
func (lc *AuthController) LoggerWithUserID(userID uint) logger.Interface {
	return lc.logger.With(logger.Uint("user_id", userID))
}

type checkResponse struct {
	UserID uint `json:"user_id"`
}

type userResponse struct {
	ID              uint       `json:"id"`
	Name            string     `json:"name"`
	Email           string     `json:"email"`
	EmailVerified   bool       `json:"email_verified"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Check handles the GET /auth/check endpoint.
func (lc *AuthController) Check(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		lc.logger.Error("Check error: userID not found in context")
		c.Status(http.StatusInternalServerError)
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		lc.logger.Error("Check error: userID type assertion failed")
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, checkResponse{UserID: uid})
}
