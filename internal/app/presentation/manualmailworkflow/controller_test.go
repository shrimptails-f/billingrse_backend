package manualmailworkflow

import (
	mfdomain "business/internal/mailfetch/domain"
	manualapp "business/internal/manualmailworkflow/application"
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setUserID(c *gin.Context, uid uint) {
	c.Set("userID", uid)
}

func executeRouter(ctrl *Controller) *gin.Engine {
	r := gin.New()
	r.POST("/manual-mail-workflows", func(c *gin.Context) { setUserID(c, 1) }, ctrl.Execute)
	return r
}

func TestExecute_200(t *testing.T) {
	t.Parallel()

	uc := new(mockUseCase)
	uc.On("Execute", mock.Anything, manualapp.Command{
		UserID:       1,
		ConnectionID: 12,
		Condition: manualapp.FetchCondition{
			LabelName: "billing",
			Since:     time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
			Until:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		},
	}).Return(manualapp.Result{
		Fetch: manualapp.FetchResult{
			Provider:            "gmail",
			AccountIdentifier:   "user@example.com",
			MatchedMessageCount: 3,
			CreatedEmailIDs:     []uint{101, 102},
			ExistingEmailIDs:    []uint{201},
			Failures: []manualapp.FetchFailure{
				{ExternalMessageID: "msg-3", Stage: "save", Code: "email_save_failed"},
			},
		},
		Analysis: manualapp.AnalyzeResult{
			ParsedEmailIDs:     []uint{301, 302},
			AnalyzedEmailCount: 2,
			ParsedEmailCount:   2,
			Failures: []manualapp.AnalysisFailure{
				{EmailID: 102, ExternalMessageID: "msg-2", Stage: "analyze", Code: "analysis_failed"},
			},
		},
		VendorResolution: manualapp.VendorResolutionResult{
			ResolvedItems: []manualapp.ResolvedItem{
				{
					ParsedEmailID:     301,
					EmailID:           101,
					ExternalMessageID: "msg-1",
					VendorID:          401,
					VendorName:        "Acme",
					MatchedBy:         "name_exact",
				},
			},
			ResolvedCount:                1,
			UnresolvedCount:              1,
			UnresolvedExternalMessageIDs: []string{"msg-2"},
			Failures: []manualapp.VendorResolutionFailure{
				{ParsedEmailID: 302, EmailID: 102, ExternalMessageID: "msg-2", Stage: "resolve_vendor", Code: "vendor_resolution_failed"},
			},
		},
		BillingEligibility: manualapp.BillingEligibilityResult{
			EligibleItems: []manualapp.EligibleItem{
				{
					ParsedEmailID:     301,
					EmailID:           101,
					ExternalMessageID: "msg-1",
					VendorID:          401,
					VendorName:        "Acme",
					MatchedBy:         "name_exact",
					BillingNumber:     "digest_8e4b1c",
					InvoiceNumber:     nil,
					Amount:            1200,
					Currency:          "JPY",
					BillingDate:       nil,
					PaymentCycle:      "one_time",
				},
			},
			EligibleCount: 1,
			IneligibleItems: []manualapp.IneligibleItem{
				{
					ParsedEmailID:     302,
					EmailID:           102,
					ExternalMessageID: "msg-2",
					VendorID:          401,
					VendorName:        "Acme",
					MatchedBy:         "name_exact",
					ReasonCode:        "currency_empty",
				},
			},
			IneligibleCount: 1,
			Failures: []manualapp.BillingEligibilityFailure{
				{ParsedEmailID: 303, EmailID: 103, ExternalMessageID: "msg-3", Stage: "normalize_input", Code: "invalid_eligibility_target"},
			},
		},
		Billing: manualapp.BillingResult{
			CreatedItems: []manualapp.BillingCreatedItem{
				{
					BillingID:         501,
					ParsedEmailID:     301,
					EmailID:           101,
					ExternalMessageID: "msg-1",
					VendorID:          401,
					VendorName:        "Acme",
					BillingNumber:     "digest_8e4b1c",
				},
			},
			CreatedCount: 1,
			DuplicateItems: []manualapp.BillingDuplicateItem{
				{
					ExistingBillingID: 500,
					ParsedEmailID:     302,
					EmailID:           102,
					ExternalMessageID: "msg-2",
					VendorID:          401,
					VendorName:        "Acme",
					BillingNumber:     "digest_9a7f00",
				},
			},
			DuplicateCount: 1,
			Failures: []manualapp.BillingFailure{
				{ParsedEmailID: 303, EmailID: 103, ExternalMessageID: "msg-3", Stage: "save_billing", Code: "billing_persist_failed"},
			},
		},
	}, nil).Once()

	ctrl := newTestController(uc)
	r := executeRouter(ctrl)

	body := []byte(`{"connection_id":12,"label_name":"billing","since":"2026-03-24T00:00:00Z","until":"2026-03-25T00:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/manual-mail-workflows", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, `{
		"message": "メール取得ワークフローが完了しました。",
		"fetch": {
			"provider": "gmail",
			"account_identifier": "user@example.com",
			"matched_message_count": 3,
			"created_email_count": 2,
			"created_email_ids": [101, 102],
			"existing_email_count": 1,
			"existing_email_ids": [201],
			"failure_count": 1,
			"failures": [
				{
					"external_message_id": "msg-3",
					"stage": "save",
					"code": "email_save_failed"
				}
			]
		},
		"analysis": {
			"analyzed_email_count": 2,
			"parsed_email_count": 2,
			"parsed_email_ids": [301, 302],
			"failure_count": 1,
			"failures": [
				{
					"email_id": 102,
					"external_message_id": "msg-2",
					"stage": "analyze",
					"code": "analysis_failed"
				}
			]
		},
		"vendor_resolution": {
			"resolved_count": 1,
			"resolved_items": [
				{
					"parsed_email_id": 301,
					"email_id": 101,
					"external_message_id": "msg-1",
					"vendor_id": 401,
					"vendor_name": "Acme",
					"matched_by": "name_exact"
				}
			],
			"unresolved_count": 1,
			"unresolved_external_message_ids": ["msg-2"],
			"failure_count": 1,
			"failures": [
				{
					"parsed_email_id": 302,
					"email_id": 102,
					"external_message_id": "msg-2",
					"stage": "resolve_vendor",
					"code": "vendor_resolution_failed"
				}
			]
		},
		"billing_eligibility": {
			"eligible_count": 1,
			"eligible_items": [
				{
					"parsed_email_id": 301,
					"email_id": 101,
					"external_message_id": "msg-1",
					"vendor_id": 401,
					"vendor_name": "Acme",
					"matched_by": "name_exact",
					"billing_number": "digest_8e4b1c",
					"invoice_number": null,
					"amount": 1200,
					"currency": "JPY",
					"billing_date": null,
					"payment_cycle": "one_time"
				}
			],
			"ineligible_count": 1,
			"ineligible_items": [
				{
					"parsed_email_id": 302,
					"email_id": 102,
					"external_message_id": "msg-2",
					"vendor_id": 401,
					"vendor_name": "Acme",
					"matched_by": "name_exact",
					"reason_code": "currency_empty"
				}
			],
			"failure_count": 1,
			"failures": [
				{
					"parsed_email_id": 303,
					"email_id": 103,
					"external_message_id": "msg-3",
					"stage": "normalize_input",
					"code": "invalid_eligibility_target"
				}
			]
		},
		"billing": {
			"created_count": 1,
			"created_items": [
				{
					"billing_id": 501,
					"parsed_email_id": 301,
					"email_id": 101,
					"external_message_id": "msg-1",
					"vendor_id": 401,
					"vendor_name": "Acme",
					"billing_number": "digest_8e4b1c"
				}
			],
			"duplicate_count": 1,
			"duplicate_items": [
				{
					"existing_billing_id": 500,
					"parsed_email_id": 302,
					"email_id": 102,
					"external_message_id": "msg-2",
					"vendor_id": 401,
					"vendor_name": "Acme",
					"billing_number": "digest_9a7f00"
				}
			],
			"failure_count": 1,
			"failures": [
				{
					"parsed_email_id": 303,
					"email_id": 103,
					"external_message_id": "msg-3",
					"stage": "save_billing",
					"code": "billing_persist_failed"
				}
			]
		}
	}`, resp.Body.String())
	uc.AssertExpectations(t)
}

func TestExecute_400_InvalidRequest(t *testing.T) {
	t.Parallel()

	uc := new(mockUseCase)
	ctrl := newTestController(uc)
	r := executeRouter(ctrl)

	body := []byte(`{"connection_id":12,"label_name":"","since":"2026-03-24T00:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/manual-mail-workflows", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "invalid_request")
}

func TestExecute_401_NoUser(t *testing.T) {
	t.Parallel()

	uc := new(mockUseCase)
	ctrl := newTestController(uc)

	r := gin.New()
	r.POST("/manual-mail-workflows", ctrl.Execute)

	body := []byte(`{"connection_id":12,"label_name":"billing","since":"2026-03-24T00:00:00Z","until":"2026-03-25T00:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/manual-mail-workflows", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "unauthorized")
}

func TestExecute_404_ConnectionNotFound(t *testing.T) {
	t.Parallel()

	uc := new(mockUseCase)
	uc.On("Execute", mock.Anything, mock.Anything).Return(manualapp.Result{}, mfdomain.ErrConnectionNotFound).Once()

	ctrl := newTestController(uc)
	r := executeRouter(ctrl)

	body := []byte(`{"connection_id":12,"label_name":"billing","since":"2026-03-24T00:00:00Z","until":"2026-03-25T00:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/manual-mail-workflows", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNotFound, resp.Code)
	assert.Contains(t, resp.Body.String(), "mail_account_connection_not_found")
	uc.AssertExpectations(t)
}

func TestExecute_503_ProviderUnavailable(t *testing.T) {
	t.Parallel()

	uc := new(mockUseCase)
	uc.On("Execute", mock.Anything, mock.Anything).Return(manualapp.Result{}, errors.Join(mfdomain.ErrProviderSessionBuildFailed, errors.New("oauth failed"))).Once()

	ctrl := newTestController(uc)
	r := executeRouter(ctrl)

	body := []byte(`{"connection_id":12,"label_name":"billing","since":"2026-03-24T00:00:00Z","until":"2026-03-25T00:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/manual-mail-workflows", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)
	assert.Contains(t, resp.Body.String(), "mail_provider_unavailable")
	uc.AssertExpectations(t)
}
