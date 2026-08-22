package pixelysia

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// validationFinding is a single result reported by the source validator.
// Fatal findings make validation fail; informational findings are reported
// without affecting the exit status.
type validationFinding struct {
	fatal bool
	msg   string
}

// dispatcherRefRe matches quoted Pixelysia theme references inside the root
// dispatcher, e.g. "themes/tui/Amber/Main.qml".
var dispatcherRefRe = regexp.MustCompile(`"(themes/[^"]+)"`)

// ValidateSource performs a read-only validation of a Pixelysia source tree.
// It never mutates the system and is safe to run unprivileged. Findings are
// written to out; the returned error is non-nil when at least one fatal
// finding was recorded or when the tree cannot be validated at all.
//
// Validation scope (v0.5.0):
//   - structural validity via the canonical theme classifier
//   - metadata.desktop presence (informational when missing)
//   - declared background= assets resolve to real files
//   - declared font= families are represented in the theme's QML usage
//     (textual/semantic check; it does not parse TTF name tables or prove
//     that a runtime font engine will resolve the family)
//   - every theme reference in the root dispatcher resolves to a file
func ValidateSource(srcRoot string, out io.Writer) error {
	if err := validateSourceRoot(srcRoot); err != nil {
		return err
	}

	themes, err := collectSourceThemes(srcRoot)
	if err != nil {
		return err
	}

	fatals := 0
	informative := 0
	ok := 0

	emit := func(finding validationFinding) {
		switch {
		case finding.fatal:
			fatals++
			fmt.Fprintf(out, "[FAIL] %s\n", finding.msg)
		default:
			informative++
			fmt.Fprintf(out, "[INFO] %s\n", finding.msg)
		}
	}

	for _, t := range themes {
		findings := checkDiscoveredTheme(t)
		if len(findings) == 0 {
			ok++
			fmt.Fprintf(out, "[OK] %s\n", t.ID)
			continue
		}
		for _, f := range findings {
			f.msg = t.ID + ": " + f.msg
			emit(f)
		}
	}

	dispatcherFindings := validateDispatcher(srcRoot)
	if len(dispatcherFindings) == 0 {
		fmt.Fprintf(out, "[OK] dispatcher references resolved\n")
	}
	for _, f := range dispatcherFindings {
		emit(f)
	}

	summary := fmt.Sprintf("validated %d themes: %d ok, %d failed, %d informational",
		len(themes), ok, fatals, informative)
	if _, err := fmt.Fprintln(out, summary); err != nil {
		return err
	}

	if fatals > 0 {
		return errors.New("theme validation failed")
	}
	return nil
}

// validateThemeSource checks a single discovered theme and returns its
// findings.
func checkDiscoveredTheme(t Theme) []validationFinding {
	var findings []validationFinding

	if !hasRegularFile(t.Path, "metadata.desktop") {
		findings = append(findings, validationFinding{
			fatal: false,
			msg:   "no metadata.desktop (advisory)",
		})
	}

	if bg := t.Conf["background"]; bg != "" {
		if !hasRegularFile(t.Path, bg) {
			findings = append(findings, validationFinding{
				fatal: true,
				msg:   fmt.Sprintf("background=%s does not exist", bg),
			})
		}
	}

	if font := t.Conf["font"]; font != "" {
		qmlPath := filepath.Join(t.Path, "Main.qml")
		qml, err := os.ReadFile(qmlPath)
		if err != nil {
			findings = append(findings, validationFinding{
				fatal: true,
				msg:   fmt.Sprintf("read Main.qml to verify font=%s: %v", font, err),
			})
		} else if !qmlMentionsFont(string(qml), font) {
			findings = append(findings, validationFinding{
				fatal: true,
				msg:   fmt.Sprintf("font=%s is not referenced in Main.qml", font),
			})
		}
	}

	return findings
}

// qmlMentionsFont reports whether a font family declaration is represented
// in a theme's QML usage. Comparison is textual and whitespace/case
// insensitive so that conventions such as `font=PixelifySans` match QML
// `font.family: "Pixelify Sans"`. This intentionally does not parse TTF
// name tables and does not prove runtime font resolution.
func qmlMentionsFont(qml string, family string) bool {
	normalize := func(s string) string {
		s = strings.ToLower(s)
		return strings.ReplaceAll(s, " ", "")
	}
	return strings.Contains(normalize(qml), normalize(family))
}

// validateDispatcher verifies that every quoted theme reference inside the
// root dispatcher Main.qml resolves to a real file in the source tree.
func validateDispatcher(srcRoot string) []validationFinding {
	qml, err := os.ReadFile(filepath.Join(srcRoot, "Main.qml"))
	if err != nil {
		return []validationFinding{{
			fatal: true,
			msg:   fmt.Sprintf("read dispatcher Main.qml: %v", err),
		}}
	}

	seen := make(map[string]bool)
	var findings []validationFinding
	for _, match := range dispatcherRefRe.FindAllStringSubmatch(string(qml), -1) {
		ref := match[1]
		if seen[ref] {
			continue
		}
		seen[ref] = true

		target := filepath.Join(srcRoot, filepath.FromSlash(ref))
		info, statErr := os.Stat(target)
		if statErr != nil || !info.Mode().IsRegular() {
			findings = append(findings, validationFinding{
				fatal: true,
				msg:   fmt.Sprintf("dispatcher reference %q does not resolve", ref),
			})
		}
	}
	return findings
}
