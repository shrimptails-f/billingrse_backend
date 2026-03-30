// このファイルは nplusone_gorm テスト用の最小 stub です。
// GORM のメソッドチェーンを型付きで解析できるようにしています。
package gorm

type DB struct{}

type Session struct{}

func (db *DB) Where(_ any, _ ...any) *DB {
	return db
}

func (db *DB) First(_ any, _ ...any) *DB {
	return db
}

func (db *DB) Session(_ *Session) *DB {
	return db
}
