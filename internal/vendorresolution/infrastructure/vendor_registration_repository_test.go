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
	require.Equal(t, "アマゾンジャパン合同会社", vendor.Name)

	var vendors []vendorRecord
	require.NoError(t, env.db.Find(&vendors).Error)
	require.Len(t, vendors, 1)
	require.Equal(t, "アマゾンジャパン合同会社", vendors[0].Name)
	require.Equal(t, "アマゾンジャパン合同会社", vendors[0].NormalizedName)

	var aliases []vendorAliasRecord
	require.NoError(t, env.db.Find(&aliases).Error)
	require.Len(t, aliases, 1)
	require.Equal(t, vendors[0].ID, aliases[0].VendorID)
	require.Equal(t, vrdomain.MatchedByNameExact, aliases[0].AliasType)
}

// 観点:
// - normalized_name が一致する既存 vendor は再利用し、欠けている alias だけを補完すること
func TestVendorRegistrationRepository_EnsureByPlan_ReusesVendorAndAddsAlias(t *testing.T) {
	t.Parallel()

	env := newVendorRegistrationInfraTestEnv(t)
	defer env.clean()

	seedVendorRecord(t, env.db, 10, "Acme", "acme", testTime(9, 0))

	vendor, err := env.repository.EnsureByPlan(context.Background(), commondomain.VendorRegistrationPlan{
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
	require.Equal(t, "Acme", vendor.Name)

	var vendors []vendorRecord
	require.NoError(t, env.db.Find(&vendors).Error)
	require.Len(t, vendors, 1)

	var aliases []vendorAliasRecord
	require.NoError(t, env.db.Find(&aliases).Error)
	require.Len(t, aliases, 1)
	require.Equal(t, uint(10), aliases[0].VendorID)
}

// 観点:
// - 同じ登録計画で何度呼んでも vendor / alias を重複作成しないこと
func TestVendorRegistrationRepository_EnsureByPlan_IsIdempotent(t *testing.T) {
	t.Parallel()

	env := newVendorRegistrationInfraTestEnv(t)
	defer env.clean()

	plan := commondomain.VendorRegistrationPlan{
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
}

func seedVendorRecord(t *testing.T, db *gorm.DB, id uint, name, normalized string, createdAt time.Time) {
	t.Helper()

	record := vendorRecord{
		ID:             id,
		Name:           name,
		NormalizedName: normalized,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}
	require.NoError(t, db.Create(&record).Error)
}
