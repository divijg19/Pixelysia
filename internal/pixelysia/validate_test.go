package pixelysia

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// createValidateFixture builds a source tree for validator tests and returns
// its root together with a helper to add themes.
func createValidateFixture(t *testing.T) (string, func(id string)) {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "fonts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "fonts", "F.ttf"), []byte("f"), 0o644); err != nil {
		t.Fatal(err)
	}

	addTheme := func(id string) {
		t.Helper()
		dir := filepath.Join(root, "themes", filepath.FromSlash(id))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Main.qml"), []byte("font.family: \"Pixelify Sans\""), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "metadata.desktop"), []byte("[SddmGreeterTheme]"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root, addTheme
}

func writeDispatcher(t *testing.T, root string, refs ...string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("Loader { source: random([\n")
	for _, ref := range refs {
		b.WriteString("\t\"" + ref + "\",\n")
	}
	b.WriteString("]) }")
	if err := os.WriteFile(filepath.Join(root, "Main.qml"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestValidateCleanTree(t *testing.T) {
	setupTestGlobals(t)

	root, addTheme := createValidateFixture(t)
	addTheme("forest")
	addTheme("tui/Amber")
	writeDispatcher(t, root, "themes/forest/Main.qml", "themes/tui/Amber/Main.qml")

	var out bytes.Buffer
	if err := ValidateSource(root, &out); err != nil {
		t.Fatalf("expected clean tree to pass: %v\n%s", err, out.String())
	}
	for _, want := range []string{"[OK] forest", "[OK] tui/Amber", "[OK] dispatcher"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out.String())
		}
	}
}

func TestValidateBrokenBackgroundReference(t *testing.T) {
	setupTestGlobals(t)

	root, addTheme := createValidateFixture(t)
	addTheme("pixel-broken")
	writeDispatcher(t, root, "themes/pixel-broken/Main.qml")
	if err := os.WriteFile(filepath.Join(root, "themes", "pixel-broken", "theme.conf"),
		[]byte("[General]\nbackground=missing.mp4\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	err := ValidateSource(root, &out)
	if err == nil {
		t.Fatalf("expected broken background reference to fail:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "background=missing.mp4 does not exist") {
		t.Fatalf("unexpected output:\n%s", out.String())
	}
}

func TestValidateBrokenFontDeclaration(t *testing.T) {
	setupTestGlobals(t)

	root, addTheme := createValidateFixture(t)
	addTheme("sword-like") // QML declares "Pixelify Sans"
	writeDispatcher(t, root, "themes/sword-like/Main.qml")
	// The historical sword defect: conf declares a font the QML never uses.
	if err := os.WriteFile(filepath.Join(root, "themes", "sword-like", "theme.conf"),
		[]byte("[General]\nfont=Outfit\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	err := ValidateSource(root, &out)
	if err == nil {
		t.Fatalf("expected stale font declaration to fail:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "font=Outfit is not referenced in Main.qml") {
		t.Fatalf("unexpected output:\n%s", out.String())
	}
}

func TestValidateFontNormalizationMatchesConvention(t *testing.T) {
	setupTestGlobals(t)

	root, addTheme := createValidateFixture(t)
	addTheme("normalized")
	writeDispatcher(t, root, "themes/normalized/Main.qml")
	if err := os.WriteFile(filepath.Join(root, "themes", "normalized", "theme.conf"),
		[]byte("[General]\nfont=PixelifySans\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := ValidateSource(root, &out); err != nil {
		t.Fatalf("expected spacing/case-insensitive font match to pass: %v\n%s", err, out.String())
	}
}

func TestValidateDanglingDispatcherReference(t *testing.T) {
	setupTestGlobals(t)

	root, addTheme := createValidateFixture(t)
	addTheme("forest")
	writeDispatcher(t, root, "themes/forest/Main.qml", "themes/ghost/Main.qml")

	var out bytes.Buffer
	err := ValidateSource(root, &out)
	if err == nil {
		t.Fatalf("expected dangling dispatcher reference to fail:\n%s", out.String())
	}
	if !strings.Contains(out.String(), `dispatcher reference "themes/ghost/Main.qml" does not resolve`) {
		t.Fatalf("unexpected output:\n%s", out.String())
	}
}

func TestValidateMalformedThemeDirectory(t *testing.T) {
	setupTestGlobals(t)

	root, _ := createValidateFixture(t)
	broken := filepath.Join(root, "themes", "broken")
	if err := os.MkdirAll(broken, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(broken, "metadata.desktop"), []byte("[SddmGreeterTheme]"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := ValidateSource(root, &out); err == nil {
		t.Fatal("expected malformed theme directory to fail validation")
	}
}

func TestValidateMissingMetadataIsInformationalOnly(t *testing.T) {
	setupTestGlobals(t)

	root, addTheme := createValidateFixture(t)
	addTheme("bare")
	writeDispatcher(t, root, "themes/bare/Main.qml")
	if err := os.Remove(filepath.Join(root, "themes", "bare", "metadata.desktop")); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := ValidateSource(root, &out); err != nil {
		t.Fatalf("expected missing metadata.desktop to be advisory only: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "[INFO] bare: no metadata.desktop") {
		t.Fatalf("expected informational finding, got:\n%s", out.String())
	}
}

func TestValidateInvalidSourceRoot(t *testing.T) {
	setupTestGlobals(t)

	dir := t.TempDir()
	var out bytes.Buffer
	if err := ValidateSource(dir, &out); err == nil {
		t.Fatal("expected invalid source root to fail")
	} else if !strings.Contains(err.Error(), "Main.qml") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestValidateRealRepository exercises the validator against the actual
// Pixelysia checkout when tests run from within it. It must remain green:
// any fatal finding in the real tree is a release defect.
func TestValidateRealRepository(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("cannot determine repository location")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	if _, err := os.Stat(filepath.Join(repoRoot, "themes", "tui")); err != nil {
		t.Skipf("repository source tree not available at %s", repoRoot)
	}

	var out bytes.Buffer
	if err := ValidateSource(repoRoot, &out); err != nil {
		t.Fatalf("real repository failed validation: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "28 ok") {
		t.Fatalf("expected all 28 repository themes to validate, got:\n%s", out.String())
	}
}
