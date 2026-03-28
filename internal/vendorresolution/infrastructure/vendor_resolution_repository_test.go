package infrastructure

import (
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/library/mysql"
	vrdomain "business/internal/vendorresolution/domain"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type vendorResolutionInfraTestEnv struct {
	repository *VendorResolutionRepository
	db         *gorm.DB
	clean      func() error
}

// newVendorResolutionInfraTestEnv は vendor resolution repository の integration test 用 DB を初期化する。
func newVendorResolutionInfraTestEnv(t *testing.T) *vendorResolutionInfraTestEnv {
	t.Helper()

	mysqlConn, cleanup, err := mysql.CreateNewTestDB()
	if err != nil {
		skipIfVendorResolutionDBUnavailable(t, err)
	}
	require.NoError(t, err)
	require.NoError(t, mysqlConn.DB.AutoMigrate(&vendorRecord{}, &vendorAliasRecord{}))

	return &vendorResolutionInfraTestEnv{
		repository: NewVendorResolutionRepository(mysqlConn.DB, logger.NewNop()),
		db:         mysqlConn.DB,
		clean:      cleanup,
	}
}

// skipIfVendorResolutionDBUnavailable はローカル DB が使えない環境で integration test を skip する。
func skipIfVendorResolutionDBUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "dial tcp") || strings.Contains(err.Error(), "lookup mysql") {
		t.Skipf("Skipping repository integration test: %v", err)
	}
}

// 観点:
// - fetch は各ルール用の候補群を 1 回で集められること
// - user_id が違う alias は混ざらないこと
func TestVendorResolutionRepository_FetchFacts_CollectsCandidatesForEachRule(t *testing.T) {
	t.Parallel()

	env := newVendorResolutionInfraTestEnv(t)
	defer env.clean()

	acmeVendor := seedVendor(t, env.db, 1, 1, "Acme", "acme", testTime(9, 0))
	domainVendor := seedVendor(t, env.db, 3, 1, "Domain Vendor", "domain vendor", testTime(9, 10))
	senderVendor := seedVendor(t, env.db, 4, 1, "Sender Vendor", "sender vendor", testTime(9, 15))
	subjectVendor := seedVendor(t, env.db, 5, 1, "Subject Vendor", "subject vendor", testTime(9, 20))
	otherUserVendor := seedVendor(t, env.db, 6, 2, "Other User Acme", "other user acme", testTime(9, 25))

	seedAlias(t, env.db, acmeVendor.UserID, acmeVendor.ID, vrdomain.MatchedByNameExact, "Acme", "acme", testTime(9, 30))
	seedAlias(t, env.db, domainVendor.UserID, domainVendor.ID, vrdomain.MatchedBySenderDomain, "acme.example.com", "acme.example.com", testTime(9, 50))
	seedAlias(t, env.db, senderVendor.UserID, senderVendor.ID, vrdomain.MatchedBySenderName, "Acme Billing", "acme billing", testTime(10, 0))
	seedAlias(t, env.db, subjectVendor.UserID, subjectVendor.ID, vrdomain.MatchedBySubjectKeyword, "acme cloud invoice", "acme cloud invoice", testTime(10, 10))
	seedAlias(t, env.db, otherUserVendor.UserID, otherUserVendor.ID, vrdomain.MatchedByNameExact, "Acme", "acme", testTime(10, 20))

	facts, err := env.repository.FetchFacts(context.Background(), vrdomain.VendorResolutionFetchPlan{
		UserID:            1,
		NameExactValue:    "acme",
		SenderDomainValue: "acme.example.com",
		SenderNameValue:   "acme billing",
		SubjectValue:      "your acme cloud invoice is ready",
	})
	require.NoError(t, err)
	require.Len(t, facts.NameExactCandidates, 1)
	require.Len(t, facts.SenderDomainCandidates, 1)
	require.Len(t, facts.SenderNameCandidates, 1)
	require.Len(t, facts.SubjectKeywordCandidates, 1)

	require.Equal(t, []uint{acmeVendor.ID}, vendorIDs(facts.NameExactCandidates))
	require.Equal(t, domainVendor.ID, facts.SenderDomainCandidates[0].Vendor.ID)
	require.Equal(t, senderVendor.ID, facts.SenderNameCandidates[0].Vendor.ID)
	require.Equal(t, subjectVendor.ID, facts.SubjectKeywordCandidates[0].Vendor.ID)
}

// 観点:
// - subject keyword は subject に含まれる alias だけを返すこと
func TestVendorResolutionRepository_FetchFacts_FiltersSubjectKeywordMatches(t *testing.T) {
	t.Parallel()

	env := newVendorResolutionInfraTestEnv(t)
	defer env.clean()

	vendor1 := seedVendor(t, env.db, 1, 1, "Invoice", "invoice", testTime(9, 0))
	vendor2 := seedVendor(t, env.db, 2, 1, "Shipping", "shipping", testTime(9, 5))

	seedAlias(t, env.db, vendor1.UserID, vendor1.ID, vrdomain.MatchedBySubjectKeyword, "invoice ready", "invoice ready", testTime(9, 10))
	seedAlias(t, env.db, vendor2.UserID, vendor2.ID, vrdomain.MatchedBySubjectKeyword, "shipping notice", "shipping notice", testTime(9, 20))

	facts, err := env.repository.FetchFacts(context.Background(), vrdomain.VendorResolutionFetchPlan{
		UserID:       1,
		SubjectValue: "invoice ready for download",
	})
	require.NoError(t, err)
	require.Len(t, facts.SubjectKeywordCandidates, 1)
	require.Equal(t, vendor1.ID, facts.SubjectKeywordCandidates[0].Vendor.ID)
}

func seedVendor(t *testing.T, db *gorm.DB, id uint, userID uint, name, normalized string, createdAt time.Time) commondomain.Vendor {
	t.Helper()

	record := vendorRecord{
		ID:             id,
		UserID:         userID,
		Name:           name,
		NormalizedName: normalized,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}
	require.NoError(t, db.Create(&record).Error)

	return commondomain.Vendor{
		ID:     record.ID,
		UserID: record.UserID,
		Name:   record.Name,
	}
}

func seedAlias(t *testing.T, db *gorm.DB, userID uint, vendorID uint, aliasType, aliasValue, normalizedValue string, createdAt time.Time) vendorAliasRecord {
	t.Helper()

	record := vendorAliasRecord{
		UserID:          userID,
		VendorID:        vendorID,
		AliasType:       aliasType,
		AliasValue:      aliasValue,
		NormalizedValue: normalizedValue,
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}
	require.NoError(t, db.Create(&record).Error)
	return record
}

func vendorIDs(candidates []commondomain.VendorAliasCandidate) []uint {
	ids := make([]uint, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.Vendor.ID)
	}
	return ids
}

func testTime(hour, minute int) time.Time {
	return time.Date(2026, 3, 24, hour, minute, 0, 0, time.UTC)
}
