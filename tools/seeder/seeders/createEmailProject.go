package seeders

import (
	"business/tools/migrations/model"

	"gorm.io/gorm"
)

// CreateEmailProject は案件メール専用情報のサンプルデータを投入する。
func CreateEmailProject(tx *gorm.DB) error {
	var err error

	// ポインタ型のヘルパー関数
	stringPtr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }

	emailProjects := []model.EmailProject{
		{
			UserID:          1,
			EmailID:         1,
			ProjectTitle:    stringPtr("ECサイト構築プロジェクト"),
			EntryTiming:     stringPtr("2024年2月〜"),
			EndTiming:       stringPtr("2024年8月"),
			WorkLocation:    stringPtr("東京都渋谷区（リモート可）"),
			PriceFrom:       intPtr(600000),
			PriceTo:         intPtr(800000),
			RemoteType:      stringPtr("フルリモート可"),
			RemoteFrequency: stringPtr("週5日"),
		},
		{
			UserID:          1,
			EmailID:         3,
			ProjectTitle:    stringPtr("AIチャットボット開発"),
			EntryTiming:     stringPtr("2024年3月〜"),
			EndTiming:       stringPtr("2024年12月"),
			WorkLocation:    stringPtr("東京都港区"),
			PriceFrom:       intPtr(800000),
			PriceTo:         intPtr(1200000),
			RemoteType:      stringPtr("出社必須"),
			RemoteFrequency: stringPtr("週5日出社"),
		},
		{
			UserID:          1,
			EmailID:         4,
			ProjectTitle:    stringPtr("マイクロサービス基盤構築"),
			EntryTiming:     stringPtr("2024年4月〜"),
			EndTiming:       stringPtr("2024年10月"),
			WorkLocation:    stringPtr("東京都千代田区"),
			PriceFrom:       intPtr(750000),
			PriceTo:         intPtr(1000000),
			RemoteType:      stringPtr("フルリモート可"),
			RemoteFrequency: stringPtr("完全リモート"),
		},
	}

	for _, emailProject := range emailProjects {
		err := tx.Create(&emailProject).Error
		if err != nil {
			return err
		}
	}

	return err
}
