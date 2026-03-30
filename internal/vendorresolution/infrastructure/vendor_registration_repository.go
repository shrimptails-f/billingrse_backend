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

type vendorAliasLookupKey struct {
	AliasType       string
	NormalizedValue string
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
		if err := r.ensureAliases(tx, vendor.UserID, vendor.ID, plan.Aliases); err != nil {
			return err
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

// ensureAliases は登録計画に含まれる alias 群を vendor 配下へ補完する。
func (r *VendorRegistrationRepository) ensureAliases(tx *gorm.DB, userID uint, vendorID uint, aliases []commondomain.VendorRegistrationAlias) error {
	records := buildVendorAliasRecords(userID, vendorID, aliases)
	if len(records) == 0 {
		return nil
	}

	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "alias_type"},
			{Name: "normalized_value"},
		},
		DoNothing: true,
	}).Create(&records).Error; err != nil {
		return fmt.Errorf("failed to create vendor aliases: %w", err)
	}

	existing, err := r.findAliasesByKeys(tx, userID, records)
	if err != nil {
		return err
	}

	existingByKey := make(map[vendorAliasLookupKey]vendorAliasRecord, len(existing))
	for _, record := range existing {
		existingByKey[vendorAliasLookupKey{
			AliasType:       record.AliasType,
			NormalizedValue: record.NormalizedValue,
		}] = record
	}

	for _, record := range records {
		existingRecord, found := existingByKey[vendorAliasLookupKey{
			AliasType:       record.AliasType,
			NormalizedValue: record.NormalizedValue,
		}]
		if !found {
			return fmt.Errorf("vendor alias upsert finished without persisted row")
		}
		if existingRecord.VendorID != vendorID {
			return fmt.Errorf("vendor alias already belongs to another vendor")
		}
	}

	return nil
}

func buildVendorAliasRecords(userID uint, vendorID uint, aliases []commondomain.VendorRegistrationAlias) []vendorAliasRecord {
	records := make([]vendorAliasRecord, 0, len(aliases))
	seen := make(map[vendorAliasLookupKey]struct{}, len(aliases))

	for _, alias := range aliases {
		if alias.AliasType == "" || alias.AliasValue == "" || alias.NormalizedValue == "" {
			continue
		}
		if utf8.RuneCountInString(alias.NormalizedValue) > vendorNameColumnLimit {
			continue
		}

		key := vendorAliasLookupKey{
			AliasType:       alias.AliasType,
			NormalizedValue: alias.NormalizedValue,
		}
		if _, found := seen[key]; found {
			continue
		}
		seen[key] = struct{}{}

		records = append(records, vendorAliasRecord{
			UserID:          userID,
			VendorID:        vendorID,
			AliasType:       alias.AliasType,
			AliasValue:      alias.AliasValue,
			NormalizedValue: alias.NormalizedValue,
		})
	}

	return records
}

func (r *VendorRegistrationRepository) findAliasesByKeys(tx *gorm.DB, userID uint, aliases []vendorAliasRecord) ([]vendorAliasRecord, error) {
	if len(aliases) == 0 {
		return nil, nil
	}

	conditions := make([]string, 0, len(aliases))
	args := make([]interface{}, 0, 1+len(aliases)*2)
	args = append(args, userID)
	for _, alias := range aliases {
		conditions = append(conditions, "(alias_type = ? AND normalized_value = ?)")
		args = append(args, alias.AliasType, alias.NormalizedValue)
	}

	var records []vendorAliasRecord
	err := tx.Where(
		"user_id = ? AND ("+strings.Join(conditions, " OR ")+")",
		args...,
	).Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("failed to lookup vendor aliases by user_id, alias_type, and normalized_value: %w", err)
	}
	return records, nil
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
