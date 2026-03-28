package infrastructure

import "time"

type vendorRecord struct {
	ID             uint      `gorm:"column:id;primaryKey;autoIncrement"`
	UserID         uint      `gorm:"column:user_id;not null;uniqueIndex:uni_vendors_user_normalized_name,priority:1"`
	Name           string    `gorm:"column:name;size:255;not null"`
	NormalizedName string    `gorm:"column:normalized_name;size:255;not null;uniqueIndex:uni_vendors_user_normalized_name,priority:2"`
	CreatedAt      time.Time `gorm:"column:created_at;not null"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null"`
}

// TableName は vendors テーブルを明示する。
func (vendorRecord) TableName() string {
	return "vendors"
}

// vendorAliasRecord は read/write repository が参照する vendor_aliases の内部表現。
type vendorAliasRecord struct {
	ID              uint      `gorm:"column:id;primaryKey;autoIncrement"`
	UserID          uint      `gorm:"column:user_id;not null;index:idx_vendor_aliases_user_type_normalized,priority:1;uniqueIndex:uni_vendor_aliases_user_type_value,priority:1"`
	VendorID        uint      `gorm:"column:vendor_id;not null;index:idx_vendor_aliases_vendor_id"`
	AliasType       string    `gorm:"column:alias_type;size:50;not null;index:idx_vendor_aliases_user_type_normalized,priority:2;uniqueIndex:uni_vendor_aliases_user_type_value,priority:2"`
	AliasValue      string    `gorm:"column:alias_value;type:text;not null"`
	NormalizedValue string    `gorm:"column:normalized_value;size:255;not null;index:idx_vendor_aliases_user_type_normalized,priority:3;uniqueIndex:uni_vendor_aliases_user_type_value,priority:3"`
	CreatedAt       time.Time `gorm:"column:created_at;not null"`
	UpdatedAt       time.Time `gorm:"column:updated_at;not null"`
}

// TableName は vendor_aliases テーブルを明示する。
func (vendorAliasRecord) TableName() string {
	return "vendor_aliases"
}

// resolvedAliasRecord は JOIN 済み lookup 結果を一時的に受ける read model。
type resolvedAliasRecord struct {
	AliasID         uint      `gorm:"column:alias_id"`
	VendorID        uint      `gorm:"column:vendor_id"`
	VendorUserID    uint      `gorm:"column:vendor_user_id"`
	VendorName      string    `gorm:"column:vendor_name"`
	AliasValue      string    `gorm:"column:alias_value"`
	NormalizedValue string    `gorm:"column:normalized_value"`
	AliasCreatedAt  time.Time `gorm:"column:alias_created_at"`
}
