package pixelysia

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	requiredUID   = 0
	requiredGID   = 0
	commandRunner = exec.Command
)

func copyFile(src string, dst string, mode os.FileMode) error {
	if err := ensureRegularFile(src); err != nil {
		return fmt.Errorf("validate source file %s: %w", src, err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create destination directory for %s: %w", dst, err)
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("open destination file %s: %w", dst, err)
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copy %s to %s: %w", src, dst, err)
	}

	if err := out.Sync(); err != nil {
		out.Close()
		return fmt.Errorf("sync destination file %s: %w", dst, err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("close destination file %s: %w", dst, err)
	}

	return nil
}

func copyDir(src string, dst string) error {
	if err := ensureDirectory(src); err != nil {
		return fmt.Errorf("validate source directory %s: %w", src, err)
	}

	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("create destination directory %s: %w", dst, err)
	}

	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if strings.HasPrefix(rel, "..") {
			return fmt.Errorf("invalid relative path during copy: %s", rel)
		}

		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are not supported in install sources: %s", path)
		}

		if !d.Type().IsRegular() {
			return fmt.Errorf("unsupported file type in install sources: %s", path)
		}

		return copyFile(path, target, 0o644)
	})
	if err != nil {
		return fmt.Errorf("copy directory %s to %s: %w", src, dst, err)
	}

	return nil
}

func setOwnershipAndModeRecursive(root string, uid int, gid int, fileMode os.FileMode, dirMode os.FileMode) error {
	if err := ensureDirectory(root); err != nil {
		return err
	}

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if err := os.Chown(path, uid, gid); err != nil {
			return fmt.Errorf("set ownership on %s failed; rerun with sudo: %w", path, err)
		}

		mode := fileMode
		if d.IsDir() {
			mode = dirMode
		}

		if err := os.Chmod(path, mode); err != nil {
			return fmt.Errorf("set permissions on %s: %w", path, err)
		}

		return nil
	})
}

func replaceDirAtomic(tmpDir string, dest string) error {
	if err := ensureDirectory(tmpDir); err != nil {
		return fmt.Errorf("validate temporary directory for replace: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("create destination parent directory %s: %w", filepath.Dir(dest), err)
	}

	backupPath := ""
	if _, err := os.Stat(dest); err == nil {
		backupPath = filepath.Join(filepath.Dir(dest), "."+filepath.Base(dest)+".backup-"+strconv.FormatInt(time.Now().UnixNano(), 10))
		if err := os.Rename(dest, backupPath); err != nil {
			return fmt.Errorf("move existing destination to backup: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat destination %s: %w", dest, err)
	}

	if err := moveDir(tmpDir, dest); err != nil {
		if backupPath != "" {
			_ = os.Rename(backupPath, dest)
		}
		return fmt.Errorf("move temporary directory into place: %w", err)
	}

	if backupPath != "" {
		if err := os.RemoveAll(backupPath); err != nil {
			return fmt.Errorf("remove destination backup: %w", err)
		}
	}

	return nil
}

func moveDir(src string, dst string) error {
	if err := os.Rename(src, dst); err != nil {
		var linkErr *os.LinkError
		if errors.As(err, &linkErr) && errors.Is(linkErr.Err, syscall.EXDEV) {
			if err := copyDir(src, dst); err != nil {
				return err
			}
			if err := os.RemoveAll(src); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	return nil
}

func validateThemeName(name string) error {
	if name == "" {
		return errors.New("theme name cannot be empty")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("invalid theme name %q", name)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("invalid theme name %q", name)
	}

	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return fmt.Errorf("invalid theme name %q", name)
	}
	return nil
}

func themePath(name string) (string, error) {
	if err := validateThemeName(name); err != nil {
		return "", err
	}
	p := filepath.Clean(filepath.Join(sddmThemesDir, name))
	prefix := sddmThemesDir + string(os.PathSeparator)
	if !strings.HasPrefix(p, prefix) {
		return "", errors.New("resolved theme path escapes themes directory")
	}
	return p, nil
}

func ensureDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

func ensureRegularFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", path)
	}
	return nil
}

type doctorCheck struct {
	Name   string
	OK     bool
	Detail string
}

func RunDoctor(out io.Writer) error {
	checks := make([]doctorCheck, 0, 5)

	checks = append(checks, checkFontsInstalled())
	checks = append(checks, checkFontCache())
	checks = append(checks, checkThemesPresent())
	checks = append(checks, checkConfigExists())
	checks = append(checks, checkThemePermissions())

	failed := false
	for _, c := range checks {
		status := "OK"
		if !c.OK {
			status = "FAIL"
			failed = true
		}
		if c.Detail == "" {
			if _, err := fmt.Fprintf(out, "[%s] %s\n", status, c.Name); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(out, "[%s] %s: %s\n", status, c.Name, c.Detail); err != nil {
			return err
		}
	}

	if failed {
		return errors.New("doctor found one or more issues")
	}
	return nil
}

func checkFontsInstalled() doctorCheck {
	entries, err := os.ReadDir(fontDir)
	if err != nil {
		return doctorCheck{Name: "fonts installed", OK: false, Detail: err.Error()}
	}

	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".ttf") {
			count++
		}
	}

	if count == 0 {
		return doctorCheck{Name: "fonts installed", OK: false, Detail: "no .ttf files found"}
	}
	return doctorCheck{Name: "fonts installed", OK: true, Detail: fmt.Sprintf("%d font(s) found", count)}
}

