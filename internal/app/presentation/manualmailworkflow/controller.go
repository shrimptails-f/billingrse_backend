package manualmailworkflow

import (
	"business/internal/app/httpresponse"
	"business/internal/library/logger"
	mfdomain "business/internal/mailfetch/domain"
	manualapp "business/internal/manualmailworkflow/application"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Controller handles manual mail workflow HTTP requests.
type Controller struct {
	usecase manualapp.UseCase
	log     logger.Interface
}

// NewController creates a new Controller.
func NewController(usecase manualapp.UseCase, log logger.Interface) *Controller {
	if log == nil {
		log = logger.NewNop()
	}

	return &Controller{
		usecase: usecase,
		log:     log.With(logger.Component("manual_mail_workflow_controller")),
	}
}

type executeRequest struct {
	ConnectionID uint      `json:"connection_id" binding:"required"`
	LabelName    string    `json:"label_name" binding:"required"`
	Since        time.Time `json:"since" binding:"required"`
	Until        time.Time `json:"until" binding:"required"`
}

type executeResponse struct {
	Message            string                            `json:"message"`
	Fetch              fetchSummaryResponse              `json:"fetch"`
	Analysis           analysisSummaryResponse           `json:"analysis"`
	VendorResolution   vendorResolutionSummaryResponse   `json:"vendor_resolution"`
	BillingEligibility billingEligibilitySummaryResponse `json:"billing_eligibility"`
	Billing            billingSummaryResponse            `json:"billing"`
}

type fetchSummaryResponse struct {
	Provider            string                 `json:"provider"`
	AccountIdentifier   string                 `json:"account_identifier"`
	MatchedMessageCount int                    `json:"matched_message_count"`
	CreatedEmailCount   int                    `json:"created_email_count"`
	CreatedEmailIDs     []uint                 `json:"created_email_ids"`
	ExistingEmailCount  int                    `json:"existing_email_count"`
	ExistingEmailIDs    []uint                 `json:"existing_email_ids"`
	FailureCount        int                    `json:"failure_count"`
	Failures            []fetchFailureResponse `json:"failures"`
}

type fetchFailureResponse struct {
	ExternalMessageID string `json:"external_message_id"`
	Stage             string `json:"stage"`
	Code              string `json:"code"`
}

type analysisSummaryResponse struct {
	AnalyzedEmailCount int                       `json:"analyzed_email_count"`
	ParsedEmailCount   int                       `json:"parsed_email_count"`
	ParsedEmailIDs     []uint                    `json:"parsed_email_ids"`
	FailureCount       int                       `json:"failure_count"`
	Failures           []analysisFailureResponse `json:"failures"`
}

type analysisFailureResponse struct {
	EmailID           uint   `json:"email_id"`
	ExternalMessageID string `json:"external_message_id"`
	Stage             string `json:"stage"`
	Code              string `json:"code"`
}

type vendorResolutionSummaryResponse struct {
	ResolvedCount                int                               `json:"resolved_count"`
	ResolvedItems                []vendorResolutionResolvedItem    `json:"resolved_items"`
	UnresolvedCount              int                               `json:"unresolved_count"`
	UnresolvedExternalMessageIDs []string                          `json:"unresolved_external_message_ids"`
	FailureCount                 int                               `json:"failure_count"`
	Failures                     []vendorResolutionFailureResponse `json:"failures"`
}

type vendorResolutionResolvedItem struct {
	ParsedEmailID     uint   `json:"parsed_email_id"`
	EmailID           uint   `json:"email_id"`
	ExternalMessageID string `json:"external_message_id"`
	VendorID          uint   `json:"vendor_id"`
	VendorName        string `json:"vendor_name"`
	MatchedBy         string `json:"matched_by"`
}

type vendorResolutionFailureResponse struct {
	ParsedEmailID     uint   `json:"parsed_email_id"`
	EmailID           uint   `json:"email_id"`
	ExternalMessageID string `json:"external_message_id"`
	Stage             string `json:"stage"`
	Code              string `json:"code"`
}

type billingEligibilitySummaryResponse struct {
	EligibleCount   int                                 `json:"eligible_count"`
	EligibleItems   []billingEligibilityEligibleItem    `json:"eligible_items"`
	IneligibleCount int                                 `json:"ineligible_count"`
	IneligibleItems []billingEligibilityIneligibleItem  `json:"ineligible_items"`
	FailureCount    int                                 `json:"failure_count"`
	Failures        []billingEligibilityFailureResponse `json:"failures"`
}

type billingEligibilityEligibleItem struct {
	ParsedEmailID     uint       `json:"parsed_email_id"`
	EmailID           uint       `json:"email_id"`
	ExternalMessageID string     `json:"external_message_id"`
	VendorID          uint       `json:"vendor_id"`
	VendorName        string     `json:"vendor_name"`
	MatchedBy         string     `json:"matched_by"`
	BillingNumber     string     `json:"billing_number"`
	InvoiceNumber     *string    `json:"invoice_number"`
	Amount            float64    `json:"amount"`
	Currency          string     `json:"currency"`
	BillingDate       *time.Time `json:"billing_date"`
	PaymentCycle      string     `json:"payment_cycle"`
}

type billingEligibilityIneligibleItem struct {
	ParsedEmailID     uint   `json:"parsed_email_id"`
	EmailID           uint   `json:"email_id"`
	ExternalMessageID string `json:"external_message_id"`
	VendorID          uint   `json:"vendor_id"`
	VendorName        string `json:"vendor_name"`
	MatchedBy         string `json:"matched_by"`
	ReasonCode        string `json:"reason_code"`
}

type billingEligibilityFailureResponse struct {
	ParsedEmailID     uint   `json:"parsed_email_id"`
	EmailID           uint   `json:"email_id"`
	ExternalMessageID string `json:"external_message_id"`
	Stage             string `json:"stage"`
	Code              string `json:"code"`
}

type billingSummaryResponse struct {
	CreatedCount   int                      `json:"created_count"`
	CreatedItems   []billingCreatedItem     `json:"created_items"`
	DuplicateCount int                      `json:"duplicate_count"`
	DuplicateItems []billingDuplicateItem   `json:"duplicate_items"`
	FailureCount   int                      `json:"failure_count"`
	Failures       []billingFailureResponse `json:"failures"`
}

type billingCreatedItem struct {
	BillingID         uint   `json:"billing_id"`
	ParsedEmailID     uint   `json:"parsed_email_id"`
	EmailID           uint   `json:"email_id"`
	ExternalMessageID string `json:"external_message_id"`
	VendorID          uint   `json:"vendor_id"`
	VendorName        string `json:"vendor_name"`
	BillingNumber     string `json:"billing_number"`
}

type billingDuplicateItem struct {
	ExistingBillingID uint   `json:"existing_billing_id"`
	ParsedEmailID     uint   `json:"parsed_email_id"`
	EmailID           uint   `json:"email_id"`
	ExternalMessageID string `json:"external_message_id"`
	VendorID          uint   `json:"vendor_id"`
	VendorName        string `json:"vendor_name"`
	BillingNumber     string `json:"billing_number"`
}

type billingFailureResponse struct {
	ParsedEmailID     uint   `json:"parsed_email_id"`
	EmailID           uint   `json:"email_id"`
	ExternalMessageID string `json:"external_message_id"`
	Stage             string `json:"stage"`
	Code              string `json:"code"`
}

// Execute handles POST /api/v1/manual-mail-workflows.
func (ctrl *Controller) Execute(c *gin.Context) {
	reqLog := ctrl.log
	if l, err := ctrl.log.WithContext(c.Request.Context()); err == nil {
		reqLog = l
	}

	uid, ok := currentUserID(c)
	if !ok {
		return
	}

	var req executeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.WriteInvalidRequest(c)
		return
	}

	result, err := ctrl.usecase.Execute(c.Request.Context(), manualapp.Command{
		UserID:       uid,
		ConnectionID: req.ConnectionID,
		Condition: manualapp.FetchCondition{
			LabelName: req.LabelName,
			Since:     req.Since,
			Until:     req.Until,
		},
	})
	if err != nil {
		ctrl.writeExecutionError(c, reqLog, uid, req.ConnectionID, err)
		return
	}

	c.JSON(http.StatusOK, executeResponse{
		Message:            "メール取得ワークフローが完了しました。",
		Fetch:              buildFetchSummaryResponse(result.Fetch),
		Analysis:           buildAnalysisSummaryResponse(result.Analysis),
		VendorResolution:   buildVendorResolutionSummaryResponse(result.VendorResolution),
		BillingEligibility: buildBillingEligibilitySummaryResponse(result.BillingEligibility),
		Billing:            buildBillingSummaryResponse(result.Billing),
	})
}

