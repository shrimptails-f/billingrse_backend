package seeders

import (
	"business/tools/migrations/model"
	"time"

	"gorm.io/gorm"
)

// CreateEmailKeywordGroup はメールとキーワードグループの関連のサンプルデータを投入する。
func CreateEmailKeywordGroup(tx *gorm.DB) error {
	var err error

	emailKeywordGroups := []model.EmailKeywordGroup{
		// email001 (Java案件) の関連
		{
			UserID:         1,
			EmailID:        1,
			KeywordGroupID: 1, // Java
			CreatedAt:      time.Now(),
		},
		{
			UserID:         1,
			EmailID:        1,
			KeywordGroupID: 6, // Spring Boot
			CreatedAt:      time.Now(),
		},
		{
			UserID:         1,
			EmailID:        1,
			KeywordGroupID: 13, // MySQL
			CreatedAt:      time.Now(),
		},
		// email002 (React案件) の関連
		{
			UserID:         1,
			EmailID:        2,
			KeywordGroupID: 7, // React
			CreatedAt:      time.Now(),
		},
		{
			UserID:         1,
			EmailID:        2,
			KeywordGroupID: 4, // TypeScript
			CreatedAt:      time.Now(),
		},
		{
			UserID:         1,
			EmailID:        2,
			KeywordGroupID: 3, // JavaScript
			CreatedAt:      time.Now(),
		},
		// email003 (Python機械学習案件) の関連
		{
			UserID:         1,
			EmailID:        3,
			KeywordGroupID: 2, // Python
			CreatedAt:      time.Now(),
		},
		{
			UserID:         1,
			EmailID:        3,
			KeywordGroupID: 9, // Django
			CreatedAt:      time.Now(),
		},
		{
			UserID:         1,
			EmailID:        3,
			KeywordGroupID: 10, // AWS
			CreatedAt:      time.Now(),
		},
		// email004 (Go案件) の関連
		{
			UserID:         1,
			EmailID:        4,
			KeywordGroupID: 5, // Go
			CreatedAt:      time.Now(),
		},
		{
			UserID:         1,
			EmailID:        4,
			KeywordGroupID: 11, // Docker
			CreatedAt:      time.Now(),
		},
		{
			UserID:         1,
			EmailID:        4,
			KeywordGroupID: 12, // Kubernetes
			CreatedAt:      time.Now(),
		},
		// email005 (フルスタック案件) の関連
		{
			UserID:         1,
			EmailID:        5,
			KeywordGroupID: 7, // React
			CreatedAt:      time.Now(),
		},
		{
			UserID:         1,
			EmailID:        5,
			KeywordGroupID: 3, // JavaScript
			CreatedAt:      time.Now(),
		},
		{
			UserID:         1,
			EmailID:        5,
			KeywordGroupID: 10, // AWS
			CreatedAt:      time.Now(),
		},
	}

	for _, emailKeywordGroup := range emailKeywordGroups {
		err := tx.Create(&emailKeywordGroup).Error
		if err != nil {
			return err
		}
	}

	return err
}
