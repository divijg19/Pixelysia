package pixelysia

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetThemeCreatesConfig(t *testing.T) {
	setupTestGlobals(t)

	root := t.TempDir()
	sddmThemesDir = filepath.Join(root, "themes")
	sddmConfigDir = filepath.Join(root, "conf")
	sddmConfigPath = filepath.Join(sddmConfigDir, "theme.conf")

	themeDir := filepath.Join(sddmThemesDir, "alpha")
	if err := os.MkdirAll(themeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := SetTheme("alpha"); err != nil {
		t.Fatalf("SetTheme failed: %v", err)
	}

	cfg, err := os.ReadFile(sddmConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), "[Theme]") || !strings.Contains(string(cfg), "Current=alpha") {
		t.Fatalf("unexpected config contents:\n%s", string(cfg))
	}
}

func TestSetThemeMergesWithExistingConfig(t *testing.T) {
	setupTestGlobals(t)

	root := t.TempDir()
	sddmThemesDir = filepath.Join(root, "themes")
	sddmConfigDir = filepath.Join(root, "conf")
	sddmConfigPath = filepath.Join(sddmConfigDir, "theme.conf")

	if err := os.MkdirAll(filepath.Join(sddmThemesDir, "beta"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sddmConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}

	existing := "[General]\nDisplayServer=x11\n\n[Theme]\nCurrent=old\n"
	if err := os.WriteFile(sddmConfigPath, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SetTheme("beta"); err != nil {
		t.Fatalf("SetTheme failed: %v", err)
	}

	cfg, err := os.ReadFile(sddmConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(cfg)
	if !strings.Contains(got, "DisplayServer=x11") {
		t.Fatalf("expected existing config to be preserved: %s", got)
	}
	if !strings.Contains(got, "Current=beta") {
		t.Fatalf("expected theme to be updated: %s", got)
	}
	if strings.Contains(got, "Current=old") {
		t.Fatalf("expected old theme value to be replaced: %s", got)
	}
}

func TestCurrentTheme(t *testing.T) {
	setupTestGlobals(t)

	root := t.TempDir()
	sddmConfigDir = filepath.Join(root, "conf")
	sddmConfigPath = filepath.Join(sddmConfigDir, "theme.conf")
	if err := os.MkdirAll(sddmConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := "[General]\nX11=true\n\n[Theme]\nCurrent=gamma\n"
	if err := os.WriteFile(sddmConfigPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	current, err := CurrentTheme()
	if err != nil {
		t.Fatalf("CurrentTheme failed: %v", err)
	}
	if current != "gamma" {
		t.Fatalf("unexpected current theme: %s", current)
	}
}

func TestCurrentThemeMissing(t *testing.T) {
	setupTestGlobals(t)

	root := t.TempDir()
	sddmConfigDir = filepath.Join(root, "conf")
	sddmConfigPath = filepath.Join(sddmConfigDir, "theme.conf")
	if err := os.MkdirAll(sddmConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sddmConfigPath, []byte("[General]\nX11=true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := CurrentTheme()
	if err == nil {
		t.Fatal("expected CurrentTheme to fail when [Theme]/Current is missing")
	}
}

func TestUpdateThemeConfigAddsMissingThemeSection(t *testing.T) {
	setupTestGlobals(t)

	in := []byte("[General]\nDisplayServer=wayland\n")
	out := string(updateThemeConfig(in, "delta"))

	if !strings.Contains(out, "[General]") || !strings.Contains(out, "DisplayServer=wayland") {
		t.Fatalf("expected existing section to remain: %s", out)
	}
	if !strings.Contains(out, "[Theme]") || !strings.Contains(out, "Current=delta") {
		t.Fatalf("expected theme section to be added: %s", out)
	}
}

func TestSetThemeFailsWhenThemeNotInstalled(t *testing.T) {
	setupTestGlobals(t)

	root := t.TempDir()
	sddmThemesDir = filepath.Join(root, "themes")
	sddmConfigDir = filepath.Join(root, "conf")
	sddmConfigPath = filepath.Join(sddmConfigDir, "theme.conf")

	err := SetTheme("missing")
	if err == nil {
		t.Fatal("expected SetTheme to fail for missing installed theme")
	}
}

func TestSetThemeWithEmptyConfig(t *testing.T) {
	setupTestGlobals(t)

	root := t.TempDir()
	sddmThemesDir = filepath.Join(root, "themes")
	sddmConfigDir = filepath.Join(root, "conf")
	sddmConfigPath = filepath.Join(sddmConfigDir, "theme.conf")

	if err := os.MkdirAll(filepath.Join(sddmThemesDir, "omega"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sddmConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sddmConfigPath, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SetTheme("omega"); err != nil {
		t.Fatalf("SetTheme failed: %v", err)
	}

	b, err := os.ReadFile(sddmConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	out := strings.TrimSpace(string(b))
	if out != "[Theme]\nCurrent=omega" {
		t.Fatalf("unexpected config for empty input:\n%s", string(b))
	}
}

func TestSetThemeWithMalformedConfig(t *testing.T) {
	setupTestGlobals(t)

	root := t.TempDir()
	sddmThemesDir = filepath.Join(root, "themes")
	sddmConfigDir = filepath.Join(root, "conf")
	sddmConfigPath = filepath.Join(sddmConfigDir, "theme.conf")

	if err := os.MkdirAll(filepath.Join(sddmThemesDir, "delta"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sddmConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}

	malformed := "[General\nDisplayServer=x11\n"
	if err := os.WriteFile(sddmConfigPath, []byte(malformed), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SetTheme("delta"); err != nil {
		t.Fatalf("SetTheme failed on malformed config: %v", err)
	}

	b, err := os.ReadFile(sddmConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	out := string(b)
	if !strings.Contains(out, "DisplayServer=x11") {
		t.Fatalf("expected malformed content to be preserved, got:\n%s", out)
	}
	if !strings.Contains(out, "[Theme]\nCurrent=delta") {
		t.Fatalf("expected [Theme] section to be appended, got:\n%s", out)
	}
}

func TestSetThemeWithDuplicateThemeSections(t *testing.T) {
	setupTestGlobals(t)

	root := t.TempDir()
	sddmThemesDir = filepath.Join(root, "themes")
	sddmConfigDir = filepath.Join(root, "conf")
	sddmConfigPath = filepath.Join(sddmConfigDir, "theme.conf")

	if err := os.MkdirAll(filepath.Join(sddmThemesDir, "neo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sddmConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}

	in := "[Theme]\nCurrent=one\n\n[Theme]\nCurrent=two\n"
	if err := os.WriteFile(sddmConfigPath, []byte(in), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SetTheme("neo"); err != nil {
		t.Fatalf("SetTheme failed: %v", err)
	}

	b, err := os.ReadFile(sddmConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	out := string(b)
	if strings.Count(out, "Current=neo") != 2 {
		t.Fatalf("expected both theme sections to be updated, got:\n%s", out)
	}
	if strings.Contains(out, "Current=one") || strings.Contains(out, "Current=two") {
		t.Fatalf("expected old values to be replaced, got:\n%s", out)
	}
}
