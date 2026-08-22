package pixelysia

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	fullThemeName = "pixelysia"
	sourceDirEnv  = "PIXELYSIA_SOURCE_DIR"
)

var (
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

	// Discover and validate the complete theme set before touching the
	// system so that a broken source tree cannot leave a partially
	// completed installation behind.
	var themeNames []string
	switch {
	case opts.Split:
		themeNames, err = discoverSourceThemes(srcRoot)
		if err != nil {
			return err
		}
	case opts.Theme != "":
		if err := validateThemeName(opts.Theme); err != nil {
			return fmt.Errorf("invalid theme %q: %w", opts.Theme, err)
		}
		themeNames = []string{opts.Theme}
	default:
		// Full mode validates every discovered theme up front as well;
		// installation itself copies the whole themes tree.
		themeNames, err = discoverSourceThemes(srcRoot)
		if err != nil {
			return err
		}
	}

	for _, name := range themeNames {
		src := filepath.Join(srcRoot, "themes", filepath.FromSlash(name))
		if err := validateThemeSource(src); err != nil {
			return fmt.Errorf("validate source theme %q: %w", name, err)
		}
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

	tmpDir, err := os.MkdirTemp("", ".pixelysia-fonts-")
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
	if err := os.Chown(tmpDir, requiredUID, requiredGID); err != nil {
		return fmt.Errorf("set temp fonts ownership: %w", err)
	}

	for _, src := range fonts {
		dst := filepath.Join(tmpDir, filepath.Base(src))
		if err := copyFile(src, dst, 0o644); err != nil {
			return err
		}
		if err := os.Chown(dst, requiredUID, requiredGID); err != nil {
			return fmt.Errorf("set font ownership %s: %w", dst, err)
		}
	}

	if err := replaceDirAtomic(tmpDir, fontDir); err != nil {
		return err
	}
	cleanupTmp = false

	cmd := commandRunner("fc-cache", "-f")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rebuild font cache failed; ensure fc-cache is installed and rerun with sudo: %w", err)
	}

	return nil
}

