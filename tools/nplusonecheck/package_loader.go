package nplusonecheck

import (
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

// packageState は 1 package 分の型情報と、関数宣言 index を持ちます。
type packageState struct {
	path      string
	typesInfo *types.Info
	decls     map[string]*ast.FuncDecl
}

// packageLoader は current package と import 先 package の AST / 型情報を管理します。
// 無制限に外部依存へ潜ると重くなるため、current package と同じ workspace 配下だけを読みます。
type packageLoader struct {
	currentPkgPath string
	gopathRoot     string
	modulePrefix   string
	workspaceRoot  string
	packages       map[string]*packageState
}

// newPackageLoader は current package を初期状態として loader を作ります。
func newPackageLoader(pass *analysis.Pass) *packageLoader {
	current := buildPackageState(pass.Pkg.Path(), pass.TypesInfo, pass.Files)
	workspaceRoot := detectWorkspaceRoot(pass.Fset, pass.Files, current.path)
	currentDir := firstFileDir(pass.Fset, pass.Files)
	gopathRoot := detectGoPathRoot(pass.Fset, pass.Files, current.path)

	return &packageLoader{
		currentPkgPath: current.path,
		gopathRoot:     gopathRoot,
		modulePrefix:   deriveModulePrefix(current.path, currentDir, workspaceRoot),
		workspaceRoot:  workspaceRoot,
		packages: map[string]*packageState{
			current.path: current,
		},
	}
}

// currentPackage は今まさに解析対象になっている package の情報を返します。
func (l *packageLoader) currentPackage() *packageState {
	return l.packages[l.currentPkgPath]
}

// packageForObject は object が属する package の情報を返します。
// current package なら既存 state を返し、import 先なら必要に応じて load します。
func (l *packageLoader) packageForObject(obj types.Object) *packageState {
	if obj == nil || obj.Pkg() == nil {
		return nil
	}

	if pkg, ok := l.packages[obj.Pkg().Path()]; ok {
		return pkg
	}

	return l.loadPackage(obj.Pkg().Path())
}

// loadPackage は import path から package を source 付きで読み込みます。
// current workspace の外にある package は対象外として捨てます。
func (l *packageLoader) loadPackage(importPath string) *packageState {
	if importPath == "" {
		return nil
	}

	if pkg, ok := l.packages[importPath]; ok {
		return pkg
	}

	dir := l.resolvePackageDir(importPath)
	if dir == "" {
		l.packages[importPath] = nil
		return nil
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo,
		Dir: dir,
	}
	if l.gopathRoot != "" {
		cfg.Env = append(os.Environ(),
			"GO111MODULE=off",
			"GOPATH="+l.gopathRoot,
		)
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		l.packages[importPath] = nil
		return nil
	}

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			continue
		}

		files := pkg.GoFiles
		if len(files) == 0 {
			files = pkg.CompiledGoFiles
		}
		if !isWithinWorkspace(files, l.workspaceRoot) {
			l.packages[importPath] = nil
			return nil
		}

		state := buildPackageState(pkg.PkgPath, pkg.TypesInfo, pkg.Syntax)
		l.packages[importPath] = state
		return state
	}

	l.packages[importPath] = nil
	return nil
}

// buildPackageState は package の関数宣言 index を組み立てます。
func buildPackageState(path string, typesInfo *types.Info, files []*ast.File) *packageState {
	state := &packageState{
		path:      path,
		typesInfo: typesInfo,
		decls:     make(map[string]*ast.FuncDecl),
	}

	if typesInfo == nil {
		return state
	}

	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil {
				continue
			}

			obj, ok := typesInfo.Defs[fn.Name].(*types.Func)
			if !ok {
				continue
			}

			key := funcKey(obj)
			if key == "" {
				continue
			}

			state.decls[key] = fn
		}
	}

	return state
}

