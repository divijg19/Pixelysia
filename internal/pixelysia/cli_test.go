package pixelysia

import (
	"bytes"
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
