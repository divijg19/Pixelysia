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

func TestCheckFontDiscoveryIsReadOnly(t *testing.T) {
	setupTestGlobals(t)

	var invoked []string
	commandRunner = func(name string, args ...string) *exec.Cmd {
		invoked = append(invoked, name)
		if name == "fc-list" {
			return exec.Command("printf", fontDir+"/Foo.ttf: family\n")
		}
		return exec.Command("true")
	}

	result := checkFontDiscovery()
	if !result.OK {
		t.Fatalf("expected discovery to pass, got %+v", result)
	}
	if !strings.Contains(result.Detail, "1 font") {
		t.Fatalf("unexpected detail: %+v", result)
	}
	for _, name := range invoked {
		if name == "fc-cache" {
			t.Fatal("font diagnostics must never invoke fc-cache")
		}
	}
}

func TestCheckFontDiscoveryFailsWhenFontsNotIndexed(t *testing.T) {
	setupTestGlobals(t)

	commandRunner = func(name string, args ...string) *exec.Cmd {
		return exec.Command("printf", "/elsewhere/Foo.ttf: family\n")
	}

	result := checkFontDiscovery()
	if result.OK {
		t.Fatal("expected discovery to fail when no pixelysia fonts are indexed")
	}
	if !strings.Contains(result.Detail, "no fonts discovered") {
		t.Fatalf("unexpected detail: %+v", result)
	}
}

func TestCheckCurrentTheme(t *testing.T) {
	setupTestGlobals(t)

	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")
	sddmConfigDir = filepath.Join(tmpRoot, "conf")
	sddmConfigPath = filepath.Join(sddmConfigDir, "theme.conf")

	writeConfig := func(content string) {
		t.Helper()
		if err := os.MkdirAll(sddmConfigDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(sddmConfigPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	installTheme := func(id string) {
		t.Helper()
		dir := filepath.Join(sddmThemesDir, id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Main.qml"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Missing configuration is unhealthy for the current-theme check.
	os.Remove(sddmConfigPath)
	if r := checkCurrentTheme(); r.OK {
		t.Fatalf("expected missing configuration to fail, got %+v", r)
	}

	// Configuration without any Current= selection.
	writeConfig("[Theme]\n")
	if r := checkCurrentTheme(); r.OK {
		t.Fatalf("expected absent current theme to fail, got %+v", r)
	}

	// Installed current theme is healthy.
	installTheme("forest")
	writeConfig("[Theme]\nCurrent=forest\n")
	if r := checkCurrentTheme(); !r.OK || r.Detail != "forest" {
		t.Fatalf("expected healthy forest, got %+v", r)
	}

	// Nested identifiers resolve through the canonical installed listing.
	installTheme(filepath.Join("tui", "Amber"))
	writeConfig("[Theme]\nCurrent=tui/Amber\n")
	if r := checkCurrentTheme(); !r.OK || r.Detail != "tui/Amber" {
		t.Fatalf("expected healthy tui/Amber, got %+v", r)
	}

	// The audited failure mode: dangling Current after removal.
	writeConfig("[Theme]\nCurrent=forest\n")
	if err := os.RemoveAll(filepath.Join(sddmThemesDir, "forest")); err != nil {
		t.Fatal(err)
	}
	r := checkCurrentTheme()
	if r.OK {
		t.Fatalf("expected dangling current theme to fail, got %+v", r)
	}
	if !strings.Contains(r.Detail, `"forest"`) || !strings.Contains(r.Detail, "not installed") {
		t.Fatalf("unexpected detail: %+v", r)
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
		if name == "fc-list" {
			return exec.Command("printf", fontDir+"/A.ttf: family\n")
		}
		return exec.Command("true")
	}

	var out bytes.Buffer
	err := RunDoctor(&out)
	if err != nil {
		t.Fatalf("expected doctor to pass, got %v\n%s", err, out.String())
	}

	output := out.String()
	if strings.Contains(output, "[FAIL]") {
		t.Fatalf("did not expect FAIL output, got:\n%s", output)
	}
	if !strings.Contains(output, "[OK] current theme: alpha") {
		t.Fatalf("expected current theme check to pass, got:\n%s", output)
	}
	if strings.Count(output, "[OK]") < 6 {
		t.Fatalf("expected all six checks to report OK, got:\n%s", output)
	}
}
