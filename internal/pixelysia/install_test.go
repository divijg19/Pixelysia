package pixelysia

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallRejectsMutuallyExclusiveOptions(t *testing.T) {
	setupTestGlobals(t)

	err := Install(InstallOptions{Split: true, Theme: "alpha"}, io.Discard)
	if err == nil {
		t.Fatal("expected install to fail for mutually exclusive options")
	}
	if !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallFullMode(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createSourceTree(t, []string{"alpha", "beta"})
	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	fontDir = filepath.Join(tmpRoot, "fonts")

	if err := os.Setenv(sourceDirEnv, srcRoot); err != nil {
		t.Fatal(err)
	}

	var calls [][]string
	commandRunner = func(name string, args ...string) *exec.Cmd {
		calls = append(calls, append([]string{name}, args...))
		return exec.Command("true")
	}

	if err := Install(InstallOptions{}, io.Discard); err != nil {
		t.Fatalf("Install full mode failed: %v", err)
	}

	fullRoot := filepath.Join(sddmThemesDir, fullThemeName)
	mustExistFile(t, filepath.Join(fullRoot, "Main.qml"))
	mustExistFile(t, filepath.Join(fullRoot, "themes", "alpha", "Main.qml"))
	mustExistFile(t, filepath.Join(fullRoot, "fonts", "TestFont.ttf"))
	mustExistFile(t, filepath.Join(fontDir, "TestFont.ttf"))

	if len(calls) == 0 || calls[0][0] != "fc-cache" || len(calls[0]) != 2 || calls[0][1] != "-f" {
		t.Fatalf("expected fc-cache -f call, got %#v", calls)
	}
}

func TestInstallSplitMode(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createSourceTree(t, []string{"alpha", "beta"})
	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	fontDir = filepath.Join(tmpRoot, "fonts")

	if err := os.Setenv(sourceDirEnv, srcRoot); err != nil {
		t.Fatal(err)
	}
	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	if err := Install(InstallOptions{Split: true}, io.Discard); err != nil {
		t.Fatalf("Install split mode failed: %v", err)
	}

	mustExistFile(t, filepath.Join(sddmThemesDir, "alpha", "Main.qml"))
	mustExistFile(t, filepath.Join(sddmThemesDir, "beta", "Main.qml"))
	if _, err := os.Stat(filepath.Join(sddmThemesDir, fullThemeName)); !os.IsNotExist(err) {
		t.Fatalf("did not expect full theme directory, err=%v", err)
	}
}

func TestInstallSingleThemeMode(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createSourceTree(t, []string{"alpha", "beta"})
	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	fontDir = filepath.Join(tmpRoot, "fonts")

	if err := os.Setenv(sourceDirEnv, srcRoot); err != nil {
		t.Fatal(err)
	}
	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	if err := Install(InstallOptions{Theme: "beta"}, io.Discard); err != nil {
		t.Fatalf("Install single theme failed: %v", err)
	}

	mustExistFile(t, filepath.Join(sddmThemesDir, "beta", "Main.qml"))
	if _, err := os.Stat(filepath.Join(sddmThemesDir, "alpha")); !os.IsNotExist(err) {
		t.Fatalf("did not expect alpha theme, err=%v", err)
	}
}

func TestInstallFailsWhenThemeMissingMainQML(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createSourceTree(t, []string{"broken"})
	if err := os.Remove(filepath.Join(srcRoot, "themes", "broken", "Main.qml")); err != nil {
		t.Fatal(err)
	}
	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	fontDir = filepath.Join(tmpRoot, "fonts")
	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	if err := os.Setenv(sourceDirEnv, srcRoot); err != nil {
		t.Fatal(err)
	}

	err := Install(InstallOptions{Theme: "broken"}, io.Discard)
	if err == nil {
		t.Fatal("expected install to fail for invalid theme source")
	}
	if !strings.Contains(err.Error(), "Main.qml") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallReportsFontCacheFailure(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createSourceTree(t, []string{"alpha"})
	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	fontDir = filepath.Join(tmpRoot, "fonts")

	if err := os.Setenv(sourceDirEnv, srcRoot); err != nil {
		t.Fatal(err)
	}

	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	err := Install(InstallOptions{}, io.Discard)
	if err == nil {
		t.Fatal("expected install to fail when fc-cache command fails")
	}
	if !strings.Contains(err.Error(), "rebuild font cache failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallAndListIntegration(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createSourceTree(t, []string{"zeta", "alpha"})
	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	fontDir = filepath.Join(tmpRoot, "fonts")

	if err := os.Setenv(sourceDirEnv, srcRoot); err != nil {
		t.Fatal(err)
	}
	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	if err := Install(InstallOptions{Split: true}, io.Discard); err != nil {
		t.Fatalf("split install failed: %v", err)
	}

	var out bytes.Buffer
	if err := ListThemes(&out); err != nil {
		t.Fatalf("ListThemes failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 themes, got %d (%q)", len(lines), out.String())
	}
	if lines[0] != "alpha" || lines[1] != "zeta" {
		t.Fatalf("expected sorted themes, got %v", lines)
	}
}

func TestInstallIdempotentReinstall(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createSourceTree(t, []string{"alpha", "beta"})
	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	fontDir = filepath.Join(tmpRoot, "fonts")

	if err := os.Setenv(sourceDirEnv, srcRoot); err != nil {
		t.Fatal(err)
	}
	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	if err := Install(InstallOptions{Split: true}, io.Discard); err != nil {
		t.Fatalf("first install failed: %v", err)
	}
	firstCount, err := countTreeEntries(sddmThemesDir)
	if err != nil {
		t.Fatal(err)
	}

	if err := Install(InstallOptions{Split: true}, io.Discard); err != nil {
		t.Fatalf("second install failed: %v", err)
	}
	secondCount, err := countTreeEntries(sddmThemesDir)
	if err != nil {
		t.Fatal(err)
	}

	if firstCount != secondCount {
		t.Fatalf("expected stable entry count after reinstall, got %d vs %d", firstCount, secondCount)
	}
}

func TestInstallFailsForInvalidSourceEnv(t *testing.T) {
	setupTestGlobals(t)

	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	fontDir = filepath.Join(tmpRoot, "fonts")

	if err := os.Setenv(sourceDirEnv, filepath.Join(tmpRoot, "missing")); err != nil {
		t.Fatal(err)
	}

	err := Install(InstallOptions{}, io.Discard)
	if err == nil {
		t.Fatal("expected install to fail when source env is invalid")
	}
	if !strings.Contains(err.Error(), sourceDirEnv) {
		t.Fatalf("expected error to mention source env var, got %v", err)
	}
}

func TestInstallSingleThemeMissing(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createSourceTree(t, []string{"alpha"})
	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	fontDir = filepath.Join(tmpRoot, "fonts")
	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	if err := os.Setenv(sourceDirEnv, srcRoot); err != nil {
		t.Fatal(err)
	}

	err := Install(InstallOptions{Theme: "missing"}, io.Discard)
	if err == nil {
		t.Fatal("expected single theme install to fail for missing theme")
	}
}

func TestInstallFailurePreservesExistingDestination(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createSourceTree(t, []string{"alpha"})
	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	fontDir = filepath.Join(tmpRoot, "fonts")
	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	if err := os.Setenv(sourceDirEnv, srcRoot); err != nil {
		t.Fatal(err)
	}

	if err := Install(InstallOptions{}, io.Discard); err != nil {
		t.Fatalf("initial install failed: %v", err)
	}

	marker := filepath.Join(sddmThemesDir, fullThemeName, "marker.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(filepath.Join(srcRoot, "themes", "alpha", "Main.qml")); err != nil {
		t.Fatal(err)
	}

	err := Install(InstallOptions{}, io.Discard)
	if err == nil {
		t.Fatal("expected reinstall to fail with broken source theme")
	}

	b, readErr := os.ReadFile(marker)
	if readErr != nil {
		t.Fatalf("expected previous installation to remain intact, got %v", readErr)
	}
	if string(b) != "keep" {
		t.Fatalf("unexpected marker contents: %q", string(b))
	}
}

func TestInstallPermissionFailureIsExplicit(t *testing.T) {
	setupTestGlobals(t)
	if runtime.GOOS != "linux" {
		t.Skip("permission simulation is linux-specific")
	}

	srcRoot := createSourceTree(t, []string{"alpha"})
	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	fontDir = filepath.Join(tmpRoot, "fonts")
	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	if err := os.Setenv(sourceDirEnv, srcRoot); err != nil {
		t.Fatal(err)
	}

	requiredUID = os.Getuid() + 10000
	err := Install(InstallOptions{}, io.Discard)
	if err == nil {
		t.Fatal("expected permission-related failure")
	}
	if !strings.Contains(err.Error(), "ownership") && !strings.Contains(err.Error(), "operation not permitted") {
		t.Fatalf("expected actionable permission error, got %v", err)
	}
}

func createSourceTree(t *testing.T, themes []string) string {
	t.Helper()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Main.qml"), []byte("import QtQuick 2.15"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "metadata.desktop"), []byte("[SddmGreeterTheme]"), 0o644); err != nil {
		t.Fatal(err)
	}

	fontsDir := filepath.Join(root, "fonts")
	if err := os.MkdirAll(fontsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fontsDir, "TestFont.ttf"), []byte("font"), 0o644); err != nil {
		t.Fatal(err)
	}

	themesDir := filepath.Join(root, "themes")
	if err := os.MkdirAll(themesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range themes {
		themeDir := filepath.Join(themesDir, name)
		if err := os.MkdirAll(themeDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(themeDir, "Main.qml"), []byte("import QtQuick 2.15"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func mustExistFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected file %s to exist: %v", path, err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("expected %s to be a file", path)
	}
}

func countTreeEntries(root string) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(_ string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}
