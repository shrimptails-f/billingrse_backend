package test

import (
	model "business/tools/migrations/models"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDashboardSummary_Scenario_ReturnsAuthenticatedUsersCurrentMonthKPIs(t *testing.T) {
	env := newDashboardSummaryScenarioEnv(t)

	now := env.clock.Now().UTC()
	aprilStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	aprilEnd := time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC)
	aprilMiddle := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	marchEnd := time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)
	mayStart := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	aprilSecond := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)

	env.mustInsertParsedEmails([]model.ParsedEmail{
		{UserID: env.userID, EmailID: 101, AnalysisRunID: "run-april-start", Position: 0, ExtractedAt: aprilStart, PromptVersion: "v1", CreatedAt: now, UpdatedAt: now},
		{UserID: env.userID, EmailID: 102, AnalysisRunID: "run-april-end", Position: 0, ExtractedAt: aprilEnd, PromptVersion: "v1", CreatedAt: now, UpdatedAt: now},
		{UserID: env.userID, EmailID: 103, AnalysisRunID: "run-march-end", Position: 0, ExtractedAt: marchEnd, PromptVersion: "v1", CreatedAt: now, UpdatedAt: now},
		{UserID: env.userID, EmailID: 104, AnalysisRunID: "run-may-start", Position: 0, ExtractedAt: mayStart, PromptVersion: "v1", CreatedAt: now, UpdatedAt: now},
		{UserID: env.otherUserID, EmailID: 201, AnalysisRunID: "run-other-user", Position: 0, ExtractedAt: aprilMiddle, PromptVersion: "v1", CreatedAt: now, UpdatedAt: now},
	})

	env.mustInsertBillings([]model.Billing{
		{UserID: env.userID, VendorID: 1, EmailID: 301, BillingNumber: "billing-1", BillingDate: nil, BillingSummaryDate: aprilStart, PaymentCycle: "monthly", CreatedAt: now, UpdatedAt: now},
		{UserID: env.userID, VendorID: 2, EmailID: 302, BillingNumber: "billing-2", BillingDate: &aprilSecond, BillingSummaryDate: aprilSecond, PaymentCycle: "monthly", CreatedAt: now, UpdatedAt: now},
		{UserID: env.userID, VendorID: 3, EmailID: 303, BillingNumber: "billing-3", BillingDate: nil, BillingSummaryDate: marchEnd, PaymentCycle: "monthly", CreatedAt: now, UpdatedAt: now},
		{UserID: env.userID, VendorID: 4, EmailID: 304, BillingNumber: "billing-4", BillingDate: nil, BillingSummaryDate: mayStart, PaymentCycle: "monthly", CreatedAt: now, UpdatedAt: now},
		{UserID: env.otherUserID, VendorID: 5, EmailID: 401, BillingNumber: "billing-5", BillingDate: nil, BillingSummaryDate: aprilMiddle, PaymentCycle: "monthly", CreatedAt: now, UpdatedAt: now},
	})

	resp := env.getSummary(env.userID)
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, dashboardSummaryResponse{
		CurrentMonthAnalysisSuccessCount: 2,
		TotalSavedBillingCount:           4,
		CurrentMonthFallbackBillingCount: 1,
	}, env.mustDecodeSummaryResponse(resp))
}

func TestDashboardSummary_Scenario_RejectsMissingAuthorizationToken(t *testing.T) {
	env := newDashboardSummaryScenarioEnv(t)

	resp := env.getSummaryWithoutAuth()
	require.Equal(t, http.StatusUnauthorized, resp.Code)

	body := env.mustDecodeErrorResponse(resp)
	require.Equal(t, "missing_token", body.Error.Code)
	require.Equal(t, "認証トークンがありません。", body.Error.Message)
}
