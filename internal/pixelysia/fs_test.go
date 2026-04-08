package pixelysia

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCopyFile(t *testing.T) {
	setupTestGlobals(t)

	root := t.TempDir()
	src := filepath.Join(root, "src.txt")
	dst := filepath.Join(root, "nested", "dst.txt")

	if err := os.WriteFile(src, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst, 0o644); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("unexpected file contents: %q", string(got))
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("unexpected dst mode: %o", info.Mode().Perm())
	}
}

func TestCopyDirCopiesNested(t *testing.T) {
	setupTestGlobals(t)

	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	if err := os.MkdirAll(filepath.Join(src, "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "child", "file.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(dst, "child", "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "ok" {
		t.Fatalf("unexpected copied data: %q", string(b))
	}
}

func TestCopyDirRejectsSymlink(t *testing.T) {
	setupTestGlobals(t)
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on windows")
	}

	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target", filepath.Join(src, "link")); err != nil {
		t.Fatal(err)
	}

	err := copyDir(src, dst)
	if err == nil {
		t.Fatal("expected copyDir to fail on symlink")
	}
	if !strings.Contains(err.Error(), "symlinks are not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReplaceDirAtomic(t *testing.T) {
	setupTestGlobals(t)

	root := t.TempDir()
	tmp := filepath.Join(root, "tmp")
	dst := filepath.Join(root, "dest")
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "new.txt"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := replaceDirAtomic(tmp, dst); err != nil {
		t.Fatalf("replaceDirAtomic failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dst, "new.txt")); err != nil {
		t.Fatalf("expected new file in destination: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected old file to be removed, got err=%v", err)
	}
}

func TestSetOwnershipAndModeRecursive(t *testing.T) {
	setupTestGlobals(t)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(root, "sub", "file.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := setOwnershipAndModeRecursive(root, os.Getuid(), os.Getgid(), 0o644, 0o755); err != nil {
		t.Fatalf("setOwnershipAndModeRecursive failed: %v", err)
	}

	dInfo, err := os.Stat(filepath.Join(root, "sub"))
	if err != nil {
		t.Fatal(err)
	}
	if dInfo.Mode().Perm() != 0o755 {
		t.Fatalf("unexpected directory mode: %o", dInfo.Mode().Perm())
	}

	fInfo, err := os.Stat(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if fInfo.Mode().Perm() != 0o644 {
		t.Fatalf("unexpected file mode: %o", fInfo.Mode().Perm())
	}
}

func TestThemePathValidation(t *testing.T) {
	setupTestGlobals(t)

	sddmThemesDir = filepath.Join(t.TempDir(), "themes")

	if _, err := themePath("valid-theme_1"); err != nil {
		t.Fatalf("expected valid theme name, got %v", err)
	}

	if _, err := themePath("../escape"); err == nil {
		t.Fatal("expected invalid theme path to fail")
	}
}