func (ctrl *Controller) writeExecutionError(c *gin.Context, reqLog logger.Interface, userID, connectionID uint, err error) {
	switch {
	case errors.Is(err, manualapp.ErrInvalidCommand), errors.Is(err, manualapp.ErrFetchConditionInvalid), errors.Is(err, mfdomain.ErrInvalidCommand), errors.Is(err, mfdomain.ErrFetchConditionInvalid):
		httpresponse.WriteInvalidRequest(c)
	case errors.Is(err, mfdomain.ErrConnectionNotFound):
		httpresponse.WriteError(c, http.StatusNotFound, "mail_account_connection_not_found", "対象のメール連携は見つかりません。")
	case errors.Is(err, mfdomain.ErrConnectionUnavailable):
		httpresponse.WriteError(c, http.StatusForbidden, "mail_account_connection_unavailable", "対象のメール連携は現在利用できません。")
	case errors.Is(err, mfdomain.ErrProviderLabelNotFound):
		httpresponse.WriteError(c, http.StatusBadRequest, "mail_label_not_found", "指定したラベルは見つかりません。")
	case errors.Is(err, mfdomain.ErrProviderSessionBuildFailed), errors.Is(err, mfdomain.ErrProviderListFailed):
		httpresponse.WriteServiceUnavailable(c, "mail_provider_unavailable", "メールプロバイダへの接続に失敗しました。しばらくしてから再度お試しください。")
	default:
		reqLog.Error("manual_mail_workflow_failed",
			logger.UserID(userID),
			logger.Uint("connection_id", connectionID),
			logger.Err(err),
		)
		httpresponse.WriteInternalServerError(c)
	}
}

