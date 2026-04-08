package pixelysia

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	fullThemeName = "pixelysia"

	sddmThemesDir = "/usr/share/sddm/themes"
	fontDir       = "/usr/share/fonts/pixelysia"
)

type InstallOptions struct {
	Split bool
	Theme string
}

func Install(opts InstallOptions, out io.Writer) error {
	if err := requireNoMutuallyExclusive(opts.Split, opts.Theme); err != nil {
		return err
	}

	srcRoot, err := detectSourceRoot()
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintln(out, "Installing fonts..."); err != nil {
		return err
	}
	if err := installFonts(srcRoot); err != nil {
		return err
	}

	if err := os.MkdirAll(sddmThemesDir, 0o755); err != nil {
		return fmt.Errorf("create themes directory: %w", err)
	}

	switch {
	case opts.Split:
		themeNames, err := discoverSourceThemes(srcRoot)
		if err != nil {
			return err
		}
		for _, name := range themeNames {
			if _, err := fmt.Fprintf(out, "Installing theme: %s\n", name); err != nil {
				return err
			}
			if err := installSingleSplitTheme(srcRoot, name); err != nil {
				return err
			}
		}
		return nil

	case opts.Theme != "":
		if _, err := fmt.Fprintf(out, "Installing theme: %s\n", opts.Theme); err != nil {
			return err
		}
		if err := installSingleSplitTheme(srcRoot, opts.Theme); err != nil {
			return err
		}
		return nil

	default:
		if _, err := fmt.Fprintf(out, "Installing theme: %s\n", fullThemeName); err != nil {
			return err
		}
		return installFullTheme(srcRoot)
	}
}

func installFonts(srcRoot string) error {
	srcPattern := filepath.Join(srcRoot, "fonts", "*.ttf")
	fonts, err := filepath.Glob(srcPattern)
	if err != nil {
		return fmt.Errorf("read font sources: %w", err)
	}
	if len(fonts) == 0 {
		return errors.New("no .ttf files found in fonts directory")
	}

	tmpDir, err := os.MkdirTemp(filepath.Dir(fontDir), ".pixelysia-fonts-")
	if err != nil {
		return fmt.Errorf("create temp fonts directory: %w", err)
	}
	cleanupTmp := true
	defer func() {
		if cleanupTmp {
			_ = os.RemoveAll(tmpDir)
		}
	}()

	if err := os.Chmod(tmpDir, 0o755); err != nil {
		return fmt.Errorf("set temp fonts permissions: %w", err)
	}
	if err := os.Chown(tmpDir, 0, 0); err != nil {
		return fmt.Errorf("set temp fonts ownership: %w", err)
	}

	for _, src := range fonts {
		dst := filepath.Join(tmpDir, filepath.Base(src))
		if err := copyFile(src, dst, 0o644); err != nil {
			return err
		}
		if err := os.Chown(dst, 0, 0); err != nil {
			return fmt.Errorf("set font ownership %s: %w", dst, err)
		}
	}

	if err := replaceDirAtomic(tmpDir, fontDir); err != nil {
		return err
	}
	cleanupTmp = false

	cmd := exec.Command("fc-cache", "-fv")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rebuild font cache with fc-cache: %w", err)
	}

	return nil
}

