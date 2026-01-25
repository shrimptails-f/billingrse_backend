package seeders

import (
	"business/tools/migrations/model"

	"gorm.io/gorm"
)

// CreateKeyWord はキーワード（表記ゆれ）のサンプルデータを投入する。
func CreateKeyWord(tx *gorm.DB) error {
	var err error
	keyWords := []model.KeyWord{
		// Java関連の表記ゆれ
		{
			ID:   1,
			Word: "Java",
		},
		{
			ID:   2,
			Word: "Java(Spring)",
		},
		// Python関連の表記ゆれ
		{
			ID:   4,
			Word: "Python",
		},
		// JavaScript関連の表記ゆれ
		{
			ID:   7,
			Word: "JavaScript",
		},
		{
			ID:   9,
			Word: "JS",
		},
		// TypeScript関連の表記ゆれ
		{
			ID:   11,
			Word: "TypeScript",
		},
		{
			ID:   13,
			Word: "TS",
		},
		// Go関連の表記ゆれ
		{
			ID:   15,
			Word: "Go",
		},
		{
			ID:   16,
			Word: "Golang",
		},
		// Spring Boot関連の表記ゆれ
		{
			ID:   18,
			Word: "Spring Boot",
		},
		{
			ID:   20,
			Word: "Spring",
		},
		// React関連の表記ゆれ
		{
			ID:   21,
			Word: "React",
		},
		{
			ID:   22,
			Word: "React.js",
		},
		{
			ID:   23,
			Word: "ReactJS",
		},
		// Vue.js関連の表記ゆれ
		{
			ID:   24,
			Word: "Vue.js",
		},
		{
			ID:   25,
			Word: "Vue",
		},
		{
			ID:   26,
			Word: "VueJS",
		},
		// Django関連の表記ゆれ
		{
			ID:   27,
			Word: "Django",
		},
		// AWS関連の表記ゆれ
		{
			ID:   29,
			Word: "AWS",
		},
		{
			ID:   30,
			Word: "Amazon Web Services",
		},
		{
			ID:   31,
			Word: "アマゾンウェブサービス",
		},
		// Docker関連の表記ゆれ
		{
			ID:   32,
			Word: "Docker",
		},
		// Kubernetes関連の表記ゆれ
		{
			ID:   34,
			Word: "Kubernetes",
		},
		{
			ID:   35,
			Word: "K8s",
		},
		// MySQL関連の表記ゆれ
		{
			ID:   37,
			Word: "MySQL",
		},
		// PostgreSQL関連の表記ゆれ
		{
			ID:   39,
			Word: "PostgreSQL",
		},
		// 設計関連の表記ゆれ
		{
			ID:   42,
			Word: "設計",
		},
		{
			ID:   43,
			Word: "基本設計",
		},
		{
			ID:   44,
			Word: "詳細設計",
		},
		// テスト関連の表記ゆれ
		{
			ID:   45,
			Word: "テスト",
		},
		{
			ID:   46,
			Word: "単体テスト",
		},
		{
			ID:   47,
			Word: "結合テスト",
		},
		// レビュー関連の表記ゆれ
		{
			ID:   48,
			Word: "レビュー",
		},
		{
			ID:   49,
			Word: "コードレビュー",
		},
		{
			ID:   50,
			Word: "設計レビュー",
		},
	}

	for _, keyWord := range keyWords {
		err := tx.Create(&keyWord).Error
		if err != nil {
			return err
		}
	}

	return err
}
