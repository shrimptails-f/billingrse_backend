package billing

import (
	"business/internal/app/httpresponse"
	billingapp "business/internal/billing/application"
	billingdomain "business/internal/billing/domain"
	"business/internal/library/logger"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Controller handles billing list HTTP requests.
type Controller struct {
	usecase billingapp.ListUseCase
	log     logger.Interface
}

// NewController creates a new billing controller.
func NewController(usecase billingapp.ListUseCase, log logger.Interface) *Controller {
	if log == nil {
		log = logger.NewNop()
	}

	return &Controller{
		usecase: usecase,
		log:     log.With(logger.Component("billing_controller")),
	}
}

type listQueryRequest struct {
	Q                     string `form:"q"`
	EmailID               string `form:"email_id"`
	ExternalMessageID     string `form:"external_message_id"`
	DateFrom              string `form:"date_from"`
	DateTo                string `form:"date_to"`
	UseReceivedAtFallback string `form:"use_received_at_fallback"`
	Limit                 string `form:"limit"`
	Offset                string `form:"offset"`
}

type listResponse struct {
	Items      []listResponseItem `json:"items"`
	Limit      int                `json:"limit"`
	Offset     int                `json:"offset"`
	TotalCount int64              `json:"total_count"`
}

type listResponseItem struct {
	EmailID            uint       `json:"email_id"`
	ExternalMessageID  string     `json:"external_message_id"`
	VendorName         string     `json:"vendor_name"`
	ReceivedAt         time.Time  `json:"received_at"`
	BillingDate        *time.Time `json:"billing_date"`
	ProductNameDisplay *string    `json:"product_name_display"`
	Amount             float64    `json:"amount"`
	Currency           string     `json:"currency"`
}

// List handles GET /api/v1/billings.
func (ctrl *Controller) List(c *gin.Context) {
	reqLog := ctrl.log
	if withContext, err := ctrl.log.WithContext(c.Request.Context()); err == nil {
		reqLog = withContext
	}

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req listQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httpresponse.WriteInvalidRequest(c)
		return
	}

	query, err := req.toApplicationQuery(userID)
	if err != nil {
		httpresponse.WriteInvalidRequest(c)
		return
	}

	result, err := ctrl.usecase.List(c.Request.Context(), query)
	if err != nil {
		if errors.Is(err, billingdomain.ErrInvalidListQuery) {
			httpresponse.WriteInvalidRequest(c)
			return
		}

		reqLog.Error("list_billings_failed",
			logger.UserID(userID),
			logger.Err(err),
		)
		httpresponse.WriteInternalServerError(c)
		return
	}

	items := make([]listResponseItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, listResponseItem{
			EmailID:            item.EmailID,
			ExternalMessageID:  item.ExternalMessageID,
			VendorName:         item.VendorName,
			ReceivedAt:         item.ReceivedAt,
			BillingDate:        cloneTime(item.BillingDate),
			ProductNameDisplay: cloneString(item.ProductNameDisplay),
			Amount:             item.Amount,
			Currency:           item.Currency,
		})
	}

	c.JSON(http.StatusOK, listResponse{
		Items:      items,
		Limit:      result.Limit,
		Offset:     result.Offset,
		TotalCount: result.TotalCount,
	})
}

func (r listQueryRequest) toApplicationQuery(userID uint) (billingapp.ListQuery, error) {
	emailID, err := parseOptionalUint(r.EmailID)
	if err != nil {
		return billingapp.ListQuery{}, err
	}
	dateFrom, err := parseOptionalRFC3339(r.DateFrom)
	if err != nil {
		return billingapp.ListQuery{}, err
	}
	dateTo, err := parseOptionalRFC3339(r.DateTo)
	if err != nil {
		return billingapp.ListQuery{}, err
	}
	useReceivedAtFallback, err := parseOptionalBool(r.UseReceivedAtFallback)
	if err != nil {
		return billingapp.ListQuery{}, err
	}
	limit, err := parseOptionalInt(r.Limit)
	if err != nil {
		return billingapp.ListQuery{}, err
	}
	offset, err := parseOptionalInt(r.Offset)
	if err != nil {
		return billingapp.ListQuery{}, err
	}

	return billingapp.ListQuery{
		UserID:                userID,
		Q:                     r.Q,
		EmailID:               emailID,
		ExternalMessageID:     r.ExternalMessageID,
		DateFrom:              dateFrom,
		DateTo:                dateTo,
		UseReceivedAtFallback: useReceivedAtFallback,
		Limit:                 limit,
		Offset:                offset,
	}, nil
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

func parseOptionalUint(raw string) (*uint, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil {
		return nil, err
	}

	value := uint(parsed)
	return &value, nil
}

func parseOptionalInt(raw string) (*int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	value, err := strconv.Atoi(trimmed)
	if err != nil {
		return nil, err
	}

	return &value, nil
}

func parseOptionalRFC3339(raw string) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, err
	}

	utc := parsed.UTC()
	return &utc, nil
}

func parseOptionalBool(raw string) (*bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	switch strings.ToLower(trimmed) {
	case "true":
		value := true
		return &value, nil
	case "false":
		value := false
		return &value, nil
	default:
		return nil, strconv.ErrSyntax
	}
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	cloned := value.UTC()
	return &cloned
}
