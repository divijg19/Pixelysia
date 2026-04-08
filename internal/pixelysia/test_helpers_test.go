package pixelysia

import (
	"os"
	"os/exec"
	"testing"
)

type savedGlobals struct {
	sddmThemesDir  string
	fontDir        string
	sddmConfigDir  string
	sddmConfigPath string
	requiredUID    int
	requiredGID    int
	commandRunner  func(string, ...string) *exec.Cmd
}

func setupTestGlobals(t *testing.T) {
	t.Helper()

	saved := savedGlobals{
		sddmThemesDir:  sddmThemesDir,
		fontDir:        fontDir,
		sddmConfigDir:  sddmConfigDir,
		sddmConfigPath: sddmConfigPath,
		requiredUID:    requiredUID,
		requiredGID:    requiredGID,
		commandRunner:  commandRunner,
	}

	requiredUID = os.Getuid()
	requiredGID = os.Getgid()
	commandRunner = exec.Command

	t.Cleanup(func() {
		sddmThemesDir = saved.sddmThemesDir
		fontDir = saved.fontDir
		sddmConfigDir = saved.sddmConfigDir
		sddmConfigPath = saved.sddmConfigPath
		requiredUID = saved.requiredUID
		requiredGID = saved.requiredGID
		commandRunner = saved.commandRunner
		_ = os.Unsetenv(sourceDirEnv)
	})
}
