package model

import (
	"time"
)

// AgentChangeLog represents the agent_change_logs table for audit logging
type AgentChangeLog struct {
	ID        uint      `gorm:"column:id;primaryKey;autoIncrement"`
	AgentID   uint      `gorm:"column:agent_id;not null;index"`
	UserID    uint      `gorm:"column:user_id;not null;index"`
	Action    string    `gorm:"column:action;not null;size:50"`
	Note      string    `gorm:"column:note;type:text"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

// TableName specifies the table name for AgentChangeLog
func (AgentChangeLog) TableName() string {
	return "agent_change_logs"
}
