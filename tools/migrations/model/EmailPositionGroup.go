package model

// EmailPositionGroup（メールとポジショングループの中間）
type EmailPositionGroup struct {
	UserID          uint `gorm:"primaryKey"`                                                  // ユーザーID
	EmailID         uint `gorm:"primaryKey"`                                                  // メールID
	PositionGroupID uint `gorm:"primaryKey"`                                                  // ポジショングループID
	User            User `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"` // ユーザー
}
