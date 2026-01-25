package seeders

import (
	"business/tools/migrations/model"
	"time"

	"gorm.io/gorm"
)

// CreateEmail はメールのサンプルデータを投入する。
func CreateEmail(tx *gorm.DB) error {
	var err error

	// ポインタ型のヘルパー関数
	stringPtr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }

	// branch_no: 同一Gmail IDに対して複数の枝番を持つレコードを保存可能
	// 例: email001 には branch_no=0 と branch_no=1 の2つのレコードが存在
	emails := []model.Email{
		{
			UserID:       1,
			GmailID:      "email001",
			BranchNo:     0,
			Subject:      "【案件】Javaエンジニア募集（リモート可）",
			SenderName:   "田中太郎",
			SenderEmail:  "tanaka@example.com",
			ReceivedDate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Body:         stringPtr("Java Spring Bootを使用したWebアプリケーション開発案件です。リモートワーク可能です。"),
			Category:     "案件",
			// 案件情報（EmailProject）
			EmailProject: &model.EmailProject{
				UserID:          1,
				EmailID:         1,
				ProjectTitle:    stringPtr("ECサイト構築プロジェクト"),
				EntryTiming:     stringPtr("2024/02/01,2024/03/01"),
				Languages:       stringPtr("Java,JavaScript"),
				Frameworks:      stringPtr("Spring Boot,React"),
				Positions:       stringPtr("SE,PG"),
				WorkTypes:       stringPtr("バックエンド実装,フロントエンド実装"),
				MustSkills:      stringPtr("Java経験3年以上,Spring Boot経験"),
				WantSkills:      stringPtr("AWS経験,Docker経験"),
				EndTiming:       stringPtr("2024年8月"),
				WorkLocation:    stringPtr("東京都渋谷区（リモート可）"),
				PriceFrom:       intPtr(600000),
				PriceTo:         intPtr(800000),
				RemoteType:      stringPtr("フルリモート可"),
				RemoteFrequency: stringPtr("週5日"),
			},
			// EntryTimingsリレーション
			EntryTimings: []model.EntryTiming{
				{UserID: 1, EmailID: 1, StartDate: "2024/02/01"},
				{UserID: 1, EmailID: 1, StartDate: "2024/03/01"},
			},
		},
		{
			UserID:       1,
			GmailID:      "email001",
			BranchNo:     1,
			Subject:      "【案件】Javaエンジニア募集（別案件・在宅メイン）",
			SenderName:   "田中太郎",
			SenderEmail:  "tanaka@example.com",
			ReceivedDate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Body:         stringPtr("同じGmail IDから抽出された別案件です。在宅メインの金融系システム開発。"),
			Category:     "案件",
			// 案件情報（EmailProject）
			EmailProject: &model.EmailProject{
				UserID:          1,
				EmailID:         2,
				ProjectTitle:    stringPtr("金融系システム刷新プロジェクト"),
				EntryTiming:     stringPtr("2024/03/01"),
				Languages:       stringPtr("Java,Kotlin"),
				Frameworks:      stringPtr("Spring Boot,Micronaut"),
				Positions:       stringPtr("PL,アーキテクト"),
				WorkTypes:       stringPtr("基幹システム開発,要件定義"),
				MustSkills:      stringPtr("Java経験5年以上,金融業界経験"),
				WantSkills:      stringPtr("Kotlin経験,マイクロサービス設計"),
				EndTiming:       stringPtr("2025年3月"),
				WorkLocation:    stringPtr("東京都千代田区（在宅メイン）"),
				PriceFrom:       intPtr(800000),
				PriceTo:         intPtr(1000000),
				RemoteType:      stringPtr("在宅メイン"),
				RemoteFrequency: stringPtr("週4日在宅、週1日出社"),
			},
			// EntryTimingsリレーション
			EntryTimings: []model.EntryTiming{
				{UserID: 1, EmailID: 2, StartDate: "2024/03/01"},
			},
		},
		{
			UserID:       1,
			GmailID:      "email002",
			BranchNo:     0,
			Subject:      "【人材提案】React開発経験者のご紹介",
			SenderName:   "佐藤花子",
			SenderEmail:  "sato@example.com",
			ReceivedDate: time.Date(2024, 1, 16, 14, 20, 0, 0, time.UTC),
			Body:         stringPtr("React、TypeScriptでの開発経験が豊富なエンジニアをご紹介いたします。"),
			Category:     "人材提案",
		},
		{
			UserID:       1,
			GmailID:      "email003",
			BranchNo:     0,
			Subject:      "【案件】Python機械学習エンジニア募集",
			SenderName:   "山田次郎",
			SenderEmail:  "yamada@example.com",
			ReceivedDate: time.Date(2024, 1, 17, 9, 15, 0, 0, time.UTC),
			Body:         stringPtr("Python、機械学習ライブラリを使用したAIシステム開発案件です。"),
			Category:     "案件",
			// 案件情報（EmailProject）
			EmailProject: &model.EmailProject{
				UserID:          1,
				EmailID:         4,
				ProjectTitle:    stringPtr("AIチャットボット開発"),
				EntryTiming:     stringPtr("2024/03/01,2024/04/01"),
				Languages:       stringPtr("Python,SQL"),
				Frameworks:      stringPtr("TensorFlow,PyTorch,FastAPI"),
				Positions:       stringPtr("AIエンジニア,データサイエンティスト"),
				WorkTypes:       stringPtr("機械学習モデル開発,データ分析"),
				MustSkills:      stringPtr("Python経験3年以上,機械学習ライブラリ経験"),
				WantSkills:      stringPtr("深層学習経験,クラウド経験"),
				EndTiming:       stringPtr("2024年12月"),
				WorkLocation:    stringPtr("東京都港区"),
				PriceFrom:       intPtr(800000),
				PriceTo:         intPtr(1200000),
				RemoteType:      stringPtr("出社必須"),
				RemoteFrequency: stringPtr("週5日出社"),
			},
			// EntryTimingsリレーション
			EntryTimings: []model.EntryTiming{
				{UserID: 1, EmailID: 4, StartDate: "2024/03/01"},
				{UserID: 1, EmailID: 4, StartDate: "2024/04/01"},
			},
		},
		{
			UserID:       1,
			GmailID:      "email004",
			BranchNo:     0,
			Subject:      "【案件】Go言語バックエンド開発者募集",
			SenderName:   "鈴木一郎",
			SenderEmail:  "suzuki@example.com",
			ReceivedDate: time.Date(2024, 1, 18, 16, 45, 0, 0, time.UTC),
			Body:         stringPtr("Go言語でのマイクロサービス開発経験者を募集しています。"),
			Category:     "案件",
			// 案件情報（EmailProject）
			EmailProject: &model.EmailProject{
				UserID:          1,
				EmailID:         5,
				ProjectTitle:    stringPtr("マイクロサービス基盤構築"),
				EntryTiming:     stringPtr("2024/04/01,2024/05/01"),
				Languages:       stringPtr("Go,SQL"),
				Frameworks:      stringPtr("Gin,gRPC,Docker"),
				Positions:       stringPtr("バックエンドエンジニア,インフラエンジニア"),
				WorkTypes:       stringPtr("マイクロサービス開発,API設計"),
				MustSkills:      stringPtr("Go経験2年以上,Docker経験"),
				WantSkills:      stringPtr("Kubernetes経験,AWS経験"),
				EndTiming:       stringPtr("2024年10月"),
				WorkLocation:    stringPtr("東京都千代田区"),
				PriceFrom:       intPtr(750000),
				PriceTo:         intPtr(1000000),
				RemoteType:      stringPtr("フルリモート可"),
				RemoteFrequency: stringPtr("完全リモート"),
			},
			// EntryTimingsリレーション
			EntryTimings: []model.EntryTiming{
				{UserID: 1, EmailID: 5, StartDate: "2024/04/01"},
				{UserID: 1, EmailID: 5, StartDate: "2024/05/01"},
			},
		},
		{
			UserID:       1,
			GmailID:      "email005",
			BranchNo:     0,
			Subject:      "【人材提案】フルスタックエンジニアのご紹介",
			SenderName:   "高橋美咲",
			SenderEmail:  "takahashi@example.com",
			ReceivedDate: time.Date(2024, 1, 19, 11, 30, 0, 0, time.UTC),
			Body:         stringPtr("React、Node.js、AWSでの開発経験が豊富なフルスタックエンジニアです。"),
			Category:     "人材提案",
		},
	}

	for _, email := range emails {
		err := tx.Create(&email).Error
		if err != nil {
			return err
		}
	}

	return err
}