func installFullTheme(srcRoot string) error {
	tmpDir, err := os.MkdirTemp("", ".pixelysia-theme-")
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

	// All source themes were discovered and validated by Install before
	// this function was reached.
	if err := copyDir(filepath.Join(srcRoot, "themes"), filepath.Join(tmpDir, "themes")); err != nil {
		return err
	}
	if err := copyDir(filepath.Join(srcRoot, "fonts"), filepath.Join(tmpDir, "fonts")); err != nil {
		return err
	}

	if err := setOwnershipAndModeRecursive(tmpDir, requiredUID, requiredGID, 0o644, 0o755); err != nil {
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
	if err := validateThemeSource(src); err != nil {
		return fmt.Errorf("theme %q not found in source: %w", themeName, err)
	}

	dst, err := themePath(themeName)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", ".pixelysia-split-")
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
	if err := setOwnershipAndModeRecursive(tmpDir, requiredUID, requiredGID, 0o644, 0o755); err != nil {
		return err
	}

	if err := replaceDirAtomic(tmpDir, dst); err != nil {
		return err
	}
	cleanupTmp = false
	return nil
}

// ListThemes prints the identifiers of all installed themes, sorted
// lexically. Detection mirrors source discovery: a directory counts as an
// installed theme when it directly contains Main.qml or metadata.desktop,
// and container directories are searched recursively so that nested split
// installations are reported with their canonical identifiers ("tui/Amber").
func ListThemes(out io.Writer) error {
	if err := ensureDirectory(sddmThemesDir); err != nil {
		return fmt.Errorf("read installed themes: %w", err)
	}

	hasThemeMarker := func(dir string) bool {
		for _, marker := range []string{"Main.qml", "metadata.desktop"} {
			info, err := os.Stat(filepath.Join(dir, marker))
			if err == nil && info.Mode().IsRegular() {
				return true
			}
		}
		return false
	}

	names := make([]string, 0)
	walkErr := filepath.WalkDir(sddmThemesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(sddmThemesDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return fs.SkipDir
		}

		if hasThemeMarker(path) {
			names = append(names, filepath.ToSlash(rel))
			// Do not report themes bundled inside another theme.
			return fs.SkipDir
		}
		return nil
	})
	if walkErr != nil {
		return fmt.Errorf("read installed themes: %w", walkErr)
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

// discoverSourceThemes walks srcRoot/themes recursively and returns the
// identifiers of every discoverable theme, sorted lexically. Discovery is
// structural, not depth-based: a directory is a theme when it directly
// contains Main.qml; any other directory is treated as a container and its
// children are searched. This supports both flat layouts ("themes/forest")
// and nested category layouts ("themes/tui/Amber"). Identifiers use '/'
// separators relative to the themes directory (e.g. "tui/Amber").
//
// A directory that contains metadata.desktop but no Main.qml is reported as
// malformed instead of being silently skipped, so that broken themes fail
// discovery rather than disappearing from installations.
func discoverSourceThemes(srcRoot string) ([]string, error) {
	root := filepath.Join(srcRoot, "themes")
	if err := ensureDirectory(root); err != nil {
		return nil, fmt.Errorf("read source themes: %w", err)
	}

	hasRegularFile := func(dir string, name string) bool {
		info, err := os.Stat(filepath.Join(dir, name))
		return err == nil && info.Mode().IsRegular()
	}

	names := make([]string, 0)
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		id := filepath.ToSlash(rel)

		// Skip hidden directories such as .git; they can never be themes.
		if rel != "." && strings.HasPrefix(d.Name(), ".") {
			return fs.SkipDir
		}

		if rel != "." {
			if err := validateThemeName(id); err != nil {
				return fmt.Errorf("invalid theme directory %q: %w", id, err)
			}
		}

		if hasRegularFile(path, "Main.qml") {
			if rel == "." {
				return errors.New("themes directory itself must not contain Main.qml")
			}
			names = append(names, id)
			// A theme is a leaf: do not search inside it.
			return fs.SkipDir
		}

		if hasRegularFile(path, "metadata.desktop") {
			return fmt.Errorf("malformed theme directory %q: missing required Main.qml", id)
		}

		// Container/category directory: keep searching its children.
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("read source themes: %w", walkErr)
	}

	if len(names) == 0 {
		return nil, errors.New("no themes found in source themes directory")
	}

	sort.Strings(names)
	return names, nil
}

func detectSourceRoot() (string, error) {
	if env := strings.TrimSpace(os.Getenv(sourceDirEnv)); env != "" {
		abs, err := filepath.Abs(env)
		if err != nil {
			return "", fmt.Errorf("resolve %s: %w", sourceDirEnv, err)
		}
		if err := validateSourceRoot(abs); err != nil {
			return "", fmt.Errorf("invalid %s=%q: %w", sourceDirEnv, env, err)
		}
		return abs, nil
	}

	candidates := make([]string, 0, 2)

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

		if err := validateSourceRoot(abs); err != nil {
			continue
		}
		return abs, nil
	}

	return "", errors.New("unable to locate Pixelysia source directory; run from the repository root or set PIXELYSIA_SOURCE_DIR to a directory containing Main.qml, themes/, and fonts/")
}

func validateSourceRoot(root string) error {
	if err := ensureRegularFile(filepath.Join(root, "Main.qml")); err != nil {
		return fmt.Errorf("missing dispatcher Main.qml: %w", err)
	}
	if err := ensureDirectory(filepath.Join(root, "themes")); err != nil {
		return fmt.Errorf("missing themes directory: %w", err)
	}
	if err := ensureDirectory(filepath.Join(root, "fonts")); err != nil {
		return fmt.Errorf("missing fonts directory: %w", err)
	}
	return nil
}

func validateThemeSource(themeDir string) error {
	if err := ensureDirectory(themeDir); err != nil {
		return err
	}
	if err := ensureRegularFile(filepath.Join(themeDir, "Main.qml")); err != nil {
		return fmt.Errorf("missing required Main.qml: %w", err)
	}
	return nil
}
