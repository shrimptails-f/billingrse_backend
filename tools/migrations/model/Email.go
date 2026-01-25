package model

import (
	"time"
)

// Email（メール基本情報）
type Email struct {
	ID           uint      `gorm:"primaryKey;autoIncrement"`                                                        // オートインクリメントID
	BranchNo     int       `gorm:"not null;default:0;uniqueIndex:uq_emails_user_gmail_branch"`                      // 枝番
	UserID       uint      `gorm:"not null;index:idx_emails_user_received;uniqueIndex:uq_emails_user_gmail_branch"` // ユーザーID
	GmailID      string    `gorm:"size:255;uniqueIndex:uq_emails_user_gmail_branch"`                                // GメールID
	Subject      string    `gorm:"type:text;not null"`                                                              // 件名
	SenderName   string    `gorm:"size:255"`                                                                        // 差出人名
	SenderEmail  string    `gorm:"size:255;index"`                                                                  // メールアドレス
	ReceivedDate time.Time `gorm:"index:idx_emails_user_received"`                                                  // 受信日
	Body         *string   `gorm:"type:longtext"`                                                                   // 本文
	Category     string    `gorm:"size:50;index"`                                                                   // 種別（案件 / 人材提案）

	IsRead bool `gorm:"not null;default:false"` // 既読
	IsGood bool `gorm:"not null;default:false"` // いいね
	IsBad  bool `gorm:"not null;default:false"` // びみょうかも

	CreatedAt time.Time // 作成日時
	UpdatedAt time.Time // 更新日時

	// リレーション
	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"` // ユーザー

	// 子テーブル
	EmailProject        *EmailProject        `gorm:"foreignKey:EmailID;references:ID"` // 案件情報（1対1）
	EmailCandidate      *EmailCandidate      `gorm:"foreignKey:EmailID;references:ID"` // 人材情報（1対1）
	EntryTimings        []EntryTiming        `gorm:"foreignKey:EmailID;references:ID"` // 入場時期（1対多）
	EmailKeywordGroups  []EmailKeywordGroup  `gorm:"foreignKey:EmailID;references:ID"` // 技術キーワード（1対多）
	EmailPositionGroups []EmailPositionGroup `gorm:"foreignKey:EmailID;references:ID"` // ポジション（1対多）
	EmailWorkTypeGroups []EmailWorkTypeGroup `gorm:"foreignKey:EmailID;references:ID"` // 業務内容（1対多）
}
