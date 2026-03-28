package model

import "time"

// Vendor は canonical Vendor のマスタを表す。
type Vendor struct {
	ID             uint   `gorm:"primaryKey;autoIncrement"`
	UserID         uint   `gorm:"not null;uniqueIndex:uni_vendors_user_normalized_name,priority:1"`
	Name           string `gorm:"size:255;not null"`
	NormalizedName string `gorm:"size:255;not null;uniqueIndex:uni_vendors_user_normalized_name,priority:2"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// TableName は Vendor モデルのテーブル名を返す。
func (Vendor) TableName() string {
	return "vendors"
}

// VendorAlias は決定的な vendor 解決に使う alias マスタを表す。
type VendorAlias struct {
	ID              uint   `gorm:"primaryKey;autoIncrement"`
	UserID          uint   `gorm:"not null;index:idx_vendor_aliases_user_type_normalized,priority:1;uniqueIndex:uni_vendor_aliases_user_type_value,priority:1"`
	VendorID        uint   `gorm:"not null;index:idx_vendor_aliases_vendor_id"`
	AliasType       string `gorm:"size:50;not null;index:idx_vendor_aliases_user_type_normalized,priority:2;uniqueIndex:uni_vendor_aliases_user_type_value,priority:2"`
	AliasValue      string `gorm:"type:text;not null"`
	NormalizedValue string `gorm:"size:255;not null;index:idx_vendor_aliases_user_type_normalized,priority:3;uniqueIndex:uni_vendor_aliases_user_type_value,priority:3"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TableName は VendorAlias モデルのテーブル名を返す。
func (VendorAlias) TableName() string {
	return "vendor_aliases"
}
