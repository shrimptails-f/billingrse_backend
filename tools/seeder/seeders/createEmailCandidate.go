package seeders

import (
	"business/tools/migrations/model"

	"gorm.io/gorm"
)

// CreateEmailCandidate は人材提案メール専用情報のサンプルデータを投入する。
func CreateEmailCandidate(tx *gorm.DB) error {
	var err error

	// ポインタ型のヘルパー関数
	stringPtr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }

	emailCandidates := []model.EmailCandidate{
		{
			UserID:           1,
			EmailID:          2,
			CandidateName:    stringPtr("山田太郎"),
			ExperienceYears:  intPtr(5),
			SkillsSummary:    stringPtr("React、TypeScriptでの開発経験が豊富。フロントエンド開発を中心に、UI/UX設計から実装まで幅広く対応可能。"),
			AvailabilityDate: stringPtr("即日〜"),
		},
		{
			UserID:           1,
			EmailID:          5,
			CandidateName:    stringPtr("佐藤花子"),
			ExperienceYears:  intPtr(7),
			SkillsSummary:    stringPtr("React、Node.js、AWSでの開発経験が豊富なフルスタックエンジニア。設計から運用まで一貫して対応可能。"),
			AvailabilityDate: stringPtr("2024年2月〜"),
		},
	}

	for _, emailCandidate := range emailCandidates {
		err := tx.Create(&emailCandidate).Error
		if err != nil {
			return err
		}
	}

	return err
}
