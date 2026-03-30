package nplusonecheck

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// localCallState は同一 package 内の関数宣言と、
// 「この関数を実行すると最終的にクエリが走るか」の判定結果を持ちます。
type localCallState struct {
	decls    map[token.Pos]*ast.FuncDecl
	memo     map[token.Pos]bool
	visiting map[token.Pos]bool
}

// newLocalCallState は同一 package 内の top-level 関数と method を集めます。
func newLocalCallState(pass *analysis.Pass) *localCallState {
	state := &localCallState{
		decls:    make(map[token.Pos]*ast.FuncDecl),
		memo:     make(map[token.Pos]bool),
		visiting: make(map[token.Pos]bool),
	}

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil {
				continue
			}

			obj := pass.TypesInfo.Defs[fn.Name]
			if obj == nil {
				continue
			}

			state.decls[obj.Pos()] = fn
		}
	}

	return state
}

// callExecutesQuery は、この呼び出しを実行すると最終的に
// DB クエリまで到達するかを判定します。
func callExecutesQuery(pass *analysis.Pass, call *ast.CallExpr, state *localCallState) bool {
	if isQueryCall(pass, call) {
		return true
	}

	// 即時実行の無名関数は、その本体をこの場で実行するため再帰的に見ます。
	if lit, ok := call.Fun.(*ast.FuncLit); ok {
		return state.nodeContainsQuery(pass, lit.Body)
	}

	decl := state.findLocalDecl(pass, call)
	if decl == nil {
		return false
	}

	return state.declContainsQuery(pass, decl)
}

// findLocalDecl は CallExpr が同一 package の helper 関数や method なら、
// 対応する宣言を返します。
func (s *localCallState) findLocalDecl(pass *analysis.Pass, call *ast.CallExpr) *ast.FuncDecl {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return s.declByObject(pass, pass.TypesInfo.Uses[fun])
	case *ast.SelectorExpr:
		if selection := pass.TypesInfo.Selections[fun]; selection != nil {
			return s.declByObject(pass, selection.Obj())
		}
		return s.declByObject(pass, pass.TypesInfo.Uses[fun.Sel])
	default:
		return nil
	}
}

// declByObject は object が同一 package の関数なら、その宣言を返します。
func (s *localCallState) declByObject(pass *analysis.Pass, obj types.Object) *ast.FuncDecl {
	if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != pass.Pkg.Path() {
		return nil
	}

	fn, ok := obj.(*types.Func)
	if !ok {
		return nil
	}

	return s.decls[fn.Pos()]
}

// declContainsQuery は helper 関数や method の本体を調べて、
// 最終的にクエリ実行へ到達するかを返します。
func (s *localCallState) declContainsQuery(pass *analysis.Pass, decl *ast.FuncDecl) bool {
	if decl == nil || decl.Body == nil {
		return false
	}

	key := decl.Pos()

	if result, ok := s.memo[key]; ok {
		return result
	}

	if s.visiting[key] {
		return false
	}

	s.visiting[key] = true
	result := s.nodeContainsQuery(pass, decl.Body)
	delete(s.visiting, key)
	s.memo[key] = result

	return result
}

// nodeContainsQuery はノード配下にある実行される呼び出しを調べて、
// クエリまで到達する経路があるかを返します。
func (s *localCallState) nodeContainsQuery(pass *analysis.Pass, node ast.Node) bool {
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

		if callExecutesQuery(pass, call, s) {
			found = true
			return false
		}

		return true
	})

	return found
}
