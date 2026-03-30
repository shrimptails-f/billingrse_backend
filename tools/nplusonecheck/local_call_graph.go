package nplusonecheck

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// callGraphState は current package と import 先 package をまたいで、
// 「この関数を実行すると最終的にクエリが走るか」の判定結果を持ちます。
type callGraphState struct {
	loader   *packageLoader
	memo     map[string]bool
	visiting map[string]bool
}

type resolvedDecl struct {
	key  string
	pkg  *packageState
	decl *ast.FuncDecl
}

// newCallGraphState は current package を起点に call graph 追跡 state を作ります。
func newCallGraphState(pass *analysis.Pass) *callGraphState {
	return &callGraphState{
		loader:   newPackageLoader(pass),
		memo:     make(map[string]bool),
		visiting: make(map[string]bool),
	}
}

// callExecutesQuery は、この呼び出しを実行すると最終的に
// DB クエリまで到達するかを判定します。
func callExecutesQuery(pass *analysis.Pass, pkg *packageState, call *ast.CallExpr, state *callGraphState) bool {
	if pkg == nil || pkg.typesInfo == nil {
		return false
	}

	if isQueryCall(pkg.typesInfo, call) {
		return true
	}

	// 即時実行の無名関数は、その本体をこの場で実行するため再帰的に見ます。
	if lit, ok := call.Fun.(*ast.FuncLit); ok {
		return state.nodeContainsQuery(pass, pkg, lit.Body)
	}

	resolved := state.findDecl(pkg, call)
	if resolved == nil {
		return false
	}

	return state.declContainsQuery(pass, resolved)
}

// findDecl は CallExpr が current package や import 先 package の helper 関数や
// method なら、対応する宣言を返します。
func (s *callGraphState) findDecl(pkg *packageState, call *ast.CallExpr) *resolvedDecl {
	if pkg == nil || pkg.typesInfo == nil {
		return nil
	}

	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return s.resolveDecl(pkg.typesInfo.Uses[fun])
	case *ast.SelectorExpr:
		if selection := pkg.typesInfo.Selections[fun]; selection != nil {
			return s.resolveDecl(selection.Obj())
		}
		return s.resolveDecl(pkg.typesInfo.Uses[fun.Sel])
	default:
		return nil
	}
}

// resolveDecl は object が current package または import 先 package の関数なら、
// 対応する package 情報と宣言を返します。
func (s *callGraphState) resolveDecl(obj types.Object) *resolvedDecl {
	fn, ok := obj.(*types.Func)
	if !ok {
		return nil
	}

	key := funcKey(fn)
	if key == "" {
		return nil
	}

	pkg := s.loader.packageForObject(fn)
	if pkg == nil {
		return nil
	}

	decl := pkg.decls[key]
	if decl == nil {
		return nil
	}

	return &resolvedDecl{
		key:  key,
		pkg:  pkg,
		decl: decl,
	}
}

// declContainsQuery は helper 関数や method の本体を調べて、
// 最終的にクエリ実行へ到達するかを返します。
func (s *callGraphState) declContainsQuery(pass *analysis.Pass, resolved *resolvedDecl) bool {
	if resolved == nil || resolved.decl == nil || resolved.decl.Body == nil {
		return false
	}

	if result, ok := s.memo[resolved.key]; ok {
		return result
	}

	if s.visiting[resolved.key] {
		return false
	}

	s.visiting[resolved.key] = true
	result := s.nodeContainsQuery(pass, resolved.pkg, resolved.decl.Body)
	delete(s.visiting, resolved.key)
	s.memo[resolved.key] = result

	return result
}

// nodeContainsQuery はノード配下にある実行される呼び出しを調べて、
// クエリまで到達する経路があるかを返します。
func (s *callGraphState) nodeContainsQuery(pass *analysis.Pass, pkg *packageState, node ast.Node) bool {
	found := false

	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}

		// 呼ばれていない無名関数は実行経路に乗らないため、
		// その中身はここでは探索しません。
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if callExecutesQuery(pass, pkg, call, s) {
			found = true
			return false
		}

		return true
	})

	return found
}
