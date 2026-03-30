// このファイルは「ループ内では helper 関数を呼ぶだけで、
// その先の同一 package 関数でクエリが実行される」パターンを見ます。
// ここで database/sql を使っているのは、再帰追跡の検証が主目的であり、
// GORM のメソッドチェーン固有の要素を混ぜずに最小構成で確認したいためです。
package nplusone_helper

import "database/sql"

func loadUser(db *sql.DB, id int) {
	_ = db.QueryRow("SELECT id FROM users WHERE id = ?", id)
}

func helper(db *sql.DB, id int) {
	loadUser(db, id)
}

func indirect(db *sql.DB, ids []int) {
	for _, id := range ids {
		helper(db, id) // want "possible N\\+1 query inside loop"
	}
}

func noop(_ *sql.DB, _ int) {}

func noQuery(db *sql.DB, ids []int) {
	for _, id := range ids {
		noop(db, id)
	}
}