func checkFontCache() doctorCheck {
	cmd := commandRunner("fc-cache", "-f")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return doctorCheck{Name: "font cache valid", OK: false, Detail: err.Error()}
	}
	return doctorCheck{Name: "font cache valid", OK: true}
}

func checkThemesPresent() doctorCheck {
	entries, err := os.ReadDir(sddmThemesDir)
	if err != nil {
		return doctorCheck{Name: "themes present", OK: false, Detail: err.Error()}
	}

	names := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}

	if len(names) == 0 {
		return doctorCheck{Name: "themes present", OK: false, Detail: "no installed themes found"}
	}

	sort.Strings(names)
	return doctorCheck{Name: "themes present", OK: true, Detail: strings.Join(names, ", ")}
}

func checkConfigExists() doctorCheck {
	if err := ensureRegularFile(sddmConfigPath); err != nil {
		return doctorCheck{Name: "config file exists", OK: false, Detail: err.Error()}
	}
	return doctorCheck{Name: "config file exists", OK: true}
}

func checkThemePermissions() doctorCheck {
	entries, err := os.ReadDir(sddmThemesDir)
	if err != nil {
		return doctorCheck{Name: "theme permissions", OK: false, Detail: err.Error()}
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		path := filepath.Join(sddmThemesDir, e.Name())
		err := filepath.WalkDir(path, func(itemPath string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			stat, ok := info.Sys().(*syscall.Stat_t)
			if !ok {
				return errors.New("unable to read ownership metadata")
			}

			if int(stat.Uid) != requiredUID || int(stat.Gid) != requiredGID {
				return fmt.Errorf("%s is owned by %d:%d (expected %d:%d)", itemPath, stat.Uid, stat.Gid, requiredUID, requiredGID)
			}

			expectedMode := os.FileMode(0o644)
			if d.IsDir() {
				expectedMode = 0o755
			}

			if info.Mode().Perm() != expectedMode {
				return fmt.Errorf("%s has mode %o (expected %o)", itemPath, info.Mode().Perm(), expectedMode)
			}

			return nil
		})
		if err != nil {
			return doctorCheck{Name: "theme permissions", OK: false, Detail: err.Error()}
		}
	}

	return doctorCheck{Name: "theme permissions", OK: true}
}
