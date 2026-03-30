// このファイルは「未実行の FuncLit は見ない」ことの確認用です。
// ループ内で無名関数を定義しても、呼ばれなければ検知しない想定です。
package nplusone_negative_funclit

import "database/sql"

func notExecuted(db *sql.DB, ids []int) {
	for _, id := range ids {
		fn := func() {
			_ = db.QueryRow("SELECT id FROM users WHERE id = ?", id)
		}
		_ = fn
	}
}