func buildFetchSummaryResponse(result manualapp.FetchResult) fetchSummaryResponse {
	createdEmailIDs := make([]uint, 0, len(result.CreatedEmailIDs))
	createdEmailIDs = append(createdEmailIDs, result.CreatedEmailIDs...)

	existingEmailIDs := make([]uint, 0, len(result.ExistingEmailIDs))
	existingEmailIDs = append(existingEmailIDs, result.ExistingEmailIDs...)

	failures := make([]fetchFailureResponse, 0, len(result.Failures))
	for _, failure := range result.Failures {
		failures = append(failures, fetchFailureResponse{
			ExternalMessageID: failure.ExternalMessageID,
			Stage:             failure.Stage,
			Code:              failure.Code,
		})
	}

	return fetchSummaryResponse{
		Provider:            result.Provider,
		AccountIdentifier:   result.AccountIdentifier,
		MatchedMessageCount: result.MatchedMessageCount,
		CreatedEmailCount:   len(result.CreatedEmailIDs),
		CreatedEmailIDs:     createdEmailIDs,
		ExistingEmailCount:  len(result.ExistingEmailIDs),
		ExistingEmailIDs:    existingEmailIDs,
		FailureCount:        len(result.Failures),
		Failures:            failures,
	}
}

func buildAnalysisSummaryResponse(result manualapp.AnalyzeResult) analysisSummaryResponse {
	parsedEmailIDs := make([]uint, 0, len(result.ParsedEmailIDs))
	parsedEmailIDs = append(parsedEmailIDs, result.ParsedEmailIDs...)

	failures := make([]analysisFailureResponse, 0, len(result.Failures))
	for _, failure := range result.Failures {
		failures = append(failures, analysisFailureResponse{
			EmailID:           failure.EmailID,
			ExternalMessageID: failure.ExternalMessageID,
			Stage:             failure.Stage,
			Code:              failure.Code,
		})
	}

	return analysisSummaryResponse{
		AnalyzedEmailCount: result.AnalyzedEmailCount,
		ParsedEmailCount:   result.ParsedEmailCount,
		ParsedEmailIDs:     parsedEmailIDs,
		FailureCount:       len(result.Failures),
		Failures:           failures,
	}
}

func buildVendorResolutionSummaryResponse(result manualapp.VendorResolutionResult) vendorResolutionSummaryResponse {
	resolvedItems := make([]vendorResolutionResolvedItem, 0, len(result.ResolvedItems))
	for _, item := range result.ResolvedItems {
		resolvedItems = append(resolvedItems, vendorResolutionResolvedItem{
			ParsedEmailID:     item.ParsedEmailID,
			EmailID:           item.EmailID,
			ExternalMessageID: item.ExternalMessageID,
			VendorID:          item.VendorID,
			VendorName:        item.VendorName,
			MatchedBy:         item.MatchedBy,
		})
	}

	unresolvedExternalMessageIDs := make([]string, 0, len(result.UnresolvedExternalMessageIDs))
	unresolvedExternalMessageIDs = append(unresolvedExternalMessageIDs, result.UnresolvedExternalMessageIDs...)

	failures := make([]vendorResolutionFailureResponse, 0, len(result.Failures))
	for _, failure := range result.Failures {
		failures = append(failures, vendorResolutionFailureResponse{
			ParsedEmailID:     failure.ParsedEmailID,
			EmailID:           failure.EmailID,
			ExternalMessageID: failure.ExternalMessageID,
			Stage:             failure.Stage,
			Code:              failure.Code,
		})
	}

	return vendorResolutionSummaryResponse{
		ResolvedCount:                result.ResolvedCount,
		ResolvedItems:                resolvedItems,
		UnresolvedCount:              result.UnresolvedCount,
		UnresolvedExternalMessageIDs: unresolvedExternalMessageIDs,
		FailureCount:                 len(result.Failures),
		Failures:                     failures,
	}
}

