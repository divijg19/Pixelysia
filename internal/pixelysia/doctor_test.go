package pixelysia

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckFontsInstalledMissing(t *testing.T) {
	setupTestGlobals(t)
	fontDir = filepath.Join(t.TempDir(), "fonts")

	result := checkFontsInstalled()
	if result.OK {
		t.Fatalf("expected fonts check to fail, got %+v", result)
	}
}

func TestCheckThemesPresentMissing(t *testing.T) {
	setupTestGlobals(t)
	sddmThemesDir = filepath.Join(t.TempDir(), "themes")

	result := checkThemesPresent()
	if result.OK {
		t.Fatalf("expected themes check to fail, got %+v", result)
	}
}

func TestCheckThemePermissionsInvalidModes(t *testing.T) {
	setupTestGlobals(t)

	root := t.TempDir()
	sddmThemesDir = filepath.Join(root, "themes")
	themeDir := filepath.Join(sddmThemesDir, "alpha")
	if err := os.MkdirAll(themeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(themeDir, "Main.qml")
	if err := os.WriteFile(filePath, []byte("qml"), 0o600); err != nil {
		t.Fatal(err)
	}

	result := checkThemePermissions()
	if result.OK {
		t.Fatalf("expected permission check to fail, got %+v", result)
	}
	if !strings.Contains(result.Detail, "expected 644") {
		t.Fatalf("unexpected detail: %s", result.Detail)
	}
}

func TestCheckThemePermissionsValidModes(t *testing.T) {
	setupTestGlobals(t)

	root := t.TempDir()
	sddmThemesDir = filepath.Join(root, "themes")
	themeDir := filepath.Join(sddmThemesDir, "alpha")
	if err := os.MkdirAll(themeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "Main.qml"), []byte("qml"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := checkThemePermissions()
	if !result.OK {
		t.Fatalf("expected permission check to pass, got %+v", result)
	}
}

func TestCheckConfigExistsMissing(t *testing.T) {
	setupTestGlobals(t)
	sddmConfigPath = filepath.Join(t.TempDir(), "theme.conf")

	result := checkConfigExists()
	if result.OK {
		t.Fatalf("expected config check to fail, got %+v", result)
	}
}

func TestCheckFontCacheUsesCommandRunner(t *testing.T) {
	setupTestGlobals(t)

	called := false
	commandRunner = func(name string, args ...string) *exec.Cmd {
		if name == "fc-cache" && len(args) == 1 && args[0] == "-f" {
			called = true
		}
		return exec.Command("true")
	}

	result := checkFontCache()
	if !result.OK {
		t.Fatalf("expected cache check to pass, got %+v", result)
	}
	if !called {
		t.Fatal("expected fc-cache -f to be invoked")
	}
}

func TestCheckFontCacheFailure(t *testing.T) {
	setupTestGlobals(t)

	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	result := checkFontCache()
	if result.OK {
		t.Fatalf("expected cache check to fail, got %+v", result)
	}
}

func TestRunDoctorReportsFailures(t *testing.T) {
	setupTestGlobals(t)

	root := t.TempDir()
	fontDir = filepath.Join(root, "fonts")
	sddmThemesDir = filepath.Join(root, "themes")
	sddmConfigPath = filepath.Join(root, "conf", "theme.conf")
	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	var out bytes.Buffer
	err := RunDoctor(&out)
	if err == nil {
		t.Fatal("expected doctor to return error when checks fail")
	}

	output := out.String()
	if !strings.Contains(output, "[FAIL]") {
		t.Fatalf("expected FAIL output, got:\n%s", output)
	}
}

func TestRunDoctorAllPass(t *testing.T) {
	setupTestGlobals(t)

	root := t.TempDir()
	fontDir = filepath.Join(root, "fonts")
	sddmThemesDir = filepath.Join(root, "themes")
	sddmConfigPath = filepath.Join(root, "conf", "theme.conf")
	sddmConfigDir = filepath.Dir(sddmConfigPath)

	if err := os.MkdirAll(fontDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fontDir, "A.ttf"), []byte("font"), 0o644); err != nil {
		t.Fatal(err)
	}

	themeDir := filepath.Join(sddmThemesDir, "alpha")
	if err := os.MkdirAll(themeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "Main.qml"), []byte("qml"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(sddmConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sddmConfigPath, []byte("[Theme]\nCurrent=alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	var out bytes.Buffer
	err := RunDoctor(&out)
	if err != nil {
		t.Fatalf("expected doctor to pass, got %v", err)
	}

	output := out.String()
	if strings.Contains(output, "[FAIL]") {
		t.Fatalf("did not expect FAIL output, got:\n%s", output)
	}
	if strings.Count(output, "[OK]") < 5 {
		t.Fatalf("expected all checks to report OK, got:\n%s", output)
	}
}
