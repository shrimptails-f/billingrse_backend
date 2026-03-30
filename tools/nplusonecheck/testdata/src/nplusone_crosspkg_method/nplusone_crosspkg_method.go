// このファイルは「他 package の method を呼ぶだけで、
// その先の import 先 package method でクエリが実行される」パターンを見ます。
// まずは同一 workspace 配下の package に限定して追跡する前提です。
package nplusone_crosspkg_method

import "sharedmethod"

func indirect(r *sharedmethod.Repo, ids []int) {
	for _, id := range ids {
		r.LoadUser(id) // want "possible N\\+1 query inside loop"
	}
}

func noQuery(r *sharedmethod.Repo, ids []int) {
	for _, id := range ids {
		r.Touch(id)
	}
}
