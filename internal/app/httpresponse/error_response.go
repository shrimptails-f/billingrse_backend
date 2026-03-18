package httpresponse

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse is the standard error envelope for HTTP APIs.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail describes a machine-readable and user-facing HTTP error.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

const (
	CodeInvalidRequest       = "invalid_request"
	CodeInternalServerError  = "internal_server_error"
	MessageInvalidRequest    = "入力値が不正です。"
	MessageInternalServerErr = "サーバー内部でエラーが発生しました。しばらくしてから再度お試しください。"
)

// WriteError writes the standard error envelope without aborting the handler chain.
func WriteError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// AbortError aborts the handler chain with the standard error envelope.
func AbortError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// WriteInvalidRequest writes the standard invalid request response.
func WriteInvalidRequest(c *gin.Context) {
	WriteError(c, http.StatusBadRequest, CodeInvalidRequest, MessageInvalidRequest)
}

// WriteInternalServerError writes the standard internal server error response.
func WriteInternalServerError(c *gin.Context) {
	WriteError(c, http.StatusInternalServerError, CodeInternalServerError, MessageInternalServerErr)
}

// WriteServiceUnavailable writes a 503 response using the standard error envelope.
func WriteServiceUnavailable(c *gin.Context, code, message string) {
	WriteError(c, http.StatusServiceUnavailable, code, message)
}

// AbortUnauthorized aborts with a 401 response using the standard error envelope.
func AbortUnauthorized(c *gin.Context, code, message string) {
	AbortError(c, http.StatusUnauthorized, code, message)
}

// AbortForbidden aborts with a 403 response using the standard error envelope.
func AbortForbidden(c *gin.Context, code, message string) {
	AbortError(c, http.StatusForbidden, code, message)
}

// AbortInternalServerError aborts with the standard internal server error response.
func AbortInternalServerError(c *gin.Context) {
	AbortError(c, http.StatusInternalServerError, CodeInternalServerError, MessageInternalServerErr)
}
