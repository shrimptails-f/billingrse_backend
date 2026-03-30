package nplusonecheck

import (
	"go/ast"
	"go/types"
)

// queryTarget は「どの型の、どのメソッド呼び出しを
// クエリ実行として扱うか」を表します。
type queryTarget struct {
	packagePath string
	typeName    string
	methods     map[string]struct{}
}

// queryTargets は PoC で監視する最小限の対象です。
// current package と同じ workspace 配下の helper 関数や method は再帰的にたどり、
// interface method はローカル concrete 解決または current package/direct import の
// 候補列挙で追跡します。
var queryTargets = []queryTarget{
	{
		packagePath: "database/sql",
		typeName:    "DB",
		methods: newMethodSet(
			"Exec",
			"ExecContext",
			"Prepare",
			"PrepareContext",
			"Query",
			"QueryContext",
			"QueryRow",
			"QueryRowContext",
		),
	},
	{
		packagePath: "database/sql",
		typeName:    "Tx",
		methods: newMethodSet(
			"Exec",
			"ExecContext",
			"Prepare",
			"PrepareContext",
			"Query",
			"QueryContext",
			"QueryRow",
			"QueryRowContext",
		),
	},
	{
		packagePath: "github.com/jmoiron/sqlx",
		typeName:    "DB",
		methods: newMethodSet(
			"Exec",
			"ExecContext",
			"Get",
			"NamedExec",
			"NamedQuery",
			"Preparex",
			"PreparexContext",
			"QueryRowx",
			"QueryRowxContext",
			"Queryx",
			"QueryxContext",
			"Select",
		),
	},
	{
		packagePath: "github.com/jmoiron/sqlx",
		typeName:    "Tx",
		methods: newMethodSet(
			"Exec",
			"ExecContext",
			"Get",
			"NamedExec",
			"NamedQuery",
			"Preparex",
			"PreparexContext",
			"QueryRowx",
			"QueryRowxContext",
			"Queryx",
			"QueryxContext",
			"Select",
		),
	},
	{
		packagePath: "gorm.io/gorm",
		typeName:    "DB",
		methods: newMethodSet(
			"Count",
			"Find",
			"First",
			"Last",
			"Pluck",
			"Row",
			"Rows",
			"Scan",
			"Take",
		),
	},
}

// isQueryCall は selector 呼び出しの receiver 型を解決し、
// 登録済みの DB 型とメソッド名に一致するかを判定します。
func isQueryCall(typesInfo *types.Info, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return false
	}

	named := unwrapNamedOrPointer(typesInfo.TypeOf(sel.X))
	if named == nil || named.Obj() == nil || named.Obj().Pkg() == nil {
		return false
	}

	for _, target := range queryTargets {
		if named.Obj().Pkg().Path() != target.packagePath || named.Obj().Name() != target.typeName {
			continue
		}

		_, ok := target.methods[sel.Sel.Name]
		return ok
	}

	return false
}

// unwrapNamedOrPointer は alias と pointer を剥がして
// 最終的な Named 型を取り出します。
func unwrapNamedOrPointer(typ types.Type) *types.Named {
	for typ != nil {
		typ = types.Unalias(typ)

		switch t := typ.(type) {
		case *types.Pointer:
			typ = t.Elem()
		case *types.Named:
			return t
		default:
			return nil
		}
	}

	return nil
}

func unwrapInterface(typ types.Type) *types.Interface {
	typ = types.Unalias(typ)
	if typ == nil {
		return nil
	}

	iface, ok := typ.Underlying().(*types.Interface)
	if !ok {
		return nil
	}

	return iface
}

func concreteTypeOfExpr(typesInfo *types.Info, expr ast.Expr) types.Type {
	if typesInfo == nil || expr == nil {
		return nil
	}

	switch node := expr.(type) {
	case *ast.ParenExpr:
		return concreteTypeOfExpr(typesInfo, node.X)
	case *ast.CallExpr:
		if len(node.Args) == 1 && unwrapInterface(typesInfo.TypeOf(node.Fun)) != nil {
			return concreteTypeOfExpr(typesInfo, node.Args[0])
		}
	}

	typ := typesInfo.TypeOf(expr)
	if typ == nil || unwrapInterface(typ) != nil {
		return nil
	}

	return types.Unalias(typ)
}

// newMethodSet は対象メソッド名を高速に照合するための set を作ります。
func newMethodSet(names ...string) map[string]struct{} {
	methods := make(map[string]struct{}, len(names))
	for _, name := range names {
		methods[name] = struct{}{}
	}
	return methods
}
