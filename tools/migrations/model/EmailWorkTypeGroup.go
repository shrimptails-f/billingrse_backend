package model

// EmailWorkTypeGroup（メールと業務グループの中間）
type EmailWorkTypeGroup struct {
	UserID          uint `gorm:"primaryKey"`                                                  // ユーザーID
	EmailID         uint `gorm:"primaryKey"`                                                  // メールID
	WorkTypeGroupID uint `gorm:"primaryKey"`                                                  // 業務グループID
	User            User `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"` // ユーザー
}
