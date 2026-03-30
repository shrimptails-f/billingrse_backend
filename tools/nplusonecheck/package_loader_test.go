package nplusonecheck

import (
	"path/filepath"
	"testing"
)

func TestPackageLoaderResolvePackageDir_SkipsOutsideWorkspace(t *testing.T) {
	t.Parallel()

	loader := &packageLoader{
		workspaceRoot: filepath.Join(string(filepath.Separator), "workspace", "project"),
		modulePrefix:  "example.com/project",
	}

	got := loader.resolvePackageDir("github.com/external/helper")
	if got != "" {
		t.Fatalf("resolvePackageDir() should skip external imports; got %q", got)
	}
}

func TestIsWithinWorkspace_ReturnsFalseForOutsideFiles(t *testing.T) {
	t.Parallel()

	workspaceRoot := filepath.Join(string(filepath.Separator), "workspace", "project")
	files := []string{
		filepath.Join(string(filepath.Separator), "workspace", "other", "pkg", "helper.go"),
	}

	if isWithinWorkspace(files, workspaceRoot) {
		t.Fatal("isWithinWorkspace() = true, want false for files outside workspace root")
	}
}
