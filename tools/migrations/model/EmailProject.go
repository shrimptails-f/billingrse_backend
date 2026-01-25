package model

import (
	"time"
)

// EmailProject（案件メール専用情報）
type EmailProject struct {
	ID           uint    `gorm:"primaryKey;autoIncrement"`                                    // オートインクリメントID
	UserID       uint    `gorm:"not null;index"`                                              // ユーザーID
	EmailID      uint    `gorm:"index"`                                                       // メールID（emails.idと同じ）
	ProjectTitle *string `gorm:"size:255"`                                                    // 案件名
	User         User    `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"` // ユーザー

	// 表示用（カンマ区切り）
	EntryTiming *string `gorm:"type:text"` // 入場時期（"2025/06/01,2025/07/01"）
	Languages   *string `gorm:"type:text"` // 言語（"PHP,TypeScript"）
	Frameworks  *string `gorm:"type:text"` // フレームワーク（"React,Laravel"）
	Positions   *string `gorm:"type:text"` // ポジション（"PM,SE"）
	WorkTypes   *string `gorm:"type:text"` // 業務内容（"バックエンド実装,インフラ構築"）
	MustSkills  *string `gorm:"type:text"` // MUSTスキル（"CMS知識,PowerCMS"）
	WantSkills  *string `gorm:"type:text"` // WANTスキル（"MT,Adobe製品経験"）

	// その他項目
	EndTiming       *string   `gorm:"size:255"`       // 終了時期
	WorkLocation    *string   `gorm:"size:255;index"` // 勤務場所
	PriceFrom       *int      `gorm:"type:int"`       // 単価FROM
	PriceTo         *int      `gorm:"type:int"`       // 単価TO
	RemoteType      *string   `gorm:"size:50"`        // リモート区分
	RemoteFrequency *string   `gorm:"size:255"`       // リモート頻度
	CreatedAt       time.Time // 作成日時
	UpdatedAt       time.Time // 更新日時

	// リレーション
	Email Email `gorm:"foreignKey:EmailID;references:ID"` // 親メール
}
