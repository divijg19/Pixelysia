package pixelysia

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadThemeConf(t *testing.T) {
	setupTestGlobals(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "theme.conf")
	content := "" +
		"# comment line\n" +
		"\n" +
		"[General]\n" +
		"background = bg.mp4\n" +
		"; semicolon comment\n" +
		"font=Pixelify Sans\n" +
		"color=#f0ede8\n" +
		"[Other]\n" +
		"color=#ffffff\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	conf, err := readThemeConf(path)
	if err != nil {
		t.Fatalf("readThemeConf failed: %v", err)
	}
	if conf == nil {
		t.Fatal("expected parsed configuration")
	}
	if conf["background"] != "bg.mp4" {
		t.Fatalf("unexpected background %q", conf["background"])
	}
	if conf["font"] != "Pixelify Sans" {
		t.Fatalf("unexpected font %q", conf["font"])
	}
	if conf["color"] != "#ffffff" {
		t.Fatalf("expected later section to override duplicate key, got %q", conf["color"])
	}

	// Missing files are not an error; theme.conf is optional.
	conf, err = readThemeConf(filepath.Join(dir, "absent.conf"))
	if err != nil || conf != nil {
		t.Fatalf("expected (nil, nil) for missing file, got (%v, %v)", conf, err)
	}

	// Genuinely malformed lines must surface.
	if err := os.WriteFile(filepath.Join(dir, "broken.conf"), []byte("not-a-pair\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readThemeConf(filepath.Join(dir, "broken.conf")); err == nil {
		t.Fatal("expected error for malformed configuration line")
	} else if !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCollectSourceThemesClassification(t *testing.T) {
	setupTestGlobals(t)

	root := createNestedSourceTree(t)
	// Extend the fixture with additional classification cases.
	themesDir := filepath.Join(root, "themes")

	// Deeply nested theme.
	deep := filepath.Join(themesDir, "cat", "sub", "DeepOne")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deep, "Main.qml"), []byte("import QtQuick"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Theme with its own theme.conf.
	withConf := filepath.Join(themesDir, "confessed")
	if err := os.MkdirAll(withConf, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(withConf, "Main.qml"), []byte("font.family: \"Pixelify Sans\""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(withConf, "theme.conf"), []byte("[General]\nfont=PixelifySans\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Hidden directory containing a Main.qml is skipped entirely.
	hidden := filepath.Join(themesDir, ".stash")
	if err := os.MkdirAll(hidden, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hidden, "Main.qml"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	themes, err := collectSourceThemes(root)
	if err != nil {
		t.Fatalf("collectSourceThemes failed: %v", err)
	}

	got := make([]string, 0, len(themes))
	for _, th := range themes {
		got = append(got, th.ID)
	}
	expected := []string{"cat/sub/DeepOne", "confessed", "forest", "tui/Amber", "tui/Emerald"}
	if strings.Join(got, ",") != strings.Join(expected, ",") {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	for _, th := range themes {
		if th.ID == "confessed" && th.Conf["font"] != "PixelifySans" {
			t.Fatalf("expected confessed theme to carry parsed conf, got %#v", th.Conf)
		}
		if th.ID == "forest" && th.Conf != nil {
			t.Fatalf("expected nil conf for theme without theme.conf, got %#v", th.Conf)
		}
	}
}

func TestCollectSourceThemesRejectsMarkersInRoot(t *testing.T) {
	setupTestGlobals(t)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "themes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "themes", "Main.qml"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := collectSourceThemes(root); err == nil {
		t.Fatal("expected error for Main.qml inside the themes root")
	} else if !strings.Contains(err.Error(), "themes directory itself") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCollectInstalledThemesNestedAndBundles(t *testing.T) {
	setupTestGlobals(t)

	tmpRoot := t.TempDir()
	sddmThemesDir = filepath.Join(tmpRoot, "themes")

	write := func(rel string, name string, content string) {
		t.Helper()
		dir := filepath.Join(sddmThemesDir, rel)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("plain", "metadata.desktop", "[SddmGreeterTheme]")
	write("tui/Amber", "Main.qml", "x")
	write(fullThemeName, "Main.qml", "dispatcher")
	write(filepath.Join(fullThemeName, "themes", "inner"), "Main.qml", "bundle member")

	names, err := collectInstalledThemes()
	if err != nil {
		t.Fatalf("collectInstalledThemes failed: %v", err)
	}

	// Sorted lexically: "pixelysia" < "plain" < "tui/Amber".
	expected := []string{fullThemeName, "plain", "tui/Amber"}
	if strings.Join(names, ",") != strings.Join(expected, ",") {
		t.Fatalf("expected %v, got %v", expected, names)
	}
}
