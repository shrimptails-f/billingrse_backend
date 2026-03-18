package auth

import (
	"business/internal/auth/application"
	"business/internal/library/logger"
	"business/internal/library/oswrapper"
	"time"
)

// Controller handles authentication-related HTTP requests.
type Controller struct {
	usecase application.AuthUseCaseInterface
	logger  logger.Interface
	osw     oswrapper.OsWapperInterface
}

// NewController creates a new Controller instance.
func NewController(usecase application.AuthUseCaseInterface, log logger.Interface, osw oswrapper.OsWapperInterface) *Controller {
	return &Controller{
		usecase: usecase,
		logger:  log.With(logger.Component("auth_controller")),
		osw:     osw,
	}
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
