package nplusonecheck

import (
	"go/ast"
	"go/token"
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
	if len(resolved) == 0 {
		return false
	}

	return state.declsContainQuery(pass, resolved)
}

// findDecl は CallExpr が current package や import 先 package の helper 関数や
// method なら、対応する宣言候補を返します。
func (s *callGraphState) findDecl(pkg *packageState, call *ast.CallExpr) []resolvedDecl {
	if pkg == nil || pkg.typesInfo == nil {
		return nil
	}

	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return singleResolvedDecl(s.resolveDecl(pkg.typesInfo.Uses[fun]))
	case *ast.SelectorExpr:
		if selection := pkg.typesInfo.Selections[fun]; selection != nil {
			return s.resolveSelectionDecls(pkg, fun, selection, call.Pos())
		}
		return singleResolvedDecl(s.resolveDecl(pkg.typesInfo.Uses[fun.Sel]))
	default:
		return nil
	}
}

func (s *callGraphState) resolveSelectionDecls(
	pkg *packageState,
	fun *ast.SelectorExpr,
	selection *types.Selection,
	callPos token.Pos,
) []resolvedDecl {
	if selection == nil {
		return nil
	}

	if unwrapInterface(selection.Recv()) == nil {
		return singleResolvedDecl(s.resolveDecl(selection.Obj()))
	}

	return s.resolveInterfaceMethodDecls(pkg, fun, callPos)
}

func (s *callGraphState) resolveInterfaceMethodDecls(
	pkg *packageState,
	fun *ast.SelectorExpr,
	callPos token.Pos,
) []resolvedDecl {
	if pkg == nil || pkg.typesInfo == nil || fun == nil || fun.Sel == nil {
		return nil
	}

	if resolved := s.resolveConcreteReceiverMethodDecl(pkg, fun.X, fun.Sel.Name, callPos); resolved != nil {
		return singleResolvedDecl(resolved)
	}

	iface := unwrapInterface(pkg.typesInfo.TypeOf(fun.X))
	if iface == nil {
		return nil
	}

	return s.resolveInterfaceMethodCandidates(pkg, iface, fun.Sel.Name)
}

func (s *callGraphState) resolveConcreteReceiverMethodDecl(
	pkg *packageState,
	expr ast.Expr,
	methodName string,
	callPos token.Pos,
) *resolvedDecl {
	concreteType := s.resolveConcreteType(pkg, expr, callPos)
	if concreteType == nil {
		return nil
	}

	method := lookupMethod(concreteType, methodName)
	if method == nil {
		return nil
	}

	return s.resolveDecl(method)
}

func (s *callGraphState) resolveConcreteType(pkg *packageState, expr ast.Expr, callPos token.Pos) types.Type {
	if pkg == nil || pkg.typesInfo == nil || expr == nil {
		return nil
	}

	if concreteType := concreteTypeOfExpr(pkg.typesInfo, expr); concreteType != nil {
		return concreteType
	}

	ident, ok := unwrapIdent(expr)
	if !ok {
		return nil
	}

	return s.resolveAssignedConcreteType(pkg, objectForIdent(pkg.typesInfo, ident), callPos)
}

func (s *callGraphState) resolveAssignedConcreteType(
	pkg *packageState,
	obj types.Object,
	callPos token.Pos,
) types.Type {
	if pkg == nil || pkg.typesInfo == nil || obj == nil {
		return nil
	}

	var (
		resolved  types.Type
		ambiguous bool
	)

	for _, file := range pkg.files {
		ast.Inspect(file, func(n ast.Node) bool {
			if ambiguous || n == nil {
				return !ambiguous
			}
			if n.Pos() >= callPos {
				return false
			}

			switch node := n.(type) {
			case *ast.ValueSpec:
				for idx, name := range node.Names {
					if objectForIdent(pkg.typesInfo, name) != obj {
						continue
					}

					recordResolvedType(pkg.typesInfo, &resolved, &ambiguous, indexedExpr(node.Values, idx))
				}
			case *ast.AssignStmt:
				for idx, lhs := range node.Lhs {
					ident, ok := lhs.(*ast.Ident)
					if !ok || objectForIdent(pkg.typesInfo, ident) != obj {
						continue
					}

					recordResolvedType(pkg.typesInfo, &resolved, &ambiguous, assignedExpr(node.Rhs, idx))
				}
			}

			return !ambiguous
		})
		if ambiguous {
			return nil
		}
	}

	return resolved
}

func recordResolvedType(info *types.Info, resolved *types.Type, ambiguous *bool, expr ast.Expr) {
	if ambiguous == nil || *ambiguous {
		return
	}

	typ := concreteTypeOfExpr(info, expr)
	if typ == nil {
		*ambiguous = true
		return
	}

	if resolved == nil || *resolved == nil {
		*resolved = typ
		return
	}

	if !types.Identical(*resolved, typ) {
		*ambiguous = true
	}
}

func (s *callGraphState) resolveInterfaceMethodCandidates(
	pkg *packageState,
	iface *types.Interface,
	methodName string,
) []resolvedDecl {
	if pkg == nil || iface == nil || methodName == "" {
		return nil
	}

	candidates := make([]resolvedDecl, 0)
	seen := make(map[string]struct{})
	for _, candidatePkg := range s.loader.searchPackages(pkg) {
		for _, fn := range candidatePkg.funcsByName[methodName] {
			sig, ok := fn.Type().(*types.Signature)
			if !ok || sig.Recv() == nil {
				continue
			}
			if !types.Implements(sig.Recv().Type(), iface) {
				continue
			}

			resolved := s.resolveDecl(fn)
			if resolved == nil {
				continue
			}

			if _, ok := seen[resolved.key]; ok {
				continue
			}

			seen[resolved.key] = struct{}{}
			candidates = append(candidates, *resolved)
		}
	}

	return candidates
}

func lookupMethod(typ types.Type, methodName string) *types.Func {
	methods := types.NewMethodSet(typ)
	for idx := 0; idx < methods.Len(); idx++ {
		obj, ok := methods.At(idx).Obj().(*types.Func)
		if ok && obj.Name() == methodName {
			return obj
		}
	}

	return nil
}

func unwrapIdent(expr ast.Expr) (*ast.Ident, bool) {
	switch node := expr.(type) {
	case *ast.Ident:
		return node, true
	case *ast.ParenExpr:
		return unwrapIdent(node.X)
	default:
		return nil, false
	}
}

func objectForIdent(info *types.Info, ident *ast.Ident) types.Object {
	if info == nil || ident == nil {
		return nil
	}

	if obj := info.Uses[ident]; obj != nil {
		return obj
	}

	return info.Defs[ident]
}

func indexedExpr(exprs []ast.Expr, idx int) ast.Expr {
	if idx < 0 || idx >= len(exprs) {
		return nil
	}

	return exprs[idx]
}

func assignedExpr(exprs []ast.Expr, idx int) ast.Expr {
	if len(exprs) == 1 {
		return exprs[0]
	}

	return indexedExpr(exprs, idx)
}

func singleResolvedDecl(resolved *resolvedDecl) []resolvedDecl {
	if resolved == nil {
		return nil
	}

	return []resolvedDecl{*resolved}
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

func (s *callGraphState) declsContainQuery(pass *analysis.Pass, resolved []resolvedDecl) bool {
	for idx := range resolved {
		if s.declContainsQuery(pass, &resolved[idx]) {
			return true
		}
	}

	return false
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