func installFullTheme(srcRoot string) error {
	tmpDir, err := os.MkdirTemp(sddmThemesDir, ".pixelysia-theme-")
	if err != nil {
		return fmt.Errorf("create temp theme directory: %w", err)
	}
	cleanupTmp := true
	defer func() {
		if cleanupTmp {
			_ = os.RemoveAll(tmpDir)
		}
	}()

	mainQML := filepath.Join(srcRoot, "Main.qml")
	if err := ensureRegularFile(mainQML); err != nil {
		return fmt.Errorf("validate dispatcher Main.qml: %w", err)
	}
	if err := copyFile(mainQML, filepath.Join(tmpDir, "Main.qml"), 0o644); err != nil {
		return err
	}

	metadata := filepath.Join(srcRoot, "metadata.desktop")
	if _, err := os.Stat(metadata); err == nil {
		if err := copyFile(metadata, filepath.Join(tmpDir, "metadata.desktop"), 0o644); err != nil {
			return err
		}
	}

	if err := copyDir(filepath.Join(srcRoot, "themes"), filepath.Join(tmpDir, "themes")); err != nil {
		return err
	}
	if err := copyDir(filepath.Join(srcRoot, "fonts"), filepath.Join(tmpDir, "fonts")); err != nil {
		return err
	}

	if err := setOwnershipAndModeRecursive(tmpDir, 0, 0, 0o755, 0o755); err != nil {
		return err
	}

	dst := filepath.Join(sddmThemesDir, fullThemeName)
	if err := replaceDirAtomic(tmpDir, dst); err != nil {
		return err
	}
	cleanupTmp = false
	return nil
}

func installSingleSplitTheme(srcRoot string, themeName string) error {
	if err := validateThemeName(themeName); err != nil {
		return err
	}

	src := filepath.Join(srcRoot, "themes", themeName)
	if err := ensureDirectory(src); err != nil {
		return fmt.Errorf("theme %q not found in source: %w", themeName, err)
	}

	dst, err := themePath(themeName)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp(sddmThemesDir, ".pixelysia-split-")
	if err != nil {
		return fmt.Errorf("create temp split theme directory: %w", err)
	}
	cleanupTmp := true
	defer func() {
		if cleanupTmp {
			_ = os.RemoveAll(tmpDir)
		}
	}()

	if err := copyDir(src, tmpDir); err != nil {
		return err
	}
	if err := setOwnershipAndModeRecursive(tmpDir, 0, 0, 0o755, 0o755); err != nil {
		return err
	}

	if err := replaceDirAtomic(tmpDir, dst); err != nil {
		return err
	}
	cleanupTmp = false
	return nil
}

func ListThemes(out io.Writer) error {
	entries, err := os.ReadDir(sddmThemesDir)
	if err != nil {
		return fmt.Errorf("read installed themes: %w", err)
	}

	names := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}

	sort.Strings(names)
	for _, name := range names {
		if _, err := fmt.Fprintln(out, name); err != nil {
			return err
		}
	}
	return nil
}

func RemoveTheme(name string) error {
	if err := validateThemeName(name); err != nil {
		return err
	}

	path, err := themePath(name)
	if err != nil {
		return err
	}

	if err := ensureDirectory(path); err != nil {
		return fmt.Errorf("theme %q is not installed", name)
	}

	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove theme %q: %w", name, err)
	}
	return nil
}

func discoverSourceThemes(srcRoot string) ([]string, error) {
	dir := filepath.Join(srcRoot, "themes")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read source themes: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if err := validateThemeName(e.Name()); err != nil {
			continue
		}
		names = append(names, e.Name())
	}

	if len(names) == 0 {
		return nil, errors.New("no themes found in source themes directory")
	}

	sort.Strings(names)
	return names, nil
}

func detectSourceRoot() (string, error) {
	candidates := make([]string, 0, 3)

	if env := strings.TrimSpace(os.Getenv("PIXELYSIA_SOURCE_DIR")); env != "" {
		candidates = append(candidates, env)
	}

	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd)
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates, exeDir, filepath.Clean(filepath.Join(exeDir, "..")))
	}

	seen := make(map[string]struct{})
	for _, c := range candidates {
		if c == "" {
			continue
		}
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}

		if err := ensureRegularFile(filepath.Join(abs, "Main.qml")); err != nil {
			continue
		}
		if err := ensureDirectory(filepath.Join(abs, "themes")); err != nil {
			continue
		}
		if err := ensureDirectory(filepath.Join(abs, "fonts")); err != nil {
			continue
		}
		return abs, nil
	}

	return "", errors.New("unable to locate Pixelysia source directory; run from repository root or set PIXELYSIA_SOURCE_DIR")
}
