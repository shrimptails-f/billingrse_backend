package model

import "time"

// EmailAnalysisLog は非同期メール解析ジョブのライフサイクル情報を保持します。
type EmailAnalysisLog struct {
	ID                     uint       `gorm:"primaryKey;autoIncrement"`
	UserID                 uint       `gorm:"not null;index:idx_email_analysis_logs_user_queued,priority:1"`
	Status                 string     `gorm:"size:16;not null"`
	LabelName              string     `gorm:"size:255;not null"`
	StartDate              time.Time  `gorm:"type:date;not null"`
	RetrievedCount         int        `gorm:"not null;default:0"`
	DetailFetchFailedCount int        `gorm:"not null;default:0"`
	AnalysisCount          int        `gorm:"not null;default:0"`
	AnalysisFailedCount    int        `gorm:"not null;default:0"`
	SavedCount             int        `gorm:"not null;default:0"`
	ResultNote             string     `gorm:"size:255"`
	ErrorMessage           string     `gorm:"type:text"`
	QueuedAt               time.Time  `gorm:"not null;autoCreateTime;index:idx_email_analysis_logs_user_queued,priority:2,sort:desc"`
	StartedAt              *time.Time `gorm:"default:null"`
	FinishedAt             *time.Time `gorm:"default:null"`
	ErrorAt                *time.Time `gorm:"default:null"`
	CreatedAt              time.Time  `gorm:"autoCreateTime"`
	UpdatedAt              time.Time  `gorm:"autoUpdateTime"`
}

// TableName はテーブル名を返します。
func (EmailAnalysisLog) TableName() string {
	return "email_analysis_logs"
}
