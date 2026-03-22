package dicheck

import (
	"go/ast"
	"go/types"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

// diinterfacecheck は dig.Provide(func(...)) の引数に
// interface を受けている場合、具象を受けるように警告する custom linter です。

const pluginName = "diinterfacecheck"

// config は linters.settings.custom.diinterfacecheck.settings から読み込む設定。
// PoC 段階なので、対象型を列挙する最小構成に寄せている。
type config struct {
	Targets []target `json:"targets"`
}

// target は dig.Provide の引数で避けたい interface 名と、
// 代わりに使ってほしい具体型名の対応を表す。
type target struct {
	Package   string `json:"package"`
	Type      string `json:"type"`
	Interface string `json:"interface"`
}

// defaultTargets は .golangci.yml 側で個別設定を書かなくても
// PoC がすぐ動くようにするためのデフォルト値。
var defaultTargets = []target{
	{
		Package:   "business/internal/library/gmail",
		Type:      "Client",
		Interface: "ClientInterface",
	},
	{
		Package:   "business/internal/library/gmailService",
		Type:      "Client",
		Interface: "ClientInterface",
	},
	{
		Package:   "business/internal/library/logger",
		Type:      "Logger",
		Interface: "Interface",
	},
	{
		Package:   "business/internal/library/openai",
		Type:      "Client",
		Interface: "UseCaserInterface",
	},
	{
		Package:   "business/internal/library/oswrapper",
		Type:      "OsWrapper",
		Interface: "OsWapperInterface",
	},
	{
		Package:   "business/internal/library/redis",
		Type:      "Client",
		Interface: "ClientInterface",
	},
	{
		Package:   "business/internal/library/secret",
		Type:      "secretClient",
		Interface: "Client",
	},
	{
		Package:   "business/internal/library/sendMailer",
		Type:      "SmtpClient",
		Interface: "Client",
	},
	{
		Package:   "business/internal/library/timewrapper",
		Type:      "Clock",
		Interface: "ClockInterface",
	},
}

// AST だけでは pointer や alias をまたいだ先の実型までは判定できないため、
// TypesInfo を要求して型情報付きで解析する。
func (p *plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}

// golangci-lint customで package が import された副作用で init() が走る
func init() {
	register.Plugin(pluginName, newPlugin)
}

type plugin struct {
	cfg config
}

// newPlugin は golangci-lint の custom settings を decode し、
// 明示設定がない場合は defaultTargets を補う。
func newPlugin(raw any) (register.LinterPlugin, error) {
	cfg := config{}
	if raw != nil {
		decoded, err := register.DecodeSettings[config](raw)
		if err != nil {
			return nil, err
		}
		cfg = decoded
	}

	cfg.Targets = defaultTargets

	return &plugin{cfg: cfg}, nil
}

// BuildAnalyzers は analyzer を 1 つだけ返す。
func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{
		{
			Name: pluginName,
			Doc:  "reports interface types used in dig.Provide parameters where a concrete type should be injected",
			Run: func(pass *analysis.Pass) (any, error) {
				run(pass, p.cfg.Targets)
				return nil, nil
			},
		},
	}, nil
}

// run は package 内の各ファイルを走査し、
// 第 1 引数に無名関数を渡している "*.Provide(func("の形 を探す。
func run(pass *analysis.Pass, targets []target) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil || sel.Sel.Name != "Provide" {
				return true
			}

			if len(call.Args) == 0 {
				return true
			}

			// 今回の PoC では以下のような一般的なパターンだけを対象にする。
			// container.Provide(func(dep1, dep2, ...) *Concrete { ... })
			fn, ok := call.Args[0].(*ast.FuncLit)
			if !ok || fn.Type == nil || fn.Type.Params == nil {
				return true
			}

			for _, field := range fn.Type.Params.List {
				reportIfInterface(pass, field.Type, targets)
			}

			return true
		})
	}
}

// reportIfInterface は引数の型を解決し、
// alias を剥がしたうえで target の interface と一致した場合に警告を出す。
func reportIfInterface(pass *analysis.Pass, expr ast.Expr, targets []target) {
	typ := pass.TypesInfo.TypeOf(expr)
	named := unwrapNamed(typ)
	if named == nil || named.Obj() == nil || named.Obj().Pkg() == nil {
		return
	}

	for _, target := range targets {
		if named.Obj().Pkg().Path() != target.Package || named.Obj().Name() != target.Interface {
			continue
		}

		pass.Reportf(
			expr.Pos(),
			"use *%s.%s instead of %s in dig.Provide parameters",
			named.Obj().Pkg().Name(),
			target.Type,
			typ.String(),
		)
	}
}

// unwrapNamed は alias を剥がし、Named 型ならそのまま返す。
// dig.Provide の引数は interface のことが多いため、pointer はここでは剥がさない。
func unwrapNamed(typ types.Type) *types.Named {
	for typ != nil {
		typ = types.Unalias(typ)

		switch t := typ.(type) {
		case *types.Named:
			return t
		default:
			return nil
		}
	}

	return nil
}
