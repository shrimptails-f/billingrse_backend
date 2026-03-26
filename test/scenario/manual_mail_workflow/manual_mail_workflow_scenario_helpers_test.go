package test

import (
	manualpresentation "business/internal/app/presentation/manualmailworkflow"
	billingapp "business/internal/billing/application"
	billinginfra "business/internal/billing/infrastructure"
	beapp "business/internal/billingeligibility/application"
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/library/mysql"
	macinfra "business/internal/mailaccountconnection/infrastructure"
	maapp "business/internal/mailanalysis/application"
	madomain "business/internal/mailanalysis/domain"
	mainfra "business/internal/mailanalysis/infrastructure"
	mfapp "business/internal/mailfetch/application"
	mfdomain "business/internal/mailfetch/domain"
	mfinfra "business/internal/mailfetch/infrastructure"
	manualapp "business/internal/manualmailworkflow/application"
	manualinfra "business/internal/manualmailworkflow/infrastructure"
	vrapp "business/internal/vendorresolution/application"
	vrinfra "business/internal/vendorresolution/infrastructure"
	model "business/tools/migrations/models"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type manualMailWorkflowScenarioEnv struct {
	t            *testing.T
	db           *gorm.DB
	cleanup      func() error
	clock        *manualMailWorkflowScenarioClock
	log          logger.Interface
	userID       uint
	connectionID uint
	workflowRepo *manualinfra.GormWorkflowStatusRepository
	runner       manualapp.UseCase
	listUseCase  manualapp.ListUseCase
}

type manualMailWorkflowScenarioClock struct {
	now time.Time
}

type scenarioMailFetcherFactory struct {
	t                         *testing.T
	expectedConnectionID      uint
	expectedAccountIdentifier string
	fetch                     func(ctx context.Context, cond mfdomain.FetchCondition) ([]commondomain.FetchedEmailDTO, []mfdomain.MessageFailure, error)
}

type scenarioMailFetcher struct {
	fetch func(ctx context.Context, cond mfdomain.FetchCondition) ([]commondomain.FetchedEmailDTO, []mfdomain.MessageFailure, error)
}

type scenarioAnalyzerFactory struct {
	t              *testing.T
	expectedUserID uint
	analyze        func(ctx context.Context, email maapp.EmailForAnalysisTarget) (madomain.AnalysisOutput, error)
}

type scenarioAnalyzer struct {
	analyze func(ctx context.Context, email maapp.EmailForAnalysisTarget) (madomain.AnalysisOutput, error)
}

type scenarioWorkflowHistoryListResponse struct {
	Items      []scenarioWorkflowHistoryListItem `json:"items"`
	TotalCount int64                             `json:"total_count"`
}

type scenarioWorkflowHistoryListItem struct {
	WorkflowID         string               `json:"workflow_id"`
	Provider           string               `json:"provider"`
	AccountIdentifier  string               `json:"account_identifier"`
	LabelName          string               `json:"label_name"`
	Status             string               `json:"status"`
	CurrentStage       *string              `json:"current_stage"`
	FinishedAt         *time.Time           `json:"finished_at"`
	Fetch              scenarioStageSummary `json:"fetch"`
	Analysis           scenarioStageSummary `json:"analysis"`
	VendorResolution   scenarioStageSummary `json:"vendor_resolution"`
	BillingEligibility scenarioStageSummary `json:"billing_eligibility"`
	Billing            scenarioStageSummary `json:"billing"`
}

type scenarioStageSummary struct {
	SuccessCount          int                            `json:"success_count"`
	BusinessFailureCount  int                            `json:"business_failure_count"`
	TechnicalFailureCount int                            `json:"technical_failure_count"`
	Failures              []scenarioWorkflowStageFailure `json:"failures"`
}

type scenarioWorkflowStageFailure struct {
	ExternalMessageID *string   `json:"external_message_id"`
	ReasonCode        string    `json:"reason_code"`
	Message           string    `json:"message"`
	CreatedAt         time.Time `json:"created_at"`
}

func (c *manualMailWorkflowScenarioClock) Now() time.Time {
	return c.now
}

func (c *manualMailWorkflowScenarioClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- c.now.Add(d)
	return ch
}

func (f *scenarioMailFetcherFactory) Create(ctx context.Context, conn mfdomain.ConnectionRef) (mfapp.MailFetcher, error) {
	require.Equal(f.t, f.expectedConnectionID, conn.ConnectionID)
	require.Equal(f.t, "gmail", conn.Provider)
	require.Equal(f.t, f.expectedAccountIdentifier, conn.AccountIdentifier)
	return &scenarioMailFetcher{fetch: f.fetch}, nil
}

func (f *scenarioMailFetcher) Fetch(ctx context.Context, cond mfdomain.FetchCondition) ([]commondomain.FetchedEmailDTO, []mfdomain.MessageFailure, error) {
	return f.fetch(ctx, cond)
}

func (f *scenarioAnalyzerFactory) Create(ctx context.Context, spec maapp.AnalyzerSpec) (maapp.Analyzer, error) {
	require.Equal(f.t, f.expectedUserID, spec.UserID)
	return &scenarioAnalyzer{analyze: f.analyze}, nil
}

func (a *scenarioAnalyzer) Analyze(ctx context.Context, email maapp.EmailForAnalysisTarget) (madomain.AnalysisOutput, error) {
	return a.analyze(ctx, email)
}

func newManualMailWorkflowScenarioEnv(
	t *testing.T,
	fetch func(ctx context.Context, cond mfdomain.FetchCondition) ([]commondomain.FetchedEmailDTO, []mfdomain.MessageFailure, error),
	analyze func(ctx context.Context, email maapp.EmailForAnalysisTarget) (madomain.AnalysisOutput, error),
) *manualMailWorkflowScenarioEnv {
	t.Helper()

	mysqlConn, cleanup, err := mysql.CreateNewTestDB()
	if err != nil {
		skipManualMailWorkflowScenarioDBUnavailable(t, err)
	}
	require.NoError(t, err)

	t.Cleanup(func() {
		if cleanup != nil {
			require.NoError(t, cleanup())
		}
	})

	require.NoError(t, mysqlConn.DB.AutoMigrate(
		&model.User{},
		&model.EmailCredential{},
		&model.Email{},
		&model.ParsedEmail{},
		&model.Vendor{},
		&model.VendorAlias{},
		&model.Billing{},
		&model.BillingLineItem{},
		&model.ManualMailWorkflowHistory{},
		&model.ManualMailWorkflowStageFailure{},
	))

	clock := &manualMailWorkflowScenarioClock{
		now: time.Date(2026, 3, 25, 18, 0, 0, 0, time.UTC),
	}
	log := logger.NewNop()

	env := &manualMailWorkflowScenarioEnv{
		t:       t,
		db:      mysqlConn.DB,
		cleanup: cleanup,
		clock:   clock,
		log:     log,
	}

	env.userID = env.mustCreateVerifiedUser("workflow-scenario-user", "workflow-scenario-user@example.com")
	env.connectionID = env.mustCreateConnection(env.userID, "workflow-scenario-user@gmail.com")

	connectionRepo := macinfra.NewRepository(env.db, log)
	fetchUseCase := mfapp.NewUseCase(
		mfinfra.NewMailAccountConnectionReaderAdapter(connectionRepo, log),
		&scenarioMailFetcherFactory{
			t:                         t,
			expectedConnectionID:      env.connectionID,
			expectedAccountIdentifier: "workflow-scenario-user@gmail.com",
			fetch:                     fetch,
		},
		mfinfra.NewGormEmailRepositoryAdapter(env.db, clock, log),
		log,
	)
	analysisUseCase := maapp.NewUseCase(
		clock,
		&scenarioAnalyzerFactory{
			t:              t,
			expectedUserID: env.userID,
			analyze:        analyze,
		},
		mainfra.NewGormParsedEmailRepositoryAdapter(env.db, clock, log),
		log,
	)
	vendorResolutionUseCase := vrapp.NewUseCase(
		vrinfra.NewVendorResolutionRepository(env.db, log),
		vrinfra.NewVendorRegistrationRepository(env.db, log),
		log,
	)
	billingEligibilityUseCase := beapp.NewUseCase(log)
	billingUseCase := billingapp.NewUseCase(
		billinginfra.NewBillingRepository(env.db, clock, log),
		log,
	)

	env.workflowRepo = manualinfra.NewGormWorkflowStatusRepository(env.db, clock, log)
	env.runner = manualapp.NewUseCase(
		manualinfra.NewDirectManualMailFetchAdapter(fetchUseCase),
		manualinfra.NewDirectMailAnalysisAdapter(analysisUseCase),
		manualinfra.NewDirectVendorResolutionAdapter(vendorResolutionUseCase),
		manualinfra.NewDirectBillingEligibilityAdapter(billingEligibilityUseCase),
		manualinfra.NewDirectBillingAdapter(billingUseCase),
		env.workflowRepo,
		clock,
		log,
	)
	env.listUseCase = manualapp.NewListUseCase(env.workflowRepo, log)

	return env
}

func skipManualMailWorkflowScenarioDBUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}

	msg := err.Error()
	skipPatterns := []string{
		"dial tcp",
		"connect: connection refused",
		"lookup mysql",
		"access denied",
		"environment variable MYSQL_",
		"environment variable DB_HOST",
	}
	for _, pattern := range skipPatterns {
		if strings.Contains(msg, pattern) {
			t.Skipf("Skipping ManualMailWorkflow scenario test: %v", err)
		}
	}
}

