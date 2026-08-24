package pixelysia

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestCLIInstallRejectsMutuallyExclusiveFlags(t *testing.T) {
	setupTestGlobals(t)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cli := NewCLI(&out, &errOut)

	code := cli.Run([]string{"install", "--split", "--theme", "alpha"})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(errOut.String(), "cannot be used together") {
		t.Fatalf("expected mutual exclusion error, got: %s", errOut.String())
	}
}

func TestCLIMissingSetArgument(t *testing.T) {
	setupTestGlobals(t)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cli := NewCLI(&out, &errOut)

	code := cli.Run([]string{"set"})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(errOut.String(), "set requires exactly one theme name") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestCLIMissingRemoveArgument(t *testing.T) {
	setupTestGlobals(t)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cli := NewCLI(&out, &errOut)

	code := cli.Run([]string{"remove"})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(errOut.String(), "remove requires exactly one theme name") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestCLIUnknownCommand(t *testing.T) {
	setupTestGlobals(t)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cli := NewCLI(&out, &errOut)

	code := cli.Run([]string{"unknown"})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(errOut.String(), "unknown command") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestCLIInstallRejectsPositionalArguments(t *testing.T) {
	setupTestGlobals(t)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cli := NewCLI(&out, &errOut)

	code := cli.Run([]string{"install", "extra"})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(errOut.String(), "does not accept positional arguments") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestCLIListRejectsArguments(t *testing.T) {
	setupTestGlobals(t)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cli := NewCLI(&out, &errOut)

	code := cli.Run([]string{"list", "extra"})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(errOut.String(), "list does not accept arguments") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestCLICurrentRejectsArguments(t *testing.T) {
	setupTestGlobals(t)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cli := NewCLI(&out, &errOut)

	code := cli.Run([]string{"current", "extra"})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(errOut.String(), "current does not accept arguments") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestCLIDoctorRejectsArguments(t *testing.T) {
	setupTestGlobals(t)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cli := NewCLI(&out, &errOut)

	code := cli.Run([]string{"doctor", "extra"})
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(errOut.String(), "doctor does not accept arguments") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestCLIHelpReturnsSuccess(t *testing.T) {
	setupTestGlobals(t)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cli := NewCLI(&out, &errOut)

	code := cli.Run([]string{"help"})
	if code != 0 {
		t.Fatal("expected zero exit code for help")
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("expected usage output, got: %s", out.String())
	}
}

func TestCLIValidateRejectsPositionalArguments(t *testing.T) {
	setupTestGlobals(t)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cli := NewCLI(&out, &errOut)

	code := cli.Run([]string{"validate", "extra"})
	if code == 0 {
		t.Fatal("expected non-zero exit code for positional argument")
	}
	if !strings.Contains(errOut.String(), "does not accept positional arguments") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestCLIValidateInvalidSourceExitsNonZero(t *testing.T) {
	setupTestGlobals(t)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cli := NewCLI(&out, &errOut)

	code := cli.Run([]string{"validate", "--source", "/definitely/not/a/pixelysia/tree"})
	if code == 0 {
		t.Fatal("expected non-zero exit code for invalid source")
	}
}

func TestCLIValidateSourceFixture(t *testing.T) {
	setupTestGlobals(t)

	root, addTheme := createValidateFixture(t)
	addTheme("forest")
	writeDispatcher(t, root, "themes/forest/Main.qml")

	var out bytes.Buffer
	var errOut bytes.Buffer
	cli := NewCLI(&out, &errOut)

	code := cli.Run([]string{"validate", "--source", root})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "[OK] forest") || !strings.Contains(out.String(), "1 ok") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestCLIValidateUsesDetectedSourceRoot(t *testing.T) {
	setupTestGlobals(t)

	srcRoot := createNestedSourceTree(t)
	if err := os.Setenv(sourceDirEnv, srcRoot); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	cli := NewCLI(&out, &errOut)

	code := cli.Run([]string{"validate"})
	if code != 0 {
		t.Fatalf("expected exit 0 via detected source root, got %d\nstderr=%s", code, errOut.String())
	}
	for _, want := range []string{"forest", "tui/Amber", "tui/Emerald"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out.String())
		}
	}
}

func TestCLIVersionCommand(t *testing.T) {
	setupTestGlobals(t)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cli := NewCLI(&out, &errOut)

	code := cli.Run([]string{"version"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr=%s", code, errOut.String())
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Fatal("expected version output")
	}

	code = cli.Run([]string{"version", "extra"})
	if code == 0 {
		t.Fatal("expected non-zero exit for positional argument")
	}
}
