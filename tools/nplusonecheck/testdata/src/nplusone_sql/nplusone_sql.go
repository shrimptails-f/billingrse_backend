// このファイルは「ループ内で database/sql のクエリメソッドを直接呼ぶ」パターンを見ます。
// `QueryRow(...)` は検知し、接続設定だけの呼び出しは検知しないことを確認します。
// database/sql を基準ケースにしているのは、最も単純な直接クエリ呼び出しで
// 基本動作を確認しやすいためです。
package nplusone_sql

import "database/sql"

func direct(db *sql.DB, ids []int) {
	for _, id := range ids {
		_ = db.QueryRow("SELECT id FROM users WHERE id = ?", id) // want "possible N\\+1 query inside loop"
	}
}

func notQuery(db *sql.DB, ids []int) {
	for range ids {
		db.SetMaxOpenConns(10)
	}
}
