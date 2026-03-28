package infrastructure

import (
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const vendorNameColumnLimit = 255

// VendorRegistrationRepository は canonical Vendor と alias の補完保存を担当する。
type VendorRegistrationRepository struct {
	db  *gorm.DB
	log logger.Interface
}

// NewVendorRegistrationRepository は Gorm ベースの vendor 登録 repository を生成する。
func NewVendorRegistrationRepository(db *gorm.DB, log logger.Interface) *VendorRegistrationRepository {
	if log == nil {
		log = logger.NewNop()
	}

	return &VendorRegistrationRepository{
		db:  db,
		log: log.With(logger.Component("vendor_registration_repository")),
	}
}

// EnsureByPlan は policy が作った登録計画どおりに vendor / alias を補完する。
func (r *VendorRegistrationRepository) EnsureByPlan(ctx context.Context, plan commondomain.VendorRegistrationPlan) (*commondomain.Vendor, error) {
	if ctx == nil {
		return nil, logger.ErrNilContext
	}
	if r.db == nil {
		return nil, fmt.Errorf("gorm db is not configured")
	}

	plan = normalizeRegistrationPlan(plan)
	if plan.UserID == 0 {
		return nil, fmt.Errorf("user_id is required")
	}
	if plan.VendorName == "" || plan.NormalizedVendorName == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(plan.VendorName) > vendorNameColumnLimit || utf8.RuneCountInString(plan.NormalizedVendorName) > vendorNameColumnLimit {
		// スキーマ制約に収まらない候補は切り詰めず、未解決のまま人手確認へ回す。
		return nil, nil
	}

	var vendor vendorRecord
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record, err := r.findVendorByNormalizedName(tx, plan.UserID, plan.NormalizedVendorName)
		if err != nil {
			return err
		}
		if record == nil {
			record, err = r.createVendor(tx, plan.UserID, plan.VendorName, plan.NormalizedVendorName)
			if err != nil {
				return err
			}
		}

		vendor = *record
		for _, alias := range plan.Aliases {
			if err := r.ensureAlias(tx, vendor.UserID, vendor.ID, alias); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &commondomain.Vendor{
		ID:     vendor.ID,
		UserID: vendor.UserID,
		Name:   vendor.Name,
	}, nil
}

// findVendorByNormalizedName は user_id + normalized_name 一意制約を使って既存 vendor を探す。
func (r *VendorRegistrationRepository) findVendorByNormalizedName(tx *gorm.DB, userID uint, normalizedName string) (*vendorRecord, error) {
	var record vendorRecord
	err := tx.Where("user_id = ? AND normalized_name = ?", userID, normalizedName).Take(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to lookup vendor by user_id and normalized_name: %w", err)
	}
	return &record, nil
}

// createVendor は競合時の再読込も含めて canonical Vendor を補完する。
func (r *VendorRegistrationRepository) createVendor(tx *gorm.DB, userID uint, canonicalName, normalizedName string) (*vendorRecord, error) {
	record := vendorRecord{
		UserID:         userID,
		Name:           canonicalName,
		NormalizedName: normalizedName,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "normalized_name"}},
		DoNothing: true,
	}).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("failed to create vendor: %w", err)
	}
	if record.ID != 0 {
		return &record, nil
	}

	created, err := r.findVendorByNormalizedName(tx, userID, normalizedName)
	if err != nil {
		return nil, err
	}
	if created == nil {
		return nil, fmt.Errorf("vendor upsert finished without persisted row")
	}
	return created, nil
}

// ensureAlias は登録計画に含まれる alias を vendor 配下へ補完する。
func (r *VendorRegistrationRepository) ensureAlias(tx *gorm.DB, userID uint, vendorID uint, alias commondomain.VendorRegistrationAlias) error {
	if alias.AliasType == "" || alias.AliasValue == "" || alias.NormalizedValue == "" {
		return nil
	}
	if utf8.RuneCountInString(alias.NormalizedValue) > vendorNameColumnLimit {
		return nil
	}

	record := vendorAliasRecord{
		UserID:          userID,
		VendorID:        vendorID,
		AliasType:       alias.AliasType,
		AliasValue:      alias.AliasValue,
		NormalizedValue: alias.NormalizedValue,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "alias_type"},
			{Name: "normalized_value"},
		},
		DoNothing: true,
	}).Create(&record).Error; err != nil {
		return fmt.Errorf("failed to create vendor alias: %w", err)
	}
	if record.ID != 0 {
		return nil
	}

	existing, err := r.findAliasByNormalizedValue(tx, userID, alias.AliasType, alias.NormalizedValue)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("vendor alias upsert finished without persisted row")
	}
	if existing.VendorID != vendorID {
		return fmt.Errorf("vendor alias already belongs to another vendor")
	}
	return nil
}

func (r *VendorRegistrationRepository) findAliasByNormalizedValue(tx *gorm.DB, userID uint, aliasType, normalizedValue string) (*vendorAliasRecord, error) {
	var record vendorAliasRecord
	err := tx.Where("user_id = ? AND alias_type = ? AND normalized_value = ?", userID, aliasType, normalizedValue).Take(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to lookup vendor alias by user_id, alias_type, and normalized_value: %w", err)
	}
	return &record, nil
}

func normalizeRegistrationPlan(plan commondomain.VendorRegistrationPlan) commondomain.VendorRegistrationPlan {
	plan.VendorName = strings.TrimSpace(plan.VendorName)
	plan.NormalizedVendorName = strings.TrimSpace(plan.NormalizedVendorName)

	normalizedAliases := make([]commondomain.VendorRegistrationAlias, 0, len(plan.Aliases))
	for _, alias := range plan.Aliases {
		alias.AliasType = strings.TrimSpace(alias.AliasType)
		alias.AliasValue = strings.TrimSpace(alias.AliasValue)
		alias.NormalizedValue = strings.TrimSpace(alias.NormalizedValue)
		if alias.AliasType == "" || alias.AliasValue == "" || alias.NormalizedValue == "" {
			continue
		}
		normalizedAliases = append(normalizedAliases, alias)
	}
	plan.Aliases = normalizedAliases

	return plan
}
