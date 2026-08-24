package pixelysia

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
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

func mustNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to not exist, err=%v", path, err)
	}
}

// createNestedSourceTree builds a source tree that mirrors the real
// repository hierarchy: flat top-level themes plus a "tui" category
// directory containing nested themes, plus an inert asset directory.
func createNestedSourceTree(t *testing.T) string {
	t.Helper()

	root := createSourceTree(t, []string{"forest"})
	themesDir := filepath.Join(root, "themes")

	for _, name := range []string{"Amber", "Emerald"} {
		dir := filepath.Join(themesDir, "tui", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Main.qml"), []byte("import QtQuick"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// A container directory without any theme inside must be ignored.
	if err := os.MkdirAll(filepath.Join(themesDir, "extra", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestDiscoverSourceThemesNested(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createNestedSourceTree(t)

	names, err := discoverSourceThemes(srcRoot)
	if err != nil {
		t.Fatalf("discoverSourceThemes failed: %v", err)
	}

	expected := []string{"forest", "tui/Amber", "tui/Emerald"}
	if strings.Join(names, ",") != strings.Join(expected, ",") {
		t.Fatalf("expected themes %v, got %v", expected, names)
	}
}

func TestDiscoverRejectsMalformedThemeDirectory(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createSourceTree(t, []string{"alpha"})
	broken := filepath.Join(srcRoot, "themes", "broken")
	if err := os.MkdirAll(broken, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(broken, "metadata.desktop"), []byte("[SddmGreeterTheme]"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := discoverSourceThemes(srcRoot); err == nil {
		t.Fatal("expected discovery to fail for metadata.desktop without Main.qml")
	} else if !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallFullModeWithNestedThemes(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createNestedSourceTree(t)
	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	fontDir = filepath.Join(tmpRoot, "fonts")
	if err := os.Setenv(sourceDirEnv, srcRoot); err != nil {
		t.Fatal(err)
	}
	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	if err := Install(InstallOptions{}, io.Discard); err != nil {
		t.Fatalf("full install failed: %v", err)
	}

	fullRoot := filepath.Join(sddmThemesDir, fullThemeName)
	mustExistFile(t, filepath.Join(fullRoot, "Main.qml"))
	mustExistFile(t, filepath.Join(fullRoot, "themes", "forest", "Main.qml"))
	mustExistFile(t, filepath.Join(fullRoot, "themes", "tui", "Amber", "Main.qml"))
	mustExistFile(t, filepath.Join(fullRoot, "themes", "tui", "Emerald", "Main.qml"))
}

func TestInstallSplitModeWithNestedThemes(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createNestedSourceTree(t)
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

	mustExistFile(t, filepath.Join(sddmThemesDir, "forest", "Main.qml"))
	mustExistFile(t, filepath.Join(sddmThemesDir, "tui", "Amber", "Main.qml"))
	mustExistFile(t, filepath.Join(sddmThemesDir, "tui", "Emerald", "Main.qml"))

	// The category directory itself must not be installed as a theme.
	mustNotExist(t, filepath.Join(sddmThemesDir, "tui", "Main.qml"))
	mustNotExist(t, filepath.Join(sddmThemesDir, "tui", "metadata.desktop"))
}

func TestInstallSingleNestedTheme(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createNestedSourceTree(t)
	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	fontDir = filepath.Join(tmpRoot, "fonts")
	if err := os.Setenv(sourceDirEnv, srcRoot); err != nil {
		t.Fatal(err)
	}
	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	if err := Install(InstallOptions{Theme: "tui/Amber"}, io.Discard); err != nil {
		t.Fatalf("single nested theme install failed: %v", err)
	}

	mustExistFile(t, filepath.Join(sddmThemesDir, "tui", "Amber", "Main.qml"))
	mustNotExist(t, filepath.Join(sddmThemesDir, "tui", "Emerald"))
	mustNotExist(t, filepath.Join(sddmThemesDir, "forest"))
}

func TestInstallRejectsCategoryAsTheme(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createNestedSourceTree(t)
	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	fontDir = filepath.Join(tmpRoot, "fonts")
	if err := os.Setenv(sourceDirEnv, srcRoot); err != nil {
		t.Fatal(err)
	}
	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	err := Install(InstallOptions{Theme: "tui"}, io.Discard)
	if err == nil {
		t.Fatal("expected installing a category directory to fail")
	}
	if !strings.Contains(err.Error(), "tui") {
		t.Fatalf("unexpected error: %v", err)
	}
	mustNotExist(t, filepath.Join(sddmThemesDir, "tui"))
}

func TestInstallMalformedThemeLeavesNoPartialState(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createNestedSourceTree(t)
	broken := filepath.Join(srcRoot, "themes", "broken")
	if err := os.MkdirAll(broken, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(broken, "metadata.desktop"), []byte("[SddmGreeterTheme]"), 0o644); err != nil {
		t.Fatal(err)
	}

	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	fontDir = filepath.Join(tmpRoot, "fonts")
	if err := os.Setenv(sourceDirEnv, srcRoot); err != nil {
		t.Fatal(err)
	}
	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	for _, opts := range []InstallOptions{{}, {Split: true}} {
		err := Install(opts, io.Discard)
		if err == nil {
			t.Fatalf("expected install %+v to fail for malformed theme", opts)
		}
		if !strings.Contains(err.Error(), "malformed") {
			t.Fatalf("unexpected error: %v", err)
		}
		mustNotExist(t, filepath.Join(fontDir, "TestFont.ttf"))
		mustNotExist(t, filepath.Join(sddmThemesDir, fullThemeName))
		mustNotExist(t, filepath.Join(sddmThemesDir, "forest"))
		mustNotExist(t, filepath.Join(sddmThemesDir, "tui"))
	}
}

func TestListThemesReportsNestedIdentifiers(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createNestedSourceTree(t)
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

	got := strings.Split(strings.TrimSpace(out.String()), "\n")
	expected := []string{"forest", "tui/Amber", "tui/Emerald"}
	if strings.Join(got, ",") != strings.Join(expected, ",") {
		t.Fatalf("expected listed themes %v, got %v (%q)", expected, got, out.String())
	}
}

func TestListThemesDoesNotDescendIntoBundles(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createNestedSourceTree(t)
	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	fontDir = filepath.Join(tmpRoot, "fonts")
	if err := os.Setenv(sourceDirEnv, srcRoot); err != nil {
		t.Fatal(err)
	}
	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	if err := Install(InstallOptions{}, io.Discard); err != nil {
		t.Fatalf("full install failed: %v", err)
	}

	var out bytes.Buffer
	if err := ListThemes(&out); err != nil {
		t.Fatalf("ListThemes failed: %v", err)
	}

	got := strings.Fields(out.String())
	if len(got) != 1 || got[0] != fullThemeName {
		t.Fatalf("expected only the full bundle to be listed, got %v", got)
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

// TestDiscoverRealRepositoryTree is the regression test for the v0.4.5
// installer failure: discovery treated every immediate child of themes/ as
// a theme and aborted every install mode on the nested "tui" category
// directory. When the test suite runs from a repository checkout, it
// exercises discovery and validation against the actual theme tree.
func TestDiscoverRealRepositoryTree(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("cannot determine repository location")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))

	themesDir := filepath.Join(repoRoot, "themes")
	if _, err := os.Stat(filepath.Join(repoRoot, "Main.qml")); err != nil {
		t.Skipf("repository source tree not available at %s", repoRoot)
	}
	if _, err := os.Stat(filepath.Join(themesDir, "tui")); err != nil {
		t.Skipf("nested tui themes not present at %s", themesDir)
	}

	names, err := discoverSourceThemes(repoRoot)
	if err != nil {
		t.Fatalf("discovery failed against the real repository: %v", err)
	}

	// The discovered set must equal the actual on-disk theme tree, so this
	// regression stays catalog-agnostic as the repository grows.
	expected := make([]string, 0)
	entries, _ := os.ReadDir(themesDir)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if e.Name() == "tui" {
			variants, _ := os.ReadDir(filepath.Join(themesDir, "tui"))
			for _, v := range variants {
				if v.IsDir() {
					expected = append(expected, "tui/"+v.Name())
				}
			}
			continue
		}
		if _, err := os.Stat(filepath.Join(themesDir, e.Name(), "Main.qml")); err == nil {
			expected = append(expected, e.Name())
		}
	}
	sort.Strings(expected)

	for _, mandatory := range []string{"enfield", "forest", "nier-automata",
		"pixel-cyberpunk", "tui/Amber", "winter"} {
		found := false
		for _, n := range names {
			if n == mandatory {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("mandatory theme %q missing from discovery: %v", mandatory, names)
		}
	}
	if strings.Join(names, ",") != strings.Join(expected, ",") {
		t.Fatalf("expected repository themes:\n%v\ngot:\n%v", expected, names)
	}

	for _, name := range names {
		dir := filepath.Join(themesDir, filepath.FromSlash(name))
		if err := validateThemeSource(dir); err != nil {
			t.Fatalf("theme %q failed validation: %v", name, err)
		}
	}
}

func TestRemoveWarnsWhenRemovingActiveTheme(t *testing.T) {
	setupTestGlobals(t)

	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	sddmConfigPath = filepath.Join(tmpRoot, "conf", "theme.conf")
	if err := os.MkdirAll(filepath.Dir(sddmConfigPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sddmConfigPath, []byte("[Theme]\nCurrent=alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(sddmThemesDir, "alpha")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Main.qml"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := RemoveTheme("alpha", &out); err != nil {
		t.Fatalf("remove failed: %v", err)
	}
	if !strings.Contains(out.String(), "Warning:") || !strings.Contains(out.String(), "alpha") {
		t.Fatalf("expected active-theme warning, got: %q", out.String())
	}
	mustNotExist(t, dir)
}

func TestRemoveInactiveThemeStaysSilent(t *testing.T) {
	setupTestGlobals(t)

	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	sddmConfigPath = filepath.Join(tmpRoot, "conf", "theme.conf")
	if err := os.MkdirAll(filepath.Dir(sddmConfigPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sddmConfigPath, []byte("[Theme]\nCurrent=other\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{"other", "alpha"} {
		dir := filepath.Join(sddmThemesDir, filepath.FromSlash(id))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Main.qml"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var out bytes.Buffer
	if err := RemoveTheme("alpha", &out); err != nil {
		t.Fatalf("remove failed: %v", err)
	}
	if strings.Contains(out.String(), "Warning") {
		t.Fatalf("did not expect a warning for inactive theme, got: %q", out.String())
	}
	mustNotExist(t, filepath.Join(sddmThemesDir, "alpha"))
}
