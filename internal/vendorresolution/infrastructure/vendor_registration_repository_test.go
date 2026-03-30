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

type vendorRegistrationInfraTestEnv struct {
	repository *VendorRegistrationRepository
	db         *gorm.DB
	clean      func() error
}

// newVendorRegistrationInfraTestEnv は vendor registration repository の integration test 用 DB を初期化する。
func newVendorRegistrationInfraTestEnv(t *testing.T) *vendorRegistrationInfraTestEnv {
	t.Helper()

	mysqlConn, cleanup, err := mysql.CreateNewTestDB()
	if err != nil {
		skipIfVendorRegistrationDBUnavailable(t, err)
	}
	require.NoError(t, err)
	require.NoError(t, mysqlConn.DB.AutoMigrate(&vendorRecord{}, &vendorAliasRecord{}))

	return &vendorRegistrationInfraTestEnv{
		repository: NewVendorRegistrationRepository(mysqlConn.DB, logger.NewNop()),
		db:         mysqlConn.DB,
		clean:      cleanup,
	}
}

// skipIfVendorRegistrationDBUnavailable はローカル DB が使えない環境で integration test を skip する。
func skipIfVendorRegistrationDBUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "dial tcp") || strings.Contains(err.Error(), "lookup mysql") {
		t.Skipf("Skipping repository integration test: %v", err)
	}
}

