package infrastructure

import (
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/vendorresolution/domain"
	"context"
	"fmt"

	"gorm.io/gorm"
)

// VendorResolutionRepository は vendor 判定用の候補群を DB から収集する。
type VendorResolutionRepository struct {
	db  *gorm.DB
	log logger.Interface
}

// NewVendorResolutionRepository は Gorm ベースの vendor 判定 repository を生成する。
func NewVendorResolutionRepository(db *gorm.DB, log logger.Interface) *VendorResolutionRepository {
	if log == nil {
		log = logger.NewNop()
	}

	return &VendorResolutionRepository{
		db:  db,
		log: log.With(logger.Component("vendor_resolution_repository")),
	}
}

// FetchFacts は policy が必要とする判定材料を一括で収集する。
func (r *VendorResolutionRepository) FetchFacts(ctx context.Context, plan domain.VendorResolutionFetchPlan) (domain.VendorResolutionFacts, error) {
	if ctx == nil {
		return domain.VendorResolutionFacts{}, logger.ErrNilContext
	}
	if r.db == nil {
		return domain.VendorResolutionFacts{}, fmt.Errorf("gorm db is not configured")
	}

	nameExactCandidates, err := r.fetchExactAliasCandidates(ctx, domain.MatchedByNameExact, plan.NameExactValue)
	if err != nil {
		return domain.VendorResolutionFacts{}, err
	}
	senderDomainCandidates, err := r.fetchExactAliasCandidates(ctx, domain.MatchedBySenderDomain, plan.SenderDomainValue)
	if err != nil {
		return domain.VendorResolutionFacts{}, err
	}
	senderNameCandidates, err := r.fetchExactAliasCandidates(ctx, domain.MatchedBySenderName, plan.SenderNameValue)
	if err != nil {
		return domain.VendorResolutionFacts{}, err
	}
	subjectKeywordCandidates, err := r.fetchSubjectKeywordCandidates(ctx, plan.SubjectValue)
	if err != nil {
		return domain.VendorResolutionFacts{}, err
	}

	return domain.VendorResolutionFacts{
		NameExactCandidates:      nameExactCandidates,
		SenderDomainCandidates:   senderDomainCandidates,
		SenderNameCandidates:     senderNameCandidates,
		SubjectKeywordCandidates: subjectKeywordCandidates,
	}, nil
}

// fetchExactAliasCandidates は exact 系 alias の候補群を取得する。
func (r *VendorResolutionRepository) fetchExactAliasCandidates(ctx context.Context, aliasType, normalizedValue string) ([]commondomain.VendorAliasCandidate, error) {
	if normalizedValue == "" {
		return nil, nil
	}

	var records []resolvedAliasRecord
	err := r.baseAliasQuery(ctx).
		Where("vendor_aliases.alias_type = ? AND vendor_aliases.normalized_value = ?", aliasType, normalizedValue).
		Order("vendor_aliases.created_at DESC").
		Order("vendor_aliases.id DESC").
		Find(&records).
		Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s alias candidates: %w", aliasType, err)
	}

	return toAliasCandidates(records, aliasType), nil
}

// fetchSubjectKeywordCandidates は subject に含まれる keyword alias 候補群を取得する。
func (r *VendorResolutionRepository) fetchSubjectKeywordCandidates(ctx context.Context, normalizedSubject string) ([]commondomain.VendorAliasCandidate, error) {
	if normalizedSubject == "" {
		return nil, nil
	}

	var records []resolvedAliasRecord
	err := r.baseAliasQuery(ctx).
		Where(
			"vendor_aliases.alias_type = ? AND vendor_aliases.normalized_value <> '' AND ? LIKE CONCAT('%', vendor_aliases.normalized_value, '%')",
			domain.MatchedBySubjectKeyword,
			normalizedSubject,
		).
		Order("CHAR_LENGTH(vendor_aliases.normalized_value) DESC").
		Order("vendor_aliases.created_at DESC").
		Order("vendor_aliases.id DESC").
		Find(&records).
		Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s alias candidates: %w", domain.MatchedBySubjectKeyword, err)
	}

	return toAliasCandidates(records, domain.MatchedBySubjectKeyword), nil
}

func (r *VendorResolutionRepository) baseAliasQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("vendor_aliases").
		Select(
			"vendor_aliases.id AS alias_id, " +
				"vendor_aliases.vendor_id AS vendor_id, " +
				"vendor_aliases.alias_value AS alias_value, " +
				"vendor_aliases.normalized_value AS normalized_value, " +
				"vendor_aliases.created_at AS alias_created_at, " +
				"vendors.name AS vendor_name",
		).
		Joins("JOIN vendors ON vendors.id = vendor_aliases.vendor_id")
}

func toAliasCandidates(records []resolvedAliasRecord, aliasType string) []commondomain.VendorAliasCandidate {
	candidates := make([]commondomain.VendorAliasCandidate, 0, len(records))
	for _, record := range records {
		candidates = append(candidates, commondomain.VendorAliasCandidate{
			AliasID:         record.AliasID,
			AliasType:       aliasType,
			AliasValue:      record.AliasValue,
			NormalizedValue: record.NormalizedValue,
			AliasCreatedAt:  record.AliasCreatedAt,
			Vendor: commondomain.Vendor{
				ID:   record.VendorID,
				Name: record.VendorName,
			},
		})
	}
	return candidates
}
