package pixelysia

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Theme is the canonical representation of a discovered theme.
type Theme struct {
	// ID is the slash-separated identifier relative to the themes root,
	// e.g. "forest" or "tui/Amber".
	ID string
	// Path is the absolute theme directory.
	Path string
	// Conf holds the parsed theme.conf. It is nil when the theme does not
	// declare one; the configuration file is optional in the theme contract.
	Conf map[string]string
}

// hasRegularFile reports whether dir contains a regular file with the given
// name.
func hasRegularFile(dir string, name string) bool {
	info, err := os.Stat(filepath.Join(dir, name))
	return err == nil && info.Mode().IsRegular()
}

// walkThemeTree is the single canonical traversal for theme trees. It walks
// root recursively and invokes visit once for every directory that contains
// a theme entrypoint (Main.qml) or theme metadata (metadata.desktop).
//
// Traversal policy:
//   - hidden directories are skipped entirely
//   - directories whose relative identifier fails validateThemeName are an
//     error (they can never be addressed by the CLI)
//   - any directory carrying either marker is treated as a leaf: visit is
//     invoked exactly once for it and the walk does not descend further.
//     Callers apply their own policies to (hasMain, hasMeta), which keeps
//     source discovery strict while installed listing stays lenient without
//     duplicating the traversal itself.
//   - directories carrying neither marker are containers and are searched.
//   - the root directory itself is never reported as a theme; callers that
//     need to reject misplaced markers must check for them separately.
func walkThemeTree(root string, visit func(id string, path string, hasMain bool, hasMeta bool) error) error {
	if err := ensureDirectory(root); err != nil {
		return err
	}

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
		if rel == "." {
			return nil
		}
		id := filepath.ToSlash(rel)

		if strings.HasPrefix(d.Name(), ".") {
			return fs.SkipDir
		}
		if err := validateThemeName(id); err != nil {
			return fmt.Errorf("invalid theme directory %q: %w", id, err)
		}

		hasMain := hasRegularFile(path, "Main.qml")
		hasMeta := hasRegularFile(path, "metadata.desktop")
		if !hasMain && !hasMeta {
			return nil // container directory: keep searching its children
		}

		if err := visit(id, path, hasMain, hasMeta); err != nil {
			return err
		}
		// A marked directory is a leaf: do not search inside it.
		return fs.SkipDir
	})
}

// collectSourceThemes discovers every installable theme below
// srcRoot/themes using the canonical classifier and returns them sorted by
// identifier. Source semantics are strict: a theme requires Main.qml, and a
// directory carrying metadata.desktop without Main.qml is malformed rather
// than silently skipped.
func collectSourceThemes(srcRoot string) ([]Theme, error) {
	root := filepath.Join(srcRoot, "themes")

	// Markers directly inside the themes root are never themes.
	if hasRegularFile(root, "Main.qml") {
		return nil, errors.New("themes directory itself must not contain Main.qml")
	}
	if hasRegularFile(root, "metadata.desktop") {
		return nil, fmt.Errorf(`malformed theme directory ".": missing required Main.qml`)
	}

	themes := make([]Theme, 0)
	err := walkThemeTree(root, func(id string, path string, hasMain bool, hasMeta bool) error {
		if !hasMain {
			return fmt.Errorf("malformed theme directory %q: missing required Main.qml", id)
		}
		conf, confErr := readThemeConf(filepath.Join(path, "theme.conf"))
		if confErr != nil {
			return confErr
		}
		themes = append(themes, Theme{ID: id, Path: path, Conf: conf})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read source themes: %w", err)
	}
	if len(themes) == 0 {
		return nil, errors.New("no themes found in source themes directory")
	}

	sort.Slice(themes, func(i int, j int) bool { return themes[i].ID < themes[j].ID })
	return themes, nil
}

// collectInstalledThemes lists installed themes under sddmThemesDir using
// the canonical classifier. Installed semantics are deliberately lenient:
// Main.qml or metadata.desktop alone marks an installed theme, matching what
// SDDM can load. The full bundle counts as one theme; its internal themes
// are not reported separately because the walk stops at bundle leaves.
func collectInstalledThemes() ([]string, error) {
	names := make([]string, 0)
	err := walkThemeTree(sddmThemesDir, func(id string, _ string, _ bool, _ bool) error {
		names = append(names, id)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

// readThemeConf parses a Pixelysia theme configuration file into a flat
// key/value map. Sections are recognized but flattened; later duplicate
// keys override earlier ones; comments (#, ;) and blank lines are ignored.
// A missing file yields (nil, nil): theme.conf is optional in the theme
// contract. Any other read error, or a non-comment line that is neither a
// section header nor key=value, is an error rather than being swallowed.
func readThemeConf(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	out := make(map[string]string)
	for _, raw := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if _, ok := parseSectionLine(line); ok {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			return nil, fmt.Errorf("malformed %s: expected key=value, got %q", filepath.Base(path), line)
		}
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out, nil
}