func (e *manualMailWorkflowScenarioEnv) mustCreateVerifiedUser(name, email string) uint {
	e.t.Helper()

	now := e.clock.Now().UTC()
	user := model.User{
		Name:            name,
		Email:           email,
		Password:        "$2a$10$abcdefghijklmnopqrstuv",
		EmailVerified:   true,
		EmailVerifiedAt: &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	require.NoError(e.t, e.db.Create(&user).Error)
	return user.ID
}

func (e *manualMailWorkflowScenarioEnv) mustCreateConnection(userID uint, gmailAddress string) uint {
	e.t.Helper()

	now := e.clock.Now().UTC()
	record := model.EmailCredential{
		UserID:             userID,
		Type:               "gmail",
		GmailAddress:       strings.ToLower(strings.TrimSpace(gmailAddress)),
		KeyVersion:         1,
		AccessToken:        "scenario-access-token",
		AccessTokenDigest:  "scenario-access-token-digest",
		RefreshToken:       "scenario-refresh-token",
		RefreshTokenDigest: "scenario-refresh-token-digest",
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	require.NoError(e.t, e.db.Create(&record).Error)
	return record.ID
}

func (e *manualMailWorkflowScenarioEnv) mustCreateVendor(name string) uint {
	e.t.Helper()

	now := e.clock.Now().UTC()
	record := model.Vendor{
		Name:           strings.TrimSpace(name),
		NormalizedName: commondomain.NormalizeLooseText(name),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	require.NoError(e.t, e.db.Create(&record).Error)
	return record.ID
}

func (e *manualMailWorkflowScenarioEnv) mustCreateVendorAlias(vendorID uint, aliasType, aliasValue string) uint {
	e.t.Helper()

	now := e.clock.Now().UTC()
	record := model.VendorAlias{
		VendorID:        vendorID,
		AliasType:       strings.TrimSpace(aliasType),
		AliasValue:      strings.TrimSpace(aliasValue),
		NormalizedValue: commondomain.NormalizeLooseText(aliasValue),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	require.NoError(e.t, e.db.Create(&record).Error)
	return record.ID
}

func (e *manualMailWorkflowScenarioEnv) mustCreateExistingBilling(userID, vendorID uint, billingNumber string) uint {
	e.t.Helper()

	now := e.clock.Now().UTC()
	record := model.Billing{
		UserID:             userID,
		VendorID:           vendorID,
		EmailID:            999999,
		ProductNameDisplay: stringPtr("Seeded Existing Billing"),
		BillingNumber:      strings.TrimSpace(billingNumber),
		BillingSummaryDate: now,
		PaymentCycle:       "recurring",
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	require.NoError(e.t, e.db.Create(&record).Error)
	return record.ID
}

func (e *manualMailWorkflowScenarioEnv) runWorkflow(workflowID string) (manualapp.WorkflowHistoryRef, manualapp.Result) {
	e.t.Helper()

	ref, err := e.workflowRepo.CreateQueued(context.Background(), manualapp.QueuedWorkflowHistory{
		WorkflowID:   workflowID,
		UserID:       e.userID,
		ConnectionID: e.connectionID,
		LabelName:    "billing",
		SinceAt:      e.clock.Now().Add(-24 * time.Hour).UTC(),
		UntilAt:      e.clock.Now().UTC(),
		QueuedAt:     e.clock.Now().Add(-5 * time.Minute).UTC(),
	})
	require.NoError(e.t, err)

	result, err := e.runner.Execute(context.Background(), manualapp.DispatchJob{
		HistoryID:    ref.HistoryID,
		WorkflowID:   ref.WorkflowID,
		UserID:       e.userID,
		ConnectionID: e.connectionID,
		Condition: manualapp.FetchCondition{
			LabelName: "billing",
			Since:     e.clock.Now().Add(-24 * time.Hour).UTC(),
			Until:     e.clock.Now().UTC(),
		},
	})
	require.NoError(e.t, err)

	return ref, result
}

func (e *manualMailWorkflowScenarioEnv) mustFindWorkflowHistory(historyID uint64) model.ManualMailWorkflowHistory {
	e.t.Helper()

	var history model.ManualMailWorkflowHistory
	require.NoError(e.t, e.db.First(&history, historyID).Error)
	return history
}

func (e *manualMailWorkflowScenarioEnv) mustFindStageFailures(historyID uint64) []model.ManualMailWorkflowStageFailure {
	e.t.Helper()

	var records []model.ManualMailWorkflowStageFailure
	require.NoError(e.t, e.db.
		Where("workflow_history_id = ?", historyID).
		Order("stage ASC, reason_code ASC, external_message_id ASC").
		Find(&records).Error)
	return records
}

func (e *manualMailWorkflowScenarioEnv) mustFindVendorByName(name string) model.Vendor {
	e.t.Helper()

	var vendor model.Vendor
	require.NoError(e.t, e.db.Where("name = ?", strings.TrimSpace(name)).First(&vendor).Error)
	return vendor
}

func (e *manualMailWorkflowScenarioEnv) mustCountEmails() int64 {
	e.t.Helper()

	var count int64
	require.NoError(e.t, e.db.Model(&model.Email{}).Count(&count).Error)
	return count
}

func (e *manualMailWorkflowScenarioEnv) mustCountParsedEmails() int64 {
	e.t.Helper()

	var count int64
	require.NoError(e.t, e.db.Model(&model.ParsedEmail{}).Count(&count).Error)
	return count
}

func (e *manualMailWorkflowScenarioEnv) mustCountVendors() int64 {
	e.t.Helper()

	var count int64
	require.NoError(e.t, e.db.Model(&model.Vendor{}).Count(&count).Error)
	return count
}

func (e *manualMailWorkflowScenarioEnv) mustCountVendorAliases() int64 {
	e.t.Helper()

	var count int64
	require.NoError(e.t, e.db.Model(&model.VendorAlias{}).Count(&count).Error)
	return count
}

func (e *manualMailWorkflowScenarioEnv) mustCountBillings() int64 {
	e.t.Helper()

	var count int64
	require.NoError(e.t, e.db.Model(&model.Billing{}).Count(&count).Error)
	return count
}

func (e *manualMailWorkflowScenarioEnv) listWorkflowHistories(rawQuery string) scenarioWorkflowHistoryListResponse {
	e.t.Helper()

	gin.SetMode(gin.TestMode)

	controller := manualpresentation.NewController(nil, e.listUseCase, e.log)
	router := gin.New()
	router.GET("/manual-mail-workflows", func(c *gin.Context) {
		c.Set("userID", e.userID)
	}, controller.List)

	path := "/manual-mail-workflows"
	if trimmed := strings.TrimSpace(rawQuery); trimmed != "" {
		path += "?" + trimmed
	}

	req := httptest.NewRequest(http.MethodGet, path, nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(e.t, http.StatusOK, resp.Code)

	var result scenarioWorkflowHistoryListResponse
	require.NoError(e.t, json.NewDecoder(resp.Body).Decode(&result))
	return result
}

func float64Ptr(value float64) *float64 {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func timePtr(value time.Time) *time.Time {
	return &value
}
