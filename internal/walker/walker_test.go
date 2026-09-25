package walker

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// makeTree creates the given files (with parent dirs) under root.
func makeTree(t *testing.T, root string, files ...string) {
	t.Helper()
	for _, f := range files {
		p := filepath.Join(root, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func rel(t *testing.T, root string, paths []string) []string {
	t.Helper()
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		r, err := filepath.Rel(root, p)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, filepath.ToSlash(r))
	}
	slices.Sort(out)
	return out
}

func TestWalk_FiltersIgnoredHiddenAndExtensions(t *testing.T) {
	root := t.TempDir()
	makeTree(t, root,
		"main.go",
		"pkg/a.go",
		"pkg/a_test.go",
		"pkg/notes.md",
		"vendor/dep/dep.go",
		"node_modules/x/x.go",
		".git/hooks/h.go",
		".semcode/cache.go",
		".idea/ws.go",
		"pkg/.hidden.go",
	)

	files, err := Walk(root, []string{"go"}) // extension without a dot is accepted
	if err != nil {
		t.Fatal(err)
	}

	got := rel(t, root, files)
	want := []string{"main.go", "pkg/a.go", "pkg/a_test.go"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestWalk_NoExtensionFilterReturnsAllVisibleFiles(t *testing.T) {
	root := t.TempDir()
	makeTree(t, root, "a.go", "b.md", ".env")

	files, err := Walk(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := rel(t, root, files), []string{"a.go", "b.md"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestWalk_DotDotRootIsNotTreatedAsHidden(t *testing.T) {
	root := t.TempDir()
	makeTree(t, root, "a.go", "sub/b.go")

	// os.Chdir rather than t.Chdir: the module targets Go 1.23.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Join(root, "sub")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	files, err := Walk("..", []string{".go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("walking \"..\" should find 2 files, got %v", files)
	}
}

func TestWalk_HiddenRootIsWalked(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".hidden-repo")
	makeTree(t, root, "a.go")

	files, err := Walk(root, []string{".go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected the hidden root to be walked, got %v", files)
	}
}
