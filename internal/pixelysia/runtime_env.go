package pixelysia

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	envThemesDir  = "PIXELYSIA_THEMES_DIR"
	envFontDir    = "PIXELYSIA_FONT_DIR"
	envConfigDir  = "PIXELYSIA_SDDM_CONFIG_DIR"
	envConfigPath = "PIXELYSIA_SDDM_CONFIG_PATH"
	envReqUID     = "PIXELYSIA_REQUIRED_UID"
	envReqGID     = "PIXELYSIA_REQUIRED_GID"
)

func applyRuntimeEnvOverrides() {
	if v := strings.TrimSpace(os.Getenv(envThemesDir)); v != "" {
		sddmThemesDir = v
	}
	if v := strings.TrimSpace(os.Getenv(envFontDir)); v != "" {
		fontDir = v
	}

	configPathOverridden := false
	if v := strings.TrimSpace(os.Getenv(envConfigPath)); v != "" {
		sddmConfigPath = v
		sddmConfigDir = filepath.Dir(v)
		configPathOverridden = true
	}
	if v := strings.TrimSpace(os.Getenv(envConfigDir)); v != "" {
		sddmConfigDir = v
		if !configPathOverridden {
			sddmConfigPath = filepath.Join(v, "theme.conf")
		}
	}

	if v := strings.TrimSpace(os.Getenv(envReqUID)); v != "" {
		if uid, err := strconv.Atoi(v); err == nil {
			requiredUID = uid
		}
	}
	if v := strings.TrimSpace(os.Getenv(envReqGID)); v != "" {
		if gid, err := strconv.Atoi(v); err == nil {
			requiredGID = gid
		}
	}
}