func buildBillingEligibilitySummaryResponse(result manualapp.BillingEligibilityResult) billingEligibilitySummaryResponse {
	eligibleItems := make([]billingEligibilityEligibleItem, 0, len(result.EligibleItems))
	for _, item := range result.EligibleItems {
		eligibleItems = append(eligibleItems, billingEligibilityEligibleItem{
			ParsedEmailID:     item.ParsedEmailID,
			EmailID:           item.EmailID,
			ExternalMessageID: item.ExternalMessageID,
			VendorID:          item.VendorID,
			VendorName:        item.VendorName,
			MatchedBy:         item.MatchedBy,
			BillingNumber:     item.BillingNumber,
			InvoiceNumber:     item.InvoiceNumber,
			Amount:            item.Amount,
			Currency:          item.Currency,
			BillingDate:       item.BillingDate,
			PaymentCycle:      item.PaymentCycle,
		})
	}

	ineligibleItems := make([]billingEligibilityIneligibleItem, 0, len(result.IneligibleItems))
	for _, item := range result.IneligibleItems {
		ineligibleItems = append(ineligibleItems, billingEligibilityIneligibleItem{
			ParsedEmailID:     item.ParsedEmailID,
			EmailID:           item.EmailID,
			ExternalMessageID: item.ExternalMessageID,
			VendorID:          item.VendorID,
			VendorName:        item.VendorName,
			MatchedBy:         item.MatchedBy,
			ReasonCode:        item.ReasonCode,
		})
	}

	failures := make([]billingEligibilityFailureResponse, 0, len(result.Failures))
	for _, failure := range result.Failures {
		failures = append(failures, billingEligibilityFailureResponse{
			ParsedEmailID:     failure.ParsedEmailID,
			EmailID:           failure.EmailID,
			ExternalMessageID: failure.ExternalMessageID,
			Stage:             failure.Stage,
			Code:              failure.Code,
		})
	}

	return billingEligibilitySummaryResponse{
		EligibleCount:   result.EligibleCount,
		EligibleItems:   eligibleItems,
		IneligibleCount: result.IneligibleCount,
		IneligibleItems: ineligibleItems,
		FailureCount:    len(result.Failures),
		Failures:        failures,
	}
}

func buildBillingSummaryResponse(result manualapp.BillingResult) billingSummaryResponse {
	createdItems := make([]billingCreatedItem, 0, len(result.CreatedItems))
	for _, item := range result.CreatedItems {
		createdItems = append(createdItems, billingCreatedItem{
			BillingID:         item.BillingID,
			ParsedEmailID:     item.ParsedEmailID,
			EmailID:           item.EmailID,
			ExternalMessageID: item.ExternalMessageID,
			VendorID:          item.VendorID,
			VendorName:        item.VendorName,
			BillingNumber:     item.BillingNumber,
		})
	}

	duplicateItems := make([]billingDuplicateItem, 0, len(result.DuplicateItems))
	for _, item := range result.DuplicateItems {
		duplicateItems = append(duplicateItems, billingDuplicateItem{
			ExistingBillingID: item.ExistingBillingID,
			ParsedEmailID:     item.ParsedEmailID,
			EmailID:           item.EmailID,
			ExternalMessageID: item.ExternalMessageID,
			VendorID:          item.VendorID,
			VendorName:        item.VendorName,
			BillingNumber:     item.BillingNumber,
		})
	}

	failures := make([]billingFailureResponse, 0, len(result.Failures))
	for _, failure := range result.Failures {
		failures = append(failures, billingFailureResponse{
			ParsedEmailID:     failure.ParsedEmailID,
			EmailID:           failure.EmailID,
			ExternalMessageID: failure.ExternalMessageID,
			Stage:             failure.Stage,
			Code:              failure.Code,
		})
	}

	return billingSummaryResponse{
		CreatedCount:   result.CreatedCount,
		CreatedItems:   createdItems,
		DuplicateCount: result.DuplicateCount,
		DuplicateItems: duplicateItems,
		FailureCount:   len(result.Failures),
		Failures:       failures,
	}
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
