package dicheck

import (
	"go/ast"
	"go/types"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

// newinterfacecheck は NewXxx 関数が interface ではなく
// 他 package の具体型を受け取っている場合に警告する custom linter です。

const newInterfacePluginName = "newinterfacecheck"

// allowedConcreteParams は例外的に許容する具体型。
// 現状は framework / ORM の中心型だけをホワイトリストで逃がす。
var allowedConcreteParams = []struct {
	packagePath string
	typeName    string
}{
	{
		packagePath: "gorm.io/gorm",
		typeName:    "DB",
	},
	{
		packagePath: "github.com/gin-gonic/gin",
		typeName:    "Engine",
	},
}

// allowedConcreteParamsByFunction は特定の NewXxx 関数だけに許容する具体型。
// 影響範囲を狭めるため、全体 whitelist ではなく関数単位で逃がす。
var allowedConcreteParamsByFunction = []struct {
	functionName     string
	functionPkgPath  string
	paramPackagePath string
	typeName         string
}{
	{
		functionName:     "NewBilling",
		functionPkgPath:  "business/internal/common/domain",
		paramPackagePath: "time",
		typeName:         "Time",
	},
}

// NewXxx ルール用の plugin 本体。
// 今回は設定なしの最小構成に寄せる。
type newInterfacePlugin struct{}

// package import 時に plugin を登録する。
func init() {
	register.Plugin(newInterfacePluginName, newNewInterfacePlugin)
}

// newNewInterfacePlugin は設定なしで動く最小 plugin を返す。
func newNewInterfacePlugin(_ any) (register.LinterPlugin, error) {
	return &newInterfacePlugin{}, nil
}

// 型情報が必要なので TypesInfo を要求する。
func (p *newInterfacePlugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}

// BuildAnalyzers は analyzer を 1 つだけ返す。
func (p *newInterfacePlugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{
		{
			Name: newInterfacePluginName,
			Doc:  "reports concrete cross-package parameters used by NewXxx functions",
			Run: func(pass *analysis.Pass) (any, error) {
				runNewInterfaceCheck(pass)
				return nil, nil
			},
		},
	}, nil
}

// runNewInterfaceCheck は package 内のトップレベル関数を走査し、
// NewXxx という名前の関数引数だけを検査する。
func runNewInterfaceCheck(pass *analysis.Pass) {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name == nil || fn.Type == nil || fn.Type.Params == nil {
				continue
			}

			if !isNewLikeFunction(fn.Name.Name) {
				continue
			}

			for _, field := range fn.Type.Params.List {
				typ := pass.TypesInfo.TypeOf(field.Type)
				if !shouldReportNewParamType(typ, pass.Pkg.Path(), fn.Name.Name) {
					continue
				}

				pass.Reportf(
					field.Type.Pos(),
					"%s should receive an interface instead of %s",
					fn.Name.Name,
					typ.String(),
				)
			}
		}
	}
}

// isNewLikeFunction は "New" のあとが大文字で始まる関数だけを対象にする。
// これで New 単体や newHelper のような関数は除外する。
func isNewLikeFunction(name string) bool {
	if !strings.HasPrefix(name, "New") || len(name) == len("New") {
		return false
	}

	r, _ := utf8.DecodeRuneInString(name[len("New"):])
	return unicode.IsUpper(r)
}

// shouldReportNewParamType は入口側の粗い判定。
// interface / builtin は許容し、pointer 型だけを詳細判定に回す。
func shouldReportNewParamType(typ types.Type, currentPkgPath, functionName string) bool {
	for typ != nil {
		typ = types.Unalias(typ)

		switch t := typ.(type) {
		case *types.Interface, *types.Basic:
			return false
		case *types.Pointer:
			return shouldReportPointerElem(t.Elem(), currentPkgPath, functionName)
		case *types.Named:
			return false
		default:
			return false
		}
	}

	return false
}

// shouldReportPointerElem は pointer を剥がした先が
// 「他 package の Named concrete」かどうかを判定する。
func shouldReportPointerElem(typ types.Type, currentPkgPath, functionName string) bool {
	for typ != nil {
		typ = types.Unalias(typ)

		switch t := typ.(type) {
		case *types.Pointer:
			typ = t.Elem()
		case *types.Named:
			if _, ok := t.Underlying().(*types.Interface); ok {
				return false
			}
			if t.Obj() == nil || t.Obj().Pkg() == nil {
				return false
			}
			if isAllowedConcreteParam(t) || isAllowedConcreteParamForFunction(t, currentPkgPath, functionName) {
				return false
			}
			return t.Obj().Pkg().Path() != currentPkgPath
		default:
			return false
		}
	}

	return false
}

// isAllowedConcreteParam は例外 whitelist に含まれる具体型かを返す。
func isAllowedConcreteParam(named *types.Named) bool {
	if named == nil || named.Obj() == nil || named.Obj().Pkg() == nil {
		return false
	}

	for _, allowed := range allowedConcreteParams {
		if named.Obj().Pkg().Path() == allowed.packagePath && named.Obj().Name() == allowed.typeName {
			return true
		}
	}

	return false
}

// isAllowedConcreteParamForFunction は関数単位の例外 whitelist に含まれる具体型かを返す。
func isAllowedConcreteParamForFunction(named *types.Named, currentPkgPath, functionName string) bool {
	if named == nil || named.Obj() == nil || named.Obj().Pkg() == nil {
		return false
	}

	for _, allowed := range allowedConcreteParamsByFunction {
		if functionName == allowed.functionName &&
			currentPkgPath == allowed.functionPkgPath &&
			named.Obj().Pkg().Path() == allowed.paramPackagePath &&
			named.Obj().Name() == allowed.typeName {
			return true
		}
	}

	return false
}
