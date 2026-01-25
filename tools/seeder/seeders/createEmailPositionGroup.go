package seeders

import (
	"business/tools/migrations/model"

	"gorm.io/gorm"
)

// CreateEmailPositionGroup はメールとポジショングループの関連のサンプルデータを投入する。
func CreateEmailPositionGroup(tx *gorm.DB) error {
	var err error

	emailPositionGroups := []model.EmailPositionGroup{
		// email001 (Java案件) の関連
		{
			UserID:          1,
			EmailID:         1,
			PositionGroupID: 3, // SE
		},
		{
			UserID:          1,
			EmailID:         1,
			PositionGroupID: 2, // PL
		},
		// email002 (React案件) の関連
		{
			UserID:          1,
			EmailID:         2,
			PositionGroupID: 6, // フロントエンドエンジニア
		},
		{
			UserID:          1,
			EmailID:         2,
			PositionGroupID: 3, // SE
		},
		// email003 (Python機械学習案件) の関連
		{
			UserID:          1,
			EmailID:         3,
			PositionGroupID: 10, // 機械学習エンジニア
		},
		{
			UserID:          1,
			EmailID:         3,
			PositionGroupID: 9, // データエンジニア
		},
		// email004 (Go案件) の関連
		{
			UserID:          1,
			EmailID:         4,
			PositionGroupID: 7, // バックエンドエンジニア
		},
		{
			UserID:          1,
			EmailID:         4,
			PositionGroupID: 5, // アーキテクト
		},
		// email005 (フルスタック案件) の関連
		{
			UserID:          1,
			EmailID:         5,
			PositionGroupID: 8, // フルスタックエンジニア
		},
		{
			UserID:          1,
			EmailID:         5,
			PositionGroupID: 6, // フロントエンドエンジニア
		},
		{
			UserID:          1,
			EmailID:         5,
			PositionGroupID: 7, // バックエンドエンジニア
		},
	}

	for _, emailPositionGroup := range emailPositionGroups {
		err := tx.Create(&emailPositionGroup).Error
		if err != nil {
			return err
		}
	}

	return err
}
