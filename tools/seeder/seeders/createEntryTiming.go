package seeders

import (
	"gorm.io/gorm"
)

// CreateEntryTiming は入場時期のサンプルデータを投入する。
// 注意: このseederは現在使用されていません。
// EntryTimingはcreateEmail.goでEmailProjectと一緒に作成されます。
func CreateEntryTiming(tx *gorm.DB) error {
	// 現在はcreateEmail.goでEmailProjectのリレーションとして作成されるため、
	// このseederは使用しません。
	// 必要に応じて個別のEntryTimingデータを作成する場合にのみ使用してください。

	return nil
}
