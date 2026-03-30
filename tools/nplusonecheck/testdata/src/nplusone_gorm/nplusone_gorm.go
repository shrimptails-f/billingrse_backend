// このファイルは「ループ内で GORM のクエリメソッドを直接呼ぶ」パターンを見ます。
// `Where(...).First(...)` は検知し、`Session(...)` だけなら検知しないことを確認します。
package nplusone_gorm

import "gorm.io/gorm"

type user struct {
	ID int
}

func direct(db *gorm.DB, ids []int) {
	for _, id := range ids {
		var item user
		db.Where("id = ?", id).First(&item) // want "possible N\\+1 query inside loop"
	}
}

func notQuery(db *gorm.DB, ids []int) {
	for range ids {
		db = db.Session(&gorm.Session{})
	}
}
