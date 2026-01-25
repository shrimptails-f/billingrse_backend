package seeders

import (
	"business/tools/migrations/model"

	"gorm.io/gorm"
)

// CreateKeywordGroup はキーワードグループのサンプルデータを投入する。
func CreateKeywordGroup(tx *gorm.DB) error {
	var err error
	keywordGroups := []model.KeywordGroup{
		{
			KeywordGroupID: 1,
			Name:           "Java",
			Type:           "language",
		},
		{
			KeywordGroupID: 2,
			Name:           "Python",
			Type:           "language",
		},
		{
			KeywordGroupID: 3,
			Name:           "JavaScript",
			Type:           "language",
		},
		{
			KeywordGroupID: 4,
			Name:           "TypeScript",
			Type:           "language",
		},
		{
			KeywordGroupID: 5,
			Name:           "Go",
			Type:           "language",
		},
		{
			KeywordGroupID: 6,
			Name:           "Spring Boot",
			Type:           "framework",
		},
		{
			KeywordGroupID: 7,
			Name:           "React",
			Type:           "framework",
		},
		{
			KeywordGroupID: 8,
			Name:           "Vue.js",
			Type:           "framework",
		},
		{
			KeywordGroupID: 9,
			Name:           "Django",
			Type:           "framework",
		},
		{
			KeywordGroupID: 10,
			Name:           "AWS",
			Type:           "other",
		},
		{
			KeywordGroupID: 11,
			Name:           "Docker",
			Type:           "other",
		},
		{
			KeywordGroupID: 12,
			Name:           "Kubernetes",
			Type:           "other",
		},
		{
			KeywordGroupID: 13,
			Name:           "MySQL",
			Type:           "other",
		},
		{
			KeywordGroupID: 14,
			Name:           "PostgreSQL",
			Type:           "other",
		},
		{
			KeywordGroupID: 15,
			Name:           "設計",
			Type:           "must",
		},
		{
			KeywordGroupID: 16,
			Name:           "テスト",
			Type:           "must",
		},
		{
			KeywordGroupID: 17,
			Name:           "レビュー",
			Type:           "must",
		},
	}

	for _, keywordGroup := range keywordGroups {
		err := tx.Create(&keywordGroup).Error
		if err != nil {
			return err
		}
	}

	return err
}
