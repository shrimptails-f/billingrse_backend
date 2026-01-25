package seeders

import (
	"business/tools/migrations/model"

	"gorm.io/gorm"
)

// CreatePositionGroup はポジショングループのサンプルデータを投入する。
func CreatePositionGroup(tx *gorm.DB) error {
	var err error

	positionGroups := []model.PositionGroup{
		{PositionGroupID: 1, Name: "PM"},
		{PositionGroupID: 2, Name: "PL"},
		{PositionGroupID: 3, Name: "SE"},
		{PositionGroupID: 4, Name: "PG"},
		{PositionGroupID: 5, Name: "アーキテクト"},
		{PositionGroupID: 6, Name: "フロントエンドエンジニア"},
		{PositionGroupID: 7, Name: "バックエンドエンジニア"},
		{PositionGroupID: 8, Name: "フルスタックエンジニア"},
		{PositionGroupID: 9, Name: "データエンジニア"},
		{PositionGroupID: 10, Name: "機械学習エンジニア"},
		{PositionGroupID: 11, Name: "インフラエンジニア"},
		{PositionGroupID: 12, Name: "DevOpsエンジニア"},
		{PositionGroupID: 13, Name: "QAエンジニア"},
		{PositionGroupID: 14, Name: "テスター"},
		{PositionGroupID: 15, Name: "UI/UXデザイナー"},
	}

	for _, positionGroup := range positionGroups {
		err := tx.Create(&positionGroup).Error
		if err != nil {
			return err
		}
	}

	return err
}
