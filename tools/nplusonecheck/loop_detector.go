package nplusonecheck

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// run は package 内の各ファイルを走査し、
// for / range の本体に対してクエリ実行につながる呼び出しがないかを見ます。
func run(pass *analysis.Pass) {
	state := newLocalCallState(pass)

	for _, file := range pass.Files {
		// file は 1 ファイル分の AST です。
		// ここではファイル全体を上からたどり、各構文ノードを順番に見ます。
		ast.Inspect(file, func(n ast.Node) bool {
			// n には現在見ている AST ノードが入ります。
			// そのノードが for 文か、for range 文かを型で判定します。
			switch loop := n.(type) {
			case *ast.ForStmt:
				// 通常の for 文です。
				// 例: for i := 0; i < n; i++ { ... }
				reportLoopQueries(pass, loop.Body, state)
				return false
			case *ast.RangeStmt:
				// range を使う for 文です。
				// 例: for _, v := range items { ... }
				reportLoopQueries(pass, loop.Body, state)
				return false
			default:
				return true
			}
		})
	}
}

// reportLoopQueries はループ本体の中にある CallExpr を見て、
// 代表的な DB クエリ実行に当たるものを報告します。
func reportLoopQueries(pass *analysis.Pass, body *ast.BlockStmt, state *localCallState) {
	if body == nil {
		return
	}

	ast.Inspect(body, func(n ast.Node) bool {
		// 呼ばれていない無名関数の中身までは見ないようにします。
		// 例: fn := func() { db.First(...) } のようなケースはここでは未実行です。
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if !callExecutesQuery(pass, call, state) {
			return true
		}

		pass.Reportf(call.Pos(), "possible N+1 query inside loop")
		return true
	})
}
