// このファイルは「他 package の helper 関数を呼ぶだけで、
// その先の import 先 package 関数でクエリが実行される」パターンを見ます。
// current package と同じ workspace 配下にある import 先だけを追う実装の確認用です。
package nplusone_crosspkg_helper

import (
	"database/sql"

	"sharedhelper"
)

func indirect(db *sql.DB, ids []int) {
	for _, id := range ids {
		sharedhelper.Helper(db, id) // want "possible N\\+1 query inside loop"
	}
}

func noQuery(db *sql.DB, ids []int) {
	for _, id := range ids {
		sharedhelper.Noop(db, id)
	}
}
