package mailaccountconnection

import (
	"business/internal/app/httpresponse"
	"business/internal/library/logger"
	"business/internal/mailaccountconnection/application"
	"business/internal/mailaccountconnection/domain"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Controller handles mail account connection HTTP requests.
type Controller struct {
	usecase application.UseCaseInterface
	log     logger.Interface
}

// NewController creates a new Controller.
func NewController(usecase application.UseCaseInterface, log logger.Interface) *Controller {
	if log == nil {
		log = logger.NewNop()
	}
	return &Controller{
		usecase: usecase,
		log:     log.With(logger.Component("mail_account_connection_controller")),
	}
}

type authorizeResponse struct {
	AuthorizationURL string    `json:"authorization_url"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type callbackRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

type callbackResponse struct {
	Message string `json:"message"`
}

type listConnectionsResponse struct {
	Items []connectionResponseItem `json:"items"`
}

type connectionResponseItem struct {
	ID                uint      `json:"id"`
	Provider          string    `json:"provider"`
	AccountIdentifier string    `json:"account_identifier"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Authorize handles POST /api/v1/mail-account-connections/gmail/authorize
func (ctrl *Controller) Authorize(c *gin.Context) {
	reqLog := ctrl.log
	if l, err := ctrl.log.WithContext(c.Request.Context()); err == nil {
		reqLog = l
	}

	uid, ok := currentUserID(c)
	if !ok {
		return
	}

	result, err := ctrl.usecase.Authorize(c.Request.Context(), uid)
	if err != nil {
		reqLog.Error("authorize_failed", logger.Err(err))
		httpresponse.WriteInternalServerError(c)
		return
	}

	c.JSON(http.StatusOK, authorizeResponse{
		AuthorizationURL: result.AuthorizationURL,
		ExpiresAt:        result.ExpiresAt,
	})
}

// Callback handles POST /api/v1/mail-account-connections/gmail/callback
func (ctrl *Controller) Callback(c *gin.Context) {
	reqLog := ctrl.log
	if l, err := ctrl.log.WithContext(c.Request.Context()); err == nil {
		reqLog = l
	}

	uid, ok := currentUserID(c)
	if !ok {
		return
	}

	var req callbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.WriteInvalidRequest(c)
		return
	}

	err := ctrl.usecase.Callback(c.Request.Context(), uid, req.Code, req.State)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrOAuthStateMismatch):
			httpresponse.WriteError(c, http.StatusConflict, "oauth_state_mismatch", "OAuth stateが一致しません。")
		case errors.Is(err, domain.ErrOAuthStateExpired):
			httpresponse.WriteError(c, http.StatusConflict, "oauth_state_expired", "OAuth stateの有効期限が切れています。")
		case errors.Is(err, domain.ErrOAuthExchangeFailed):
			httpresponse.WriteServiceUnavailable(c, "gmail_oauth_exchange_failed", "Googleとのトークン交換に失敗しました。しばらくしてから再度お試しください。")
		case errors.Is(err, domain.ErrGmailProfileFetchFailed):
			httpresponse.WriteServiceUnavailable(c, "gmail_profile_fetch_failed", "Gmailプロフィールの取得に失敗しました。しばらくしてから再度お試しください。")
		case errors.Is(err, domain.ErrRefreshTokenMissing):
			httpresponse.WriteServiceUnavailable(c, "gmail_oauth_exchange_failed", "Googleからリフレッシュトークンが取得できませんでした。再度お試しください。")
		case errors.Is(err, domain.ErrVaultEncryptFailed):
			reqLog.Error("vault_encrypt_failed", logger.Err(err))
			httpresponse.WriteInternalServerError(c)
		default:
			reqLog.Error("callback_failed", logger.Err(err))
			httpresponse.WriteInternalServerError(c)
		}
		return
	}

	c.JSON(http.StatusOK, callbackResponse{
		Message: "Gmail連携が完了しました。",
	})
}

// List handles GET /api/v1/mail-account-connections
func (ctrl *Controller) List(c *gin.Context) {
	reqLog := ctrl.log
	if l, err := ctrl.log.WithContext(c.Request.Context()); err == nil {
		reqLog = l
	}

	uid, ok := currentUserID(c)
	if !ok {
		return
	}

	connections, err := ctrl.usecase.ListConnections(c.Request.Context(), uid)
	if err != nil {
		reqLog.Error("list_connections_failed", logger.Err(err))
		httpresponse.WriteInternalServerError(c)
		return
	}

	items := make([]connectionResponseItem, 0, len(connections))
	for _, connection := range connections {
		items = append(items, connectionResponseItem{
			ID:                connection.ID,
			Provider:          connection.Provider,
			AccountIdentifier: connection.AccountIdentifier,
			CreatedAt:         connection.CreatedAt,
			UpdatedAt:         connection.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, listConnectionsResponse{Items: items})
}

// Disconnect handles DELETE /api/v1/mail-account-connections/:connection_id
func (ctrl *Controller) Disconnect(c *gin.Context) {
	reqLog := ctrl.log
	if l, err := ctrl.log.WithContext(c.Request.Context()); err == nil {
		reqLog = l
	}

	uid, ok := currentUserID(c)
	if !ok {
		return
	}

	connectionID, ok := currentConnectionID(c)
	if !ok {
		return
	}

	err := ctrl.usecase.Disconnect(c.Request.Context(), uid, connectionID)
	if err != nil {
		if errors.Is(err, domain.ErrCredentialNotFound) {
			httpresponse.WriteError(c, http.StatusNotFound, "mail_account_connection_not_found", "対象のメール連携は見つかりません。")
			return
		}

		reqLog.Error("disconnect_failed",
			logger.UserID(uid),
			logger.Uint("connection_id", connectionID),
			logger.Err(err),
		)
		httpresponse.WriteInternalServerError(c)
		return
	}

	c.Status(http.StatusNoContent)
}

func currentUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		httpresponse.WriteError(c, http.StatusUnauthorized, "unauthorized", "認証が必要です。")
		return 0, false
	}

	uid, ok := userID.(uint)
	if !ok {
		httpresponse.WriteInternalServerError(c)
		return 0, false
	}

	return uid, true
}

func currentConnectionID(c *gin.Context) (uint, bool) {
	rawID := c.Param("connection_id")
	connectionID, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || connectionID == 0 {
		httpresponse.WriteInvalidRequest(c)
		return 0, false
	}

	return uint(connectionID), true
}
