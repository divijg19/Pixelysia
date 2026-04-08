package pixelysia

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyRuntimeEnvOverrides(t *testing.T) {
	setupTestGlobals(t)

	root := t.TempDir()
	themes := filepath.Join(root, "themes")
	fonts := filepath.Join(root, "fonts")
	cfgDir := filepath.Join(root, "conf")
	cfgPath := filepath.Join(cfgDir, "theme.conf")

	if err := os.Setenv(envThemesDir, themes); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv(envFontDir, fonts); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv(envConfigPath, cfgPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv(envReqUID, "123"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv(envReqGID, "456"); err != nil {
		t.Fatal(err)
	}

	applyRuntimeEnvOverrides()

	if sddmThemesDir != themes {
		t.Fatalf("unexpected themes dir: %s", sddmThemesDir)
	}
	if fontDir != fonts {
		t.Fatalf("unexpected fonts dir: %s", fontDir)
	}
	if sddmConfigPath != cfgPath {
		t.Fatalf("unexpected config path: %s", sddmConfigPath)
	}
	if sddmConfigDir != cfgDir {
		t.Fatalf("unexpected config dir: %s", sddmConfigDir)
	}
	if requiredUID != 123 || requiredGID != 456 {
		t.Fatalf("unexpected uid/gid: %d:%d", requiredUID, requiredGID)
	}
}