// detectWorkspaceRoot は current package の source file から
// module root または GOPATH の src root を推定します。
func detectWorkspaceRoot(fset *token.FileSet, files []*ast.File, currentPkgPath string) string {
	if fset == nil || len(files) == 0 {
		return ""
	}

	filename := fset.PositionFor(files[0].Pos(), false).Filename
	if filename == "" {
		return ""
	}

	dir := filepath.Dir(filename)
	if root := findGoPathSrcRoot(filename); root != "" {
		rel, err := filepath.Rel(root, dir)
		if err == nil {
			rel = filepath.ToSlash(rel)
			if rel == currentPkgPath || strings.HasSuffix(currentPkgPath, "/"+rel) {
				return root
			}
		}
	}

	if root := findGoModRoot(dir); root != "" {
		return root
	}

	if root := findGoPathSrcRoot(filename); root != "" {
		return root
	}

	return dir
}

func detectGoPathRoot(fset *token.FileSet, files []*ast.File, currentPkgPath string) string {
	workspaceRoot := detectWorkspaceRoot(fset, files, currentPkgPath)
	if workspaceRoot == "" || filepath.Base(workspaceRoot) != "src" {
		return ""
	}

	return filepath.Dir(workspaceRoot)
}

func firstFileDir(fset *token.FileSet, files []*ast.File) string {
	if fset == nil || len(files) == 0 {
		return ""
	}

	filename := fset.PositionFor(files[0].Pos(), false).Filename
	if filename == "" {
		return ""
	}

	return filepath.Dir(filename)
}

func findGoModRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func findGoPathSrcRoot(filename string) string {
	cleaned := filepath.Clean(filename)
	srcSegment := string(os.PathSeparator) + "src" + string(os.PathSeparator)
	idx := strings.LastIndex(cleaned, srcSegment)
	if idx == -1 {
		return ""
	}

	return cleaned[:idx+len(srcSegment)-1]
}

func isWithinWorkspace(files []string, workspaceRoot string) bool {
	if workspaceRoot == "" {
		return false
	}

	for _, file := range files {
		if file == "" {
			continue
		}

		rel, err := filepath.Rel(workspaceRoot, file)
		if err != nil {
			continue
		}
		if rel == "." || (!strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != "..") {
			return true
		}
	}

	return false
}

func (l *packageLoader) resolvePackageDir(importPath string) string {
	if l.workspaceRoot == "" {
		return ""
	}

	if l.modulePrefix == "" {
		return filepath.Join(l.workspaceRoot, filepath.FromSlash(importPath))
	}

	if importPath == l.modulePrefix {
		return l.workspaceRoot
	}

	if strings.HasPrefix(importPath, l.modulePrefix+"/") {
		rel := strings.TrimPrefix(importPath, l.modulePrefix+"/")
		return filepath.Join(l.workspaceRoot, filepath.FromSlash(rel))
	}

	return ""
}

// funcKey は関数または method を package path と receiver で一意化します。
func funcKey(fn *types.Func) string {
	if fn == nil || fn.Pkg() == nil {
		return ""
	}

	sig, _ := fn.Type().(*types.Signature)
	if sig == nil || sig.Recv() == nil {
		return fn.Pkg().Path() + "::" + fn.Name()
	}

	return fn.Pkg().Path() + "::" + normalizeTypeString(sig.Recv().Type()) + "::" + fn.Name()
}

func normalizeTypeString(typ types.Type) string {
	return types.TypeString(types.Unalias(typ), func(pkg *types.Package) string {
		if pkg == nil {
			return ""
		}
		return pkg.Path()
	})
}

func deriveModulePrefix(currentPkgPath, currentDir, workspaceRoot string) string {
	if currentPkgPath == "" || currentDir == "" || workspaceRoot == "" {
		return ""
	}

	rel, err := filepath.Rel(workspaceRoot, currentDir)
	if err != nil {
		return ""
	}

	rel = filepath.ToSlash(rel)
	if rel == "." {
		return currentPkgPath
	}

	if rel == currentPkgPath {
		return ""
	}

	if strings.HasSuffix(currentPkgPath, "/"+rel) {
		return strings.TrimSuffix(currentPkgPath, "/"+rel)
	}

	return ""
}
