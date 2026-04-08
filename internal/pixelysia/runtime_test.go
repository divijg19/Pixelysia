package pixelysia

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type commandResult struct {
	stdout string
	stderr string
	err    error
}

func TestRuntimeCLIExecutionFlow(t *testing.T) {
	srcRoot := createSourceTree(t, []string{"alpha", "beta"})
	root := t.TempDir()
	themesDir := filepath.Join(root, "themes")
	fontsDir := filepath.Join(root, "fonts")
	configDir := filepath.Join(root, "conf")
	configPath := filepath.Join(configDir, "theme.conf")
	fakeBin := createFakeBin(t)

	env := []string{
		sourceDirEnv + "=" + srcRoot,
		envThemesDir + "=" + themesDir,
		envFontDir + "=" + fontsDir,
		envConfigDir + "=" + configDir,
		envConfigPath + "=" + configPath,
		envReqUID + "=" + fmt.Sprintf("%d", os.Getuid()),
		envReqGID + "=" + fmt.Sprintf("%d", os.Getgid()),
		"PATH=" + fakeBin + string(os.PathListSeparator) + os.Getenv("PATH"),
	}

	res := runPixelysiaCommand(t, env, "install", "--split")
	if res.err != nil {
		t.Fatalf("install failed: %v\nstderr=%s", res.err, res.stderr)
	}
	if !strings.Contains(res.stdout, "Installing theme: alpha") {
		t.Fatalf("unexpected install output: %s", res.stdout)
	}

	res = runPixelysiaCommand(t, env, "set", "alpha")
	if res.err != nil {
		t.Fatalf("set failed: %v\nstderr=%s", res.err, res.stderr)
	}

	res = runPixelysiaCommand(t, env, "current")
	if res.err != nil {
		t.Fatalf("current failed: %v\nstderr=%s", res.err, res.stderr)
	}
	if strings.TrimSpace(res.stdout) != "alpha" {
		t.Fatalf("expected current theme alpha, got %q", strings.TrimSpace(res.stdout))
	}

	res = runPixelysiaCommand(t, env, "list")
	if res.err != nil {
		t.Fatalf("list failed: %v\nstderr=%s", res.err, res.stderr)
	}
	if !strings.Contains(res.stdout, "alpha") || !strings.Contains(res.stdout, "beta") {
		t.Fatalf("unexpected list output: %s", res.stdout)
	}

	res = runPixelysiaCommand(t, env, "doctor")
	if res.err != nil {
		t.Fatalf("doctor failed: %v\nstderr=%s", res.err, res.stderr)
	}
	if !strings.Contains(res.stdout, "[OK]") {
		t.Fatalf("expected doctor OK output, got: %s", res.stdout)
	}
}

func TestRuntimeCLIErrorExitAndStderr(t *testing.T) {
	srcRoot := createSourceTree(t, []string{"alpha"})
	root := t.TempDir()
	fakeBin := createFakeBin(t)

	env := []string{
		sourceDirEnv + "=" + srcRoot,
		envThemesDir + "=" + filepath.Join(root, "themes"),
		envFontDir + "=" + filepath.Join(root, "fonts"),
		envConfigDir + "=" + filepath.Join(root, "conf"),
		envConfigPath + "=" + filepath.Join(root, "conf", "theme.conf"),
		envReqUID + "=" + fmt.Sprintf("%d", os.Getuid()),
		envReqGID + "=" + fmt.Sprintf("%d", os.Getgid()),
		"PATH=" + fakeBin + string(os.PathListSeparator) + os.Getenv("PATH"),
	}

	res := runPixelysiaCommand(t, env, "install", "--split", "--theme", "alpha")
	if res.err == nil {
		t.Fatal("expected command to fail")
	}
	if !strings.Contains(res.stderr, "cannot be used together") {
		t.Fatalf("unexpected stderr: %s", res.stderr)
	}
}

func TestRuntimeSimulatedSDDMCompatibility(t *testing.T) {
	srcRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcRoot, "Main.qml"), []byte("Loader { source: \"themes/alpha/Main.qml\" }"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcRoot, "metadata.desktop"), []byte("[SddmGreeterTheme]\nName=Pixelysia"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(srcRoot, "fonts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcRoot, "fonts", "A.ttf"), []byte("font"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(srcRoot, "themes", "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcRoot, "themes", "alpha", "Main.qml"), []byte("import QtQuick 2.15"), 0o644); err != nil {
		t.Fatal(err)
	}

	installRoot := t.TempDir()
	themesDir := filepath.Join(installRoot, "themes")
	fontsDir := filepath.Join(installRoot, "fonts")
	fakeBin := createFakeBin(t)

	env := []string{
		sourceDirEnv + "=" + srcRoot,
		envThemesDir + "=" + themesDir,
		envFontDir + "=" + fontsDir,
		envConfigDir + "=" + filepath.Join(installRoot, "conf"),
		envConfigPath + "=" + filepath.Join(installRoot, "conf", "theme.conf"),
		envReqUID + "=" + fmt.Sprintf("%d", os.Getuid()),
		envReqGID + "=" + fmt.Sprintf("%d", os.Getgid()),
		"PATH=" + fakeBin + string(os.PathListSeparator) + os.Getenv("PATH"),
	}

	res := runPixelysiaCommand(t, env, "install")
	if res.err != nil {
		t.Fatalf("full install failed: %v\nstderr=%s", res.err, res.stderr)
	}

	fullRoot := filepath.Join(themesDir, fullThemeName)
	mustExistFile(t, filepath.Join(fullRoot, "Main.qml"))
	mustExistFile(t, filepath.Join(fullRoot, "metadata.desktop"))
	mustExistFile(t, filepath.Join(fullRoot, "themes", "alpha", "Main.qml"))
	mustExistFile(t, filepath.Join(fullRoot, "fonts", "A.ttf"))

	dispatcher, err := os.ReadFile(filepath.Join(fullRoot, "Main.qml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dispatcher), "themes/alpha/Main.qml") {
		t.Fatalf("expected dispatcher to reference installed theme path, got: %s", string(dispatcher))
	}
}

func runPixelysiaCommand(t *testing.T, env []string, args ...string) commandResult {
	t.Helper()

	moduleRoot := filepath.Clean(filepath.Join(packageDir(t), "..", ".."))
	cmdArgs := append([]string{"run", "./cmd/pixelysia"}, args...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = moduleRoot
	cmd.Env = append(os.Environ(), env...)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	err := cmd.Run()
	return commandResult{stdout: out.String(), stderr: errOut.String(), err: err}
}

func packageDir(t *testing.T) string {
	t.Helper()
	d, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func createFakeBin(t *testing.T) string {
	t.Helper()

	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}

	fcCache := filepath.Join(binDir, "fc-cache")
	script := "#!/usr/bin/env sh\nexit 0\n"
	if err := os.WriteFile(fcCache, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	return binDir
}
