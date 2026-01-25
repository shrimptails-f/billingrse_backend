package seeders

import (
	"business/tools/migrations/model"

	"gorm.io/gorm"
)

// CreateWorkTypeGroup は業務グループのサンプルデータを投入する。
func CreateWorkTypeGroup(tx *gorm.DB) error {
	var err error

	workTypeGroups := []model.WorkTypeGroup{
		{WorkTypeGroupID: 1, Name: "要件定義"},
		{WorkTypeGroupID: 2, Name: "基本設計"},
		{WorkTypeGroupID: 3, Name: "詳細設計"},
		{WorkTypeGroupID: 4, Name: "フロントエンド開発"},
		{WorkTypeGroupID: 5, Name: "バックエンド開発"},
		{WorkTypeGroupID: 6, Name: "API開発"},
		{WorkTypeGroupID: 7, Name: "データベース設計"},
		{WorkTypeGroupID: 8, Name: "インフラ構築"},
		{WorkTypeGroupID: 9, Name: "テスト設計"},
		{WorkTypeGroupID: 10, Name: "単体テスト"},
		{WorkTypeGroupID: 11, Name: "結合テスト"},
		{WorkTypeGroupID: 12, Name: "システムテスト"},
		{WorkTypeGroupID: 13, Name: "運用保守"},
		{WorkTypeGroupID: 14, Name: "パフォーマンス改善"},
		{WorkTypeGroupID: 15, Name: "セキュリティ対応"},
		{WorkTypeGroupID: 16, Name: "CI/CD構築"},
		{WorkTypeGroupID: 17, Name: "監視設定"},
		{WorkTypeGroupID: 18, Name: "ドキュメント作成"},
		{WorkTypeGroupID: 19, Name: "コードレビュー"},
		{WorkTypeGroupID: 20, Name: "リファクタリング"},
	}

	for _, workTypeGroup := range workTypeGroups {
		err := tx.Create(&workTypeGroup).Error
		if err != nil {
			return err
		}
	}

	return err
}
