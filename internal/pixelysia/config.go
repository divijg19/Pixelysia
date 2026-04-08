package pixelysia

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	sddmConfigDir  = "/etc/sddm.conf.d"
	sddmConfigPath = "/etc/sddm.conf.d/theme.conf"
)

func SetTheme(theme string) error {
	if err := validateThemeName(theme); err != nil {
		return err
	}

	themeInstallPath, err := themePath(theme)
	if err != nil {
		return err
	}
	if err := ensureDirectory(themeInstallPath); err != nil {
		return fmt.Errorf("theme %q is not installed", theme)
	}

	if err := os.MkdirAll(sddmConfigDir, 0o755); err != nil {
		return fmt.Errorf("create SDDM config directory: %w", err)
	}

	existing := []byte{}
	if b, err := os.ReadFile(sddmConfigPath); err == nil {
		existing = b
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read %s: %w", sddmConfigPath, err)
	}

	updated := updateThemeConfig(existing, theme)
	if err := writeFileAtomic(sddmConfigPath, updated, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", sddmConfigPath, err)
	}
	return nil
}

func CurrentTheme() (string, error) {
	b, err := os.ReadFile(sddmConfigPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", sddmConfigPath, err)
	}

	theme, ok := parseCurrentTheme(b)
	if !ok {
		return "", errors.New("Current theme is not configured in /etc/sddm.conf.d/theme.conf")
	}
	return theme, nil
}

func updateThemeConfig(existing []byte, theme string) []byte {
	normalized := strings.ReplaceAll(string(existing), "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	out := make([]string, 0, len(lines)+3)

	inTheme := false
	foundTheme := false
	wroteCurrent := false

	for _, line := range lines {
		if section, ok := parseSectionLine(line); ok {
			if inTheme && !wroteCurrent {
				out = append(out, "Current="+theme)
				wroteCurrent = true
			}

			inTheme = strings.EqualFold(section, "Theme")
			if inTheme {
				foundTheme = true
				wroteCurrent = false
			}

			out = append(out, line)
			continue
		}

		if inTheme && isCurrentSettingLine(line) {
			if !wroteCurrent {
				out = append(out, "Current="+theme)
				wroteCurrent = true
			}
			continue
		}

		out = append(out, line)
	}

	if inTheme && !wroteCurrent {
		out = append(out, "Current="+theme)
	}

	if !foundTheme {
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
			out = append(out, "")
		}
		out = append(out, "[Theme]", "Current="+theme)
	}

	return []byte(strings.Join(out, "\n"))
}

func parseCurrentTheme(data []byte) (string, bool) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	inTheme := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if section, ok := parseSectionLine(line); ok {
			inTheme = strings.EqualFold(section, "Theme")
			continue
		}

		if !inTheme {
			continue
		}

		if !isCurrentSettingLine(line) {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		value := strings.TrimSpace(parts[1])
		if value == "" {
			continue
		}
		return value, true
	}
	return "", false
}

func parseSectionLine(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
		return "", false
	}
	name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]"))
	if name == "" {
		return "", false
	}
	return name, true
}

func isCurrentSettingLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	parts := strings.SplitN(trimmed, "=", 2)
	if len(parts) != 2 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(parts[0]), "Current")
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".pixelysia-config-")
	if err != nil {
		return err
	}

	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		_ = tmp.Close()
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		return err
	}
	if err := tmp.Chown(0, 0); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}

	cleanup = false
	return nil
}
