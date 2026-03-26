package test

import (
	commondomain "business/internal/common/domain"
	maapp "business/internal/mailanalysis/application"
	madomain "business/internal/mailanalysis/domain"
	mfdomain "business/internal/mailfetch/domain"
	manualapp "business/internal/manualmailworkflow/application"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestManualMailWorkflow_Runner_PartialSuccessScenario(t *testing.T) {
	now := time.Date(2026, 3, 24, 9, 0, 0, 0, time.UTC)

	env := newManualMailWorkflowScenarioEnv(
		t,
		func(ctx context.Context, cond mfdomain.FetchCondition) ([]commondomain.FetchedEmailDTO, []mfdomain.MessageFailure, error) {
			require.Equal(t, "billing", cond.LabelName)
			require.True(t, cond.Since.Before(cond.Until))

			return []commondomain.FetchedEmailDTO{
				{
					ID:      "msg-create",
					Subject: "Acme Cloud invoice",
					From:    "billing@acme.example",
					To:      []string{"workflow-scenario-user@example.com"},
					Date:    now,
					Body:    "acme create body",
				},
				{
					ID:      "msg-duplicate",
					Subject: "Netflix invoice",
					From:    "billing@netflix.com",
					To:      []string{"workflow-scenario-user@example.com"},
					Date:    now.Add(time.Minute),
					Body:    "netflix duplicate body",
				},
				{
					ID:      "msg-ineligible",
					Subject: "Netflix invoice missing amount",
					From:    "billing@netflix.com",
					To:      []string{"workflow-scenario-user@example.com"},
					Date:    now.Add(2 * time.Minute),
					Body:    "netflix ineligible body",
				},
				{
					ID:      "msg-unresolved",
					Subject: "Mystery service invoice",
					From:    "billing@unknown.example",
					To:      []string{"workflow-scenario-user@example.com"},
					Date:    now.Add(3 * time.Minute),
					Body:    "mystery unresolved body",
				},
			}, nil, nil
		},
		func(ctx context.Context, email maapp.EmailForAnalysisTarget) (madomain.AnalysisOutput, error) {
			switch email.ExternalMessageID {
			case "msg-create":
				return madomain.AnalysisOutput{
					PromptVersion: "scenario_v1",
					ParsedEmails: []commondomain.ParsedEmail{
						{
							ProductNameDisplay: stringPtr("Acme Cloud Pro"),
							VendorName:         stringPtr("Acme Cloud"),
							BillingNumber:      stringPtr("ACME-2026-0001"),
							Amount:             float64Ptr(4200),
							Currency:           stringPtr("JPY"),
							PaymentCycle:       stringPtr("one_time"),
							BillingDate:        timePtr(now),
						},
					},
				}, nil
			case "msg-duplicate":
				return madomain.AnalysisOutput{
					PromptVersion: "scenario_v1",
					ParsedEmails: []commondomain.ParsedEmail{
						{
							ProductNameDisplay: stringPtr("Netflix Premium"),
							VendorName:         stringPtr("Netflix"),
							BillingNumber:      stringPtr("NF-2026-0001"),
							Amount:             float64Ptr(1290),
							Currency:           stringPtr("JPY"),
							PaymentCycle:       stringPtr("recurring"),
							BillingDate:        timePtr(now),
						},
					},
				}, nil
			case "msg-ineligible":
				return madomain.AnalysisOutput{
					PromptVersion: "scenario_v1",
					ParsedEmails: []commondomain.ParsedEmail{
						{
							ProductNameDisplay: stringPtr("Netflix Standard"),
							VendorName:         stringPtr("Netflix"),
							BillingNumber:      stringPtr("NF-2026-0002"),
							Currency:           stringPtr("JPY"),
							PaymentCycle:       stringPtr("recurring"),
							BillingDate:        timePtr(now),
						},
					},
				}, nil
			case "msg-unresolved":
				return madomain.AnalysisOutput{
					PromptVersion: "scenario_v1",
					ParsedEmails: []commondomain.ParsedEmail{
						{
							ProductNameDisplay: stringPtr("Mystery Service"),
							BillingNumber:      stringPtr("MYSTERY-2026-0001"),
							Amount:             float64Ptr(999),
							Currency:           stringPtr("JPY"),
							PaymentCycle:       stringPtr("one_time"),
							BillingDate:        timePtr(now),
						},
					},
				}, nil
			default:
				t.Fatalf("unexpected analysis target: %+v", email)
				return madomain.AnalysisOutput{}, nil
			}
		},
	)

	netflixVendorID := env.mustCreateVendor("Netflix")
	env.mustCreateVendorAlias(netflixVendorID, "name_exact", "Netflix")
	env.mustCreateExistingBilling(env.userID, netflixVendorID, "NF-2026-0001")

	ref, result := env.runWorkflow("01JQ0B7N0M7H3X9C2J5K8V6P4")

	require.Equal(t, len(result.Fetch.CreatedEmails)+len(result.Fetch.ExistingEmailIDs), 4)
	require.Equal(t, 4, result.Analysis.ParsedEmailCount)
	require.Equal(t, 3, result.VendorResolution.ResolvedCount)
	require.Equal(t, 1, result.VendorResolution.UnresolvedCount)
	require.Equal(t, 2, result.BillingEligibility.EligibleCount)
	require.Equal(t, 1, result.BillingEligibility.IneligibleCount)
	require.Equal(t, 1, result.Billing.CreatedCount)
	require.Equal(t, 1, result.Billing.DuplicateCount)

	history := env.mustFindWorkflowHistory(ref.HistoryID)
	require.Equal(t, manualapp.WorkflowStatusPartialSuccess, history.Status)
	require.Nil(t, history.CurrentStage)
	require.NotNil(t, history.FinishedAt)
	require.Equal(t, 4, history.FetchSuccessCount)
	require.Equal(t, 0, history.FetchBusinessFailureCount)
	require.Equal(t, 0, history.FetchTechnicalFailureCount)
	require.Equal(t, 4, history.AnalysisSuccessCount)
	require.Equal(t, 0, history.AnalysisBusinessFailureCount)
	require.Equal(t, 0, history.AnalysisTechnicalFailureCount)
	require.Equal(t, 3, history.VendorResolutionSuccessCount)
	require.Equal(t, 1, history.VendorResolutionBusinessFailureCount)
	require.Equal(t, 0, history.VendorResolutionTechnicalFailureCount)
	require.Equal(t, 2, history.BillingEligibilitySuccessCount)
	require.Equal(t, 1, history.BillingEligibilityBusinessFailureCount)
	require.Equal(t, 0, history.BillingEligibilityTechnicalFailureCount)
	require.Equal(t, 1, history.BillingSuccessCount)
	require.Equal(t, 1, history.BillingBusinessFailureCount)
	require.Equal(t, 0, history.BillingTechnicalFailureCount)

	failures := env.mustFindStageFailures(ref.HistoryID)
	require.Len(t, failures, 3)

	byStageAndReason := make(map[string]string, len(failures))
	messageByStageAndReason := make(map[string]string, len(failures))
	for _, failure := range failures {
		require.NotNil(t, failure.ExternalMessageID)
		key := failure.Stage + ":" + failure.ReasonCode
		byStageAndReason[key] = *failure.ExternalMessageID
		messageByStageAndReason[key] = failure.Message
	}

	require.Equal(t, "msg-unresolved", byStageAndReason["vendorresolution:vendor_unresolved"])
	require.Equal(t, "msg-ineligible", byStageAndReason["billingeligibility:amount_empty"])
	require.Equal(t, "msg-duplicate", byStageAndReason["billing:duplicate_billing"])
	require.Contains(t, messageByStageAndReason["vendorresolution:vendor_unresolved"], "msg-unresolved")
	require.Contains(t, messageByStageAndReason["billingeligibility:amount_empty"], "Netflix")
	require.Contains(t, messageByStageAndReason["billingeligibility:amount_empty"], "msg-ineligible")
	require.Contains(t, messageByStageAndReason["billing:duplicate_billing"], "Netflix")
	require.Contains(t, messageByStageAndReason["billing:duplicate_billing"], "NF-2026-0001")
	require.Contains(t, messageByStageAndReason["billing:duplicate_billing"], "msg-duplicate")

	listResponse := env.listWorkflowHistories("")
	require.EqualValues(t, 1, listResponse.TotalCount)
	require.Len(t, listResponse.Items, 1)
	listItem := listResponse.Items[0]
	require.Equal(t, ref.WorkflowID, listItem.WorkflowID)
	require.Equal(t, "gmail", listItem.Provider)
	require.Equal(t, "workflow-scenario-user@gmail.com", listItem.AccountIdentifier)
	require.Equal(t, "billing", listItem.LabelName)
	require.Equal(t, manualapp.WorkflowStatusPartialSuccess, listItem.Status)
	require.Nil(t, listItem.CurrentStage)
	require.NotNil(t, listItem.FinishedAt)
	require.Equal(t, 4, listItem.Fetch.SuccessCount)
	require.Equal(t, 0, listItem.Fetch.BusinessFailureCount)
	require.Equal(t, 0, listItem.Fetch.TechnicalFailureCount)
	require.Empty(t, listItem.Fetch.Failures)
	require.Equal(t, 4, listItem.Analysis.SuccessCount)
	require.Empty(t, listItem.Analysis.Failures)
	require.Equal(t, 3, listItem.VendorResolution.SuccessCount)
	require.Equal(t, 1, listItem.VendorResolution.BusinessFailureCount)
	require.Len(t, listItem.VendorResolution.Failures, 1)
	require.Equal(t, "vendor_unresolved", listItem.VendorResolution.Failures[0].ReasonCode)
	require.NotNil(t, listItem.VendorResolution.Failures[0].ExternalMessageID)
	require.Equal(t, "msg-unresolved", *listItem.VendorResolution.Failures[0].ExternalMessageID)
	require.Equal(t, 2, listItem.BillingEligibility.SuccessCount)
	require.Equal(t, 1, listItem.BillingEligibility.BusinessFailureCount)
	require.Len(t, listItem.BillingEligibility.Failures, 1)
	require.Equal(t, "amount_empty", listItem.BillingEligibility.Failures[0].ReasonCode)
	require.NotNil(t, listItem.BillingEligibility.Failures[0].ExternalMessageID)
	require.Equal(t, "msg-ineligible", *listItem.BillingEligibility.Failures[0].ExternalMessageID)
	require.Equal(t, 1, listItem.Billing.SuccessCount)
	require.Equal(t, 1, listItem.Billing.BusinessFailureCount)
	require.Len(t, listItem.Billing.Failures, 1)
	require.Equal(t, "duplicate_billing", listItem.Billing.Failures[0].ReasonCode)
	require.NotNil(t, listItem.Billing.Failures[0].ExternalMessageID)
	require.Equal(t, "msg-duplicate", *listItem.Billing.Failures[0].ExternalMessageID)

	filteredListResponse := env.listWorkflowHistories("status=partial_success")
	require.EqualValues(t, 1, filteredListResponse.TotalCount)
	require.Len(t, filteredListResponse.Items, 1)
	require.Equal(t, ref.WorkflowID, filteredListResponse.Items[0].WorkflowID)

	require.Equal(t, int64(4), env.mustCountEmails())
	require.Equal(t, int64(4), env.mustCountParsedEmails())
	require.Equal(t, int64(2), env.mustCountVendors())
	require.Equal(t, int64(2), env.mustCountVendorAliases())
	require.Equal(t, int64(2), env.mustCountBillings())

	acmeVendor := env.mustFindVendorByName("Acme Cloud")
	require.Equal(t, "acme cloud", acmeVendor.NormalizedName)
}

func TestManualMailWorkflow_Runner_SucceededScenario(t *testing.T) {
	now := time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC)

	env := newManualMailWorkflowScenarioEnv(
		t,
		func(ctx context.Context, cond mfdomain.FetchCondition) ([]commondomain.FetchedEmailDTO, []mfdomain.MessageFailure, error) {
			require.Equal(t, "billing", cond.LabelName)

			return []commondomain.FetchedEmailDTO{
				{
					ID:      "msg-success",
					Subject: "Acme Cloud invoice",
					From:    "billing@acme.example",
					To:      []string{"workflow-scenario-user@example.com"},
					Date:    now,
					Body:    "acme success body",
				},
			}, nil, nil
		},
		func(ctx context.Context, email maapp.EmailForAnalysisTarget) (madomain.AnalysisOutput, error) {
			require.Equal(t, "msg-success", email.ExternalMessageID)

			return madomain.AnalysisOutput{
				PromptVersion: "scenario_v1",
				ParsedEmails: []commondomain.ParsedEmail{
					{
						ProductNameDisplay: stringPtr("Acme Cloud Enterprise"),
						VendorName:         stringPtr("Acme Cloud"),
						BillingNumber:      stringPtr("ACME-2026-1000"),
						Amount:             float64Ptr(5000),
						Currency:           stringPtr("JPY"),
						PaymentCycle:       stringPtr("one_time"),
						BillingDate:        timePtr(now),
					},
				},
			}, nil
		},
	)

	ref, result := env.runWorkflow("01JQ0B7N0M7H3X9C2J5K8V6P5")

	require.Equal(t, len(result.Fetch.CreatedEmails)+len(result.Fetch.ExistingEmailIDs), 1)
	require.Equal(t, 1, result.Analysis.ParsedEmailCount)
	require.Equal(t, 1, result.VendorResolution.ResolvedCount)
	require.Equal(t, 0, result.VendorResolution.UnresolvedCount)
	require.Equal(t, 1, result.BillingEligibility.EligibleCount)
	require.Equal(t, 0, result.BillingEligibility.IneligibleCount)
	require.Equal(t, 1, result.Billing.CreatedCount)
	require.Equal(t, 0, result.Billing.DuplicateCount)

	history := env.mustFindWorkflowHistory(ref.HistoryID)
	require.Equal(t, manualapp.WorkflowStatusSucceeded, history.Status)
	require.Nil(t, history.CurrentStage)
	require.NotNil(t, history.FinishedAt)
	require.Equal(t, 1, history.FetchSuccessCount)
	require.Equal(t, 0, history.FetchBusinessFailureCount)
	require.Equal(t, 0, history.FetchTechnicalFailureCount)
	require.Equal(t, 1, history.AnalysisSuccessCount)
	require.Equal(t, 0, history.AnalysisBusinessFailureCount)
	require.Equal(t, 0, history.AnalysisTechnicalFailureCount)
	require.Equal(t, 1, history.VendorResolutionSuccessCount)
	require.Equal(t, 0, history.VendorResolutionBusinessFailureCount)
	require.Equal(t, 0, history.VendorResolutionTechnicalFailureCount)
	require.Equal(t, 1, history.BillingEligibilitySuccessCount)
	require.Equal(t, 0, history.BillingEligibilityBusinessFailureCount)
	require.Equal(t, 0, history.BillingEligibilityTechnicalFailureCount)
	require.Equal(t, 1, history.BillingSuccessCount)
	require.Equal(t, 0, history.BillingBusinessFailureCount)
	require.Equal(t, 0, history.BillingTechnicalFailureCount)

	listResponse := env.listWorkflowHistories("status=succeeded")
	require.EqualValues(t, 1, listResponse.TotalCount)
	require.Len(t, listResponse.Items, 1)
	listItem := listResponse.Items[0]
	require.Equal(t, ref.WorkflowID, listItem.WorkflowID)
	require.Equal(t, "gmail", listItem.Provider)
	require.Equal(t, "workflow-scenario-user@gmail.com", listItem.AccountIdentifier)
	require.Equal(t, "billing", listItem.LabelName)
	require.Equal(t, manualapp.WorkflowStatusSucceeded, listItem.Status)
	require.Nil(t, listItem.CurrentStage)
	require.NotNil(t, listItem.FinishedAt)
	require.Equal(t, 1, listItem.Fetch.SuccessCount)
	require.Equal(t, 1, listItem.Analysis.SuccessCount)
	require.Equal(t, 1, listItem.VendorResolution.SuccessCount)
	require.Equal(t, 1, listItem.BillingEligibility.SuccessCount)
	require.Equal(t, 1, listItem.Billing.SuccessCount)
	require.Empty(t, listItem.Fetch.Failures)
	require.Empty(t, listItem.Analysis.Failures)
	require.Empty(t, listItem.VendorResolution.Failures)
	require.Empty(t, listItem.BillingEligibility.Failures)
	require.Empty(t, listItem.Billing.Failures)

	failures := env.mustFindStageFailures(ref.HistoryID)
	require.Empty(t, failures)

	require.Equal(t, int64(1), env.mustCountEmails())
	require.Equal(t, int64(1), env.mustCountParsedEmails())
	require.Equal(t, int64(1), env.mustCountVendors())
	require.Equal(t, int64(1), env.mustCountVendorAliases())
	require.Equal(t, int64(1), env.mustCountBillings())
}
