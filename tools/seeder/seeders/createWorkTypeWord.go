package seeders

import (
	"business/tools/migrations/model"

	"gorm.io/gorm"
)

// CreateWorkTypeWord は業務表記ゆれのサンプルデータを投入する。
func CreateWorkTypeWord(tx *gorm.DB) error {
	var err error

	workTypeWords := []model.WorkTypeWord{
		// 要件定義関連
		{WorkTypeGroupID: 1, Word: "要件定義"},
		{WorkTypeGroupID: 1, Word: "要求分析"},
		{WorkTypeGroupID: 1, Word: "要件整理"},
		{WorkTypeGroupID: 1, Word: "RD"},

		// 基本設計関連
		{WorkTypeGroupID: 2, Word: "基本設計"},
		{WorkTypeGroupID: 2, Word: "外部設計"},
		{WorkTypeGroupID: 2, Word: "概要設計"},
		{WorkTypeGroupID: 2, Word: "BD"},

		// 詳細設計関連
		{WorkTypeGroupID: 3, Word: "詳細設計"},
		{WorkTypeGroupID: 3, Word: "内部設計"},
		{WorkTypeGroupID: 3, Word: "DD"},

		// フロントエンド開発関連
		{WorkTypeGroupID: 4, Word: "フロントエンド開発"},
		{WorkTypeGroupID: 4, Word: "FE開発"},
		{WorkTypeGroupID: 4, Word: "フロント実装"},
		{WorkTypeGroupID: 4, Word: "UI実装"},
		{WorkTypeGroupID: 4, Word: "画面開発"},

		// バックエンド開発関連
		{WorkTypeGroupID: 5, Word: "バックエンド開発"},
		{WorkTypeGroupID: 5, Word: "BE開発"},
		{WorkTypeGroupID: 5, Word: "バックエンド実装"},
		{WorkTypeGroupID: 5, Word: "サーバーサイド開発"},
		{WorkTypeGroupID: 5, Word: "ロジック実装"},

		// API開発関連
		{WorkTypeGroupID: 6, Word: "API開発"},
		{WorkTypeGroupID: 6, Word: "API実装"},
		{WorkTypeGroupID: 6, Word: "REST API"},
		{WorkTypeGroupID: 6, Word: "GraphQL"},
		{WorkTypeGroupID: 6, Word: "Web API"},

		// データベース設計関連
		{WorkTypeGroupID: 7, Word: "データベース設計"},
		{WorkTypeGroupID: 7, Word: "DB設計"},
		{WorkTypeGroupID: 7, Word: "テーブル設計"},
		{WorkTypeGroupID: 7, Word: "スキーマ設計"},

		// インフラ構築関連
		{WorkTypeGroupID: 8, Word: "インフラ構築"},
		{WorkTypeGroupID: 8, Word: "インフラ設計"},
		{WorkTypeGroupID: 8, Word: "環境構築"},
		{WorkTypeGroupID: 8, Word: "サーバー構築"},
		{WorkTypeGroupID: 8, Word: "クラウド構築"},

		// テスト設計関連
		{WorkTypeGroupID: 9, Word: "テスト設計"},
		{WorkTypeGroupID: 9, Word: "テスト計画"},
		{WorkTypeGroupID: 9, Word: "テストケース作成"},

		// 単体テスト関連
		{WorkTypeGroupID: 10, Word: "単体テスト"},
		{WorkTypeGroupID: 10, Word: "UT"},
		{WorkTypeGroupID: 10, Word: "Unit Test"},
		{WorkTypeGroupID: 10, Word: "ユニットテスト"},

		// 結合テスト関連
		{WorkTypeGroupID: 11, Word: "結合テスト"},
		{WorkTypeGroupID: 11, Word: "IT"},
		{WorkTypeGroupID: 11, Word: "Integration Test"},
		{WorkTypeGroupID: 11, Word: "インテグレーションテスト"},

		// システムテスト関連
		{WorkTypeGroupID: 12, Word: "システムテスト"},
		{WorkTypeGroupID: 12, Word: "ST"},
		{WorkTypeGroupID: 12, Word: "System Test"},
		{WorkTypeGroupID: 12, Word: "総合テスト"},

		// 運用保守関連
		{WorkTypeGroupID: 13, Word: "運用保守"},
		{WorkTypeGroupID: 13, Word: "保守運用"},
		{WorkTypeGroupID: 13, Word: "運用"},
		{WorkTypeGroupID: 13, Word: "保守"},
		{WorkTypeGroupID: 13, Word: "メンテナンス"},

		// パフォーマンス改善関連
		{WorkTypeGroupID: 14, Word: "パフォーマンス改善"},
		{WorkTypeGroupID: 14, Word: "性能改善"},
		{WorkTypeGroupID: 14, Word: "最適化"},
		{WorkTypeGroupID: 14, Word: "チューニング"},

		// セキュリティ対応関連
		{WorkTypeGroupID: 15, Word: "セキュリティ対応"},
		{WorkTypeGroupID: 15, Word: "セキュリティ強化"},
		{WorkTypeGroupID: 15, Word: "脆弱性対応"},
		{WorkTypeGroupID: 15, Word: "セキュリティ監査"},

		// CI/CD構築関連
		{WorkTypeGroupID: 16, Word: "CI/CD構築"},
		{WorkTypeGroupID: 16, Word: "CI/CD"},
		{WorkTypeGroupID: 16, Word: "継続的インテグレーション"},
		{WorkTypeGroupID: 16, Word: "継続的デプロイ"},
		{WorkTypeGroupID: 16, Word: "自動化"},

		// 監視設定関連
		{WorkTypeGroupID: 17, Word: "監視設定"},
		{WorkTypeGroupID: 17, Word: "モニタリング"},
		{WorkTypeGroupID: 17, Word: "ログ監視"},
		{WorkTypeGroupID: 17, Word: "アラート設定"},

		// ドキュメント作成関連
		{WorkTypeGroupID: 18, Word: "ドキュメント作成"},
		{WorkTypeGroupID: 18, Word: "資料作成"},
		{WorkTypeGroupID: 18, Word: "仕様書作成"},
		{WorkTypeGroupID: 18, Word: "マニュアル作成"},

		// コードレビュー関連
		{WorkTypeGroupID: 19, Word: "コードレビュー"},
		{WorkTypeGroupID: 19, Word: "レビュー"},
		{WorkTypeGroupID: 19, Word: "Code Review"},
		{WorkTypeGroupID: 19, Word: "品質チェック"},

		// リファクタリング関連
		{WorkTypeGroupID: 20, Word: "リファクタリング"},
		{WorkTypeGroupID: 20, Word: "Refactoring"},
		{WorkTypeGroupID: 20, Word: "コード改善"},
		{WorkTypeGroupID: 20, Word: "技術的負債解消"},
	}

	for _, workTypeWord := range workTypeWords {
		err := tx.Create(&workTypeWord).Error
		if err != nil {
			return err
		}
	}

	return err
}
