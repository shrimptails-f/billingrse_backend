// このファイルは「ループ内では method を呼ぶだけで、
// その先の同一 package method でクエリが実行される」パターンを見ます。
// ここで database/sql を使っているのは、method 再帰追跡そのものを見たくて、
// ORM 依存の差分ではなく「同一 package の呼び出し解決」を単純化したいためです。
package nplusone_method

import "database/sql"

type repo struct {
	db *sql.DB
}

func (r *repo) loadUser(id int) {
	r.queryUser(id)
}

func (r *repo) queryUser(id int) {
	_ = r.db.QueryRow("SELECT id FROM users WHERE id = ?", id)
}

func indirectMethod(r *repo, ids []int) {
	for _, id := range ids {
		r.loadUser(id) // want "possible N\\+1 query inside loop"
	}
}

func (r *repo) touch(_ int) {}

func noQueryMethod(r *repo, ids []int) {
	for _, id := range ids {
		r.touch(id)
	}
}
