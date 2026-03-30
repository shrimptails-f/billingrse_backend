package nplusonecheck

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

// nplusonecheck は for / range ループ内で
// 実行される代表的な DB クエリを検知する最小 custom linter です。

const pluginName = "nplusonecheck"

type plugin struct{}

// golangci-lint custom で package が読み込まれたときに plugin を登録します。
func init() {
	register.Plugin(pluginName, newPlugin)
}

// newPlugin は設定なしで動く最小の plugin 本体を返します。
func newPlugin(_ any) (register.LinterPlugin, error) {
	return &plugin{}, nil
}

// 型情報が必要なので TypesInfo を要求します。
func (p *plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}

// BuildAnalyzers は analyzer を 1 つだけ返します。
func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{newAnalyzer()}, nil
}

// newAnalyzer はループ内クエリ検知用の analyzer を組み立てます。
func newAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: pluginName,
		Doc:  "reports likely N+1 queries executed directly inside loops",
		Run: func(pass *analysis.Pass) (any, error) {
			run(pass)
			return nil, nil
		},
	}
}
