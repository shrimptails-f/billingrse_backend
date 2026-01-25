package seeders

import (
	"business/tools/migrations/model"

	"gorm.io/gorm"
)

// CreatePositionWord はポジション表記ゆれのサンプルデータを投入する。
func CreatePositionWord(tx *gorm.DB) error {
	var err error

	positionWords := []model.PositionWord{
		// PM関連
		{PositionGroupID: 1, Word: "PM"},
		{PositionGroupID: 1, Word: "ＰＭ"},
		{PositionGroupID: 1, Word: "Project Manager"},
		{PositionGroupID: 1, Word: "プロジェクトマネージャー"},
		{PositionGroupID: 1, Word: "プロマネ"},

		// PL関連
		{PositionGroupID: 2, Word: "PL"},
		{PositionGroupID: 2, Word: "ＰＬ"},
		{PositionGroupID: 2, Word: "Project Leader"},
		{PositionGroupID: 2, Word: "プロジェクトリーダー"},
		{PositionGroupID: 2, Word: "チームリーダー"},

		// SE関連
		{PositionGroupID: 3, Word: "SE"},
		{PositionGroupID: 3, Word: "ＳＥ"},
		{PositionGroupID: 3, Word: "System Engineer"},
		{PositionGroupID: 3, Word: "システムエンジニア"},
		{PositionGroupID: 3, Word: "エンジニア"},

		// PG関連
		{PositionGroupID: 4, Word: "PG"},
		{PositionGroupID: 4, Word: "ＰＧ"},
		{PositionGroupID: 4, Word: "Programmer"},
		{PositionGroupID: 4, Word: "プログラマー"},
		{PositionGroupID: 4, Word: "プログラマ"},

		// アーキテクト関連
		{PositionGroupID: 5, Word: "アーキテクト"},
		{PositionGroupID: 5, Word: "Architect"},
		{PositionGroupID: 5, Word: "システムアーキテクト"},
		{PositionGroupID: 5, Word: "ソリューションアーキテクト"},

		// フロントエンドエンジニア関連
		{PositionGroupID: 6, Word: "フロントエンドエンジニア"},
		{PositionGroupID: 6, Word: "Frontend Engineer"},
		{PositionGroupID: 6, Word: "FE"},
		{PositionGroupID: 6, Word: "ＦＥ"},
		{PositionGroupID: 6, Word: "フロントエンド"},

		// バックエンドエンジニア関連
		{PositionGroupID: 7, Word: "バックエンドエンジニア"},
		{PositionGroupID: 7, Word: "Backend Engineer"},
		{PositionGroupID: 7, Word: "BE"},
		{PositionGroupID: 7, Word: "ＢＥ"},
		{PositionGroupID: 7, Word: "バックエンド"},
		{PositionGroupID: 7, Word: "サーバーサイドエンジニア"},

		// フルスタックエンジニア関連
		{PositionGroupID: 8, Word: "フルスタックエンジニア"},
		{PositionGroupID: 8, Word: "Full Stack Engineer"},
		{PositionGroupID: 8, Word: "フルスタック"},

		// データエンジニア関連
		{PositionGroupID: 9, Word: "データエンジニア"},
		{PositionGroupID: 9, Word: "Data Engineer"},
		{PositionGroupID: 9, Word: "DE"},
		{PositionGroupID: 9, Word: "ＤＥ"},

		// 機械学習エンジニア関連
		{PositionGroupID: 10, Word: "機械学習エンジニア"},
		{PositionGroupID: 10, Word: "Machine Learning Engineer"},
		{PositionGroupID: 10, Word: "ML Engineer"},
		{PositionGroupID: 10, Word: "MLエンジニア"},
		{PositionGroupID: 10, Word: "AIエンジニア"},

		// インフラエンジニア関連
		{PositionGroupID: 11, Word: "インフラエンジニア"},
		{PositionGroupID: 11, Word: "Infrastructure Engineer"},
		{PositionGroupID: 11, Word: "インフラ"},
		{PositionGroupID: 11, Word: "サーバーエンジニア"},

		// DevOpsエンジニア関連
		{PositionGroupID: 12, Word: "DevOpsエンジニア"},
		{PositionGroupID: 12, Word: "DevOps Engineer"},
		{PositionGroupID: 12, Word: "DevOps"},
		{PositionGroupID: 12, Word: "SRE"},

		// QAエンジニア関連
		{PositionGroupID: 13, Word: "QAエンジニア"},
		{PositionGroupID: 13, Word: "QA Engineer"},
		{PositionGroupID: 13, Word: "QA"},
		{PositionGroupID: 13, Word: "品質保証"},

		// テスター関連
		{PositionGroupID: 14, Word: "テスター"},
		{PositionGroupID: 14, Word: "Tester"},
		{PositionGroupID: 14, Word: "テスト"},

		// UI/UXデザイナー関連
		{PositionGroupID: 15, Word: "UI/UXデザイナー"},
		{PositionGroupID: 15, Word: "UIデザイナー"},
		{PositionGroupID: 15, Word: "UXデザイナー"},
		{PositionGroupID: 15, Word: "デザイナー"},
		{PositionGroupID: 15, Word: "UI/UX"},
	}

	for _, positionWord := range positionWords {
		err := tx.Create(&positionWord).Error
		if err != nil {
			return err
		}
	}

	return err
}
