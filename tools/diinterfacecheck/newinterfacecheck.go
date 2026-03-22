package diinterfacecheck

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

// NewXxx ルール用の plugin 本体。
// 今回は設定なしの最小構成に寄せる。
type newInterfacePlugin struct{}

// package import 時に plugin を登録する。
func init() {
	register.Plugin(newInterfacePluginName, newNewInterfacePlugin)
}

func newNewInterfacePlugin(_ any) (register.LinterPlugin, error) {
	return &newInterfacePlugin{}, nil
}

func (p *newInterfacePlugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}

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
				if !shouldReportNewParamType(typ, pass.Pkg.Path()) {
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

func isNewLikeFunction(name string) bool {
	if !strings.HasPrefix(name, "New") || len(name) == len("New") {
		return false
	}

	r, _ := utf8.DecodeRuneInString(name[len("New"):])
	return unicode.IsUpper(r)
}

// PoC なので interface と builtin は許容し、
// pointer を剥がした先が他 package の Named concrete だった場合だけ警告する。
func shouldReportNewParamType(typ types.Type, currentPkgPath string) bool {
	for typ != nil {
		typ = types.Unalias(typ)

		switch t := typ.(type) {
		case *types.Interface, *types.Basic:
			return false
		case *types.Pointer:
			typ = t.Elem()
		case *types.Named:
			if _, ok := t.Underlying().(*types.Interface); ok {
				return false
			}
			if t.Obj() == nil || t.Obj().Pkg() == nil {
				return false
			}
			if isAllowedConcreteParam(t) {
				return false
			}
			return t.Obj().Pkg().Path() != currentPkgPath
		default:
			return false
		}
	}

	return false
}

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