// 観点:
// - 未登録の candidate vendor 名から vendors と name_exact alias を同時に作成できること
func TestVendorRegistrationRepository_EnsureByPlan_CreatesVendorAndAlias(t *testing.T) {
	t.Parallel()

	env := newVendorRegistrationInfraTestEnv(t)
	defer env.clean()

	vendor, err := env.repository.EnsureByPlan(context.Background(), commondomain.VendorRegistrationPlan{
		UserID:               1,
		VendorName:           "アマゾンジャパン合同会社",
		NormalizedVendorName: "アマゾンジャパン合同会社",
		Aliases: []commondomain.VendorRegistrationAlias{
			{
				AliasType:       vrdomain.MatchedByNameExact,
				AliasValue:      "アマゾンジャパン合同会社",
				NormalizedValue: "アマゾンジャパン合同会社",
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, vendor)
	require.NotZero(t, vendor.ID)
	require.Equal(t, uint(1), vendor.UserID)
	require.Equal(t, "アマゾンジャパン合同会社", vendor.Name)

	var vendors []vendorRecord
	require.NoError(t, env.db.Find(&vendors).Error)
	require.Len(t, vendors, 1)
	require.Equal(t, uint(1), vendors[0].UserID)
	require.Equal(t, "アマゾンジャパン合同会社", vendors[0].Name)
	require.Equal(t, "アマゾンジャパン合同会社", vendors[0].NormalizedName)

	var aliases []vendorAliasRecord
	require.NoError(t, env.db.Find(&aliases).Error)
	require.Len(t, aliases, 1)
	require.Equal(t, uint(1), aliases[0].UserID)
	require.Equal(t, vendors[0].ID, aliases[0].VendorID)
	require.Equal(t, vrdomain.MatchedByNameExact, aliases[0].AliasType)
}

// 観点:
// - normalized_name が一致する既存 vendor は再利用し、欠けている alias だけを補完すること
func TestVendorRegistrationRepository_EnsureByPlan_ReusesVendorAndAddsAlias(t *testing.T) {
	t.Parallel()

	env := newVendorRegistrationInfraTestEnv(t)
	defer env.clean()

	seedVendorRecord(t, env.db, 10, 1, "Acme", "acme", testTime(9, 0))

	vendor, err := env.repository.EnsureByPlan(context.Background(), commondomain.VendorRegistrationPlan{
		UserID:               1,
		VendorName:           "Acme",
		NormalizedVendorName: "acme",
		Aliases: []commondomain.VendorRegistrationAlias{
			{
				AliasType:       vrdomain.MatchedByNameExact,
				AliasValue:      "Acme",
				NormalizedValue: "acme",
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, vendor)
	require.Equal(t, uint(10), vendor.ID)
	require.Equal(t, uint(1), vendor.UserID)
	require.Equal(t, "Acme", vendor.Name)

	var vendors []vendorRecord
	require.NoError(t, env.db.Find(&vendors).Error)
	require.Len(t, vendors, 1)

	var aliases []vendorAliasRecord
	require.NoError(t, env.db.Find(&aliases).Error)
	require.Len(t, aliases, 1)
	require.Equal(t, uint(1), aliases[0].UserID)
	require.Equal(t, uint(10), aliases[0].VendorID)
}

// 観点:
// - 同じ登録計画で何度呼んでも vendor / alias を重複作成しないこと
func TestVendorRegistrationRepository_EnsureByPlan_IsIdempotent(t *testing.T) {
	t.Parallel()

	env := newVendorRegistrationInfraTestEnv(t)
	defer env.clean()

	plan := commondomain.VendorRegistrationPlan{
		UserID:               1,
		VendorName:           "Acme",
		NormalizedVendorName: "acme",
		Aliases: []commondomain.VendorRegistrationAlias{
			{
				AliasType:       vrdomain.MatchedByNameExact,
				AliasValue:      "Acme",
				NormalizedValue: "acme",
			},
		},
	}

	first, err := env.repository.EnsureByPlan(context.Background(), plan)
	require.NoError(t, err)
	require.NotNil(t, first)

	second, err := env.repository.EnsureByPlan(context.Background(), plan)
	require.NoError(t, err)
	require.NotNil(t, second)
	require.Equal(t, first.ID, second.ID)

	var vendors []vendorRecord
	require.NoError(t, env.db.Find(&vendors).Error)
	require.Len(t, vendors, 1)

	var aliases []vendorAliasRecord
	require.NoError(t, env.db.Find(&aliases).Error)
	require.Len(t, aliases, 1)
	require.Equal(t, uint(1), aliases[0].UserID)
}

// 観点:
// - 複数 alias を 1 回の登録計画でまとめて補完できること
func TestVendorRegistrationRepository_EnsureByPlan_CreatesMultipleAliases(t *testing.T) {
	t.Parallel()

	env := newVendorRegistrationInfraTestEnv(t)
	defer env.clean()

	vendor, err := env.repository.EnsureByPlan(context.Background(), commondomain.VendorRegistrationPlan{
		UserID:               1,
		VendorName:           "Acme",
		NormalizedVendorName: "acme",
		Aliases: []commondomain.VendorRegistrationAlias{
			{
				AliasType:       vrdomain.MatchedByNameExact,
				AliasValue:      "Acme",
				NormalizedValue: "acme",
			},
			{
				AliasType:       vrdomain.MatchedBySenderName,
				AliasValue:      "ACME Billing",
				NormalizedValue: "acme billing",
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, vendor)

	var aliases []vendorAliasRecord
	require.NoError(t, env.db.Order("alias_type ASC").Find(&aliases).Error)
	require.Len(t, aliases, 2)
	require.Equal(t, vendor.ID, aliases[0].VendorID)
	require.Equal(t, vendor.ID, aliases[1].VendorID)
	require.Equal(t, []string{vrdomain.MatchedByNameExact, vrdomain.MatchedBySenderName}, []string{aliases[0].AliasType, aliases[1].AliasType})
}

// 観点:
// - 同じ alias key が複数回含まれても 1 件として扱うこと
func TestVendorRegistrationRepository_EnsureByPlan_DeduplicatesAliasKeys(t *testing.T) {
	t.Parallel()

	env := newVendorRegistrationInfraTestEnv(t)
	defer env.clean()

	vendor, err := env.repository.EnsureByPlan(context.Background(), commondomain.VendorRegistrationPlan{
		UserID:               1,
		VendorName:           "Acme",
		NormalizedVendorName: "acme",
		Aliases: []commondomain.VendorRegistrationAlias{
			{
				AliasType:       vrdomain.MatchedByNameExact,
				AliasValue:      "Acme",
				NormalizedValue: "acme",
			},
			{
				AliasType:       vrdomain.MatchedByNameExact,
				AliasValue:      "ACME",
				NormalizedValue: "acme",
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, vendor)

	var aliases []vendorAliasRecord
	require.NoError(t, env.db.Find(&aliases).Error)
	require.Len(t, aliases, 1)
	require.Equal(t, "Acme", aliases[0].AliasValue)
}

// 観点:
// - 同じ normalized_name でも user_id が違えば別 vendor を作成すること
func TestVendorRegistrationRepository_EnsureByPlan_CreatesUserScopedVendor(t *testing.T) {
	t.Parallel()

	env := newVendorRegistrationInfraTestEnv(t)
	defer env.clean()

	seedVendorRecord(t, env.db, 10, 1, "Acme", "acme", testTime(9, 0))
	seedAliasRecord(t, env.db, 1, 10, vrdomain.MatchedByNameExact, "Acme", "acme", testTime(9, 5))

	vendor, err := env.repository.EnsureByPlan(context.Background(), commondomain.VendorRegistrationPlan{
		UserID:               2,
		VendorName:           "Acme",
		NormalizedVendorName: "acme",
		Aliases: []commondomain.VendorRegistrationAlias{
			{
				AliasType:       vrdomain.MatchedByNameExact,
				AliasValue:      "Acme",
				NormalizedValue: "acme",
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, vendor)
	require.NotEqual(t, uint(10), vendor.ID)
	require.Equal(t, uint(2), vendor.UserID)

	var vendors []vendorRecord
	require.NoError(t, env.db.Order("id ASC").Find(&vendors).Error)
	require.Len(t, vendors, 2)
	require.Equal(t, []uint{1, 2}, []uint{vendors[0].UserID, vendors[1].UserID})

	var aliases []vendorAliasRecord
	require.NoError(t, env.db.Order("id ASC").Find(&aliases).Error)
	require.Len(t, aliases, 2)
	require.Equal(t, []uint{1, 2}, []uint{aliases[0].UserID, aliases[1].UserID})
}

// 観点:
// - alias が別 vendor に属していたら transaction 全体を rollback すること
func TestVendorRegistrationRepository_EnsureByPlan_RollsBackWhenAliasBelongsToAnotherVendor(t *testing.T) {
	t.Parallel()

	env := newVendorRegistrationInfraTestEnv(t)
	defer env.clean()

	seedVendorRecord(t, env.db, 10, 1, "Existing", "existing", testTime(9, 0))
	seedAliasRecord(t, env.db, 1, 10, vrdomain.MatchedBySenderDomain, "billing@example.com", "billing@example.com", testTime(9, 5))

	vendor, err := env.repository.EnsureByPlan(context.Background(), commondomain.VendorRegistrationPlan{
		UserID:               1,
		VendorName:           "Other Vendor",
		NormalizedVendorName: "other vendor",
		Aliases: []commondomain.VendorRegistrationAlias{
			{
				AliasType:       vrdomain.MatchedBySenderDomain,
				AliasValue:      "billing@example.com",
				NormalizedValue: "billing@example.com",
			},
		},
	})
	require.ErrorContains(t, err, "vendor alias already belongs to another vendor")
	require.Nil(t, vendor)

	var vendors []vendorRecord
	require.NoError(t, env.db.Order("id ASC").Find(&vendors).Error)
	require.Len(t, vendors, 1)
	require.Equal(t, uint(10), vendors[0].ID)

	var aliases []vendorAliasRecord
	require.NoError(t, env.db.Order("id ASC").Find(&aliases).Error)
	require.Len(t, aliases, 1)
	require.Equal(t, uint(10), aliases[0].VendorID)
}

func seedVendorRecord(t *testing.T, db *gorm.DB, id uint, userID uint, name, normalized string, createdAt time.Time) {
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
}

func seedAliasRecord(t *testing.T, db *gorm.DB, userID uint, vendorID uint, aliasType, aliasValue, normalizedValue string, createdAt time.Time) {
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
}
