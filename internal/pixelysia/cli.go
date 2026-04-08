package pixelysia

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

type CLI struct {
	out io.Writer
	err io.Writer
}

func NewCLI(out io.Writer, err io.Writer) *CLI {
	return &CLI{out: out, err: err}
}

func (c *CLI) Run(args []string) int {
	applyRuntimeEnvOverrides()

	if len(args) == 0 {
		c.printUsage()
		return 1
	}

	switch args[0] {
	case "install":
		installFlags := flag.NewFlagSet("install", flag.ContinueOnError)
		installFlags.SetOutput(c.err)

		split := installFlags.Bool("split", false, "install themes in split mode")
		theme := installFlags.String("theme", "", "install a single theme by name")

		if err := installFlags.Parse(args[1:]); err != nil {
			c.printInstallUsage()
			return 1
		}

		if installFlags.NArg() != 0 {
			fmt.Fprintln(c.err, "error: install does not accept positional arguments")
			c.printInstallUsage()
			return 1
		}

		opts := InstallOptions{Split: *split, Theme: *theme}
		if err := Install(opts, c.out); err != nil {
			fmt.Fprintf(c.err, "error: %v\n", err)
			return 1
		}
		return 0

	case "set":
		if len(args) != 2 {
			fmt.Fprintln(c.err, "error: set requires exactly one theme name")
			c.printUsage()
			return 1
		}

		if err := SetTheme(args[1]); err != nil {
			fmt.Fprintf(c.err, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(c.out, "Theme set to %s\n", args[1])
		return 0

	case "list":
		if len(args) != 1 {
			fmt.Fprintln(c.err, "error: list does not accept arguments")
			c.printUsage()
			return 1
		}

		if err := ListThemes(c.out); err != nil {
			fmt.Fprintf(c.err, "error: %v\n", err)
			return 1
		}
		return 0

	case "current":
		if len(args) != 1 {
			fmt.Fprintln(c.err, "error: current does not accept arguments")
			c.printUsage()
			return 1
		}

		theme, err := CurrentTheme()
		if err != nil {
			fmt.Fprintf(c.err, "error: %v\n", err)
			return 1
		}
		fmt.Fprintln(c.out, theme)
		return 0

	case "remove":
		if len(args) != 2 {
			fmt.Fprintln(c.err, "error: remove requires exactly one theme name")
			c.printUsage()
			return 1
		}

		if err := RemoveTheme(args[1]); err != nil {
			fmt.Fprintf(c.err, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(c.out, "Removed theme %s\n", args[1])
		return 0

	case "doctor":
		if len(args) != 1 {
			fmt.Fprintln(c.err, "error: doctor does not accept arguments")
			c.printUsage()
			return 1
		}

		if err := RunDoctor(c.out); err != nil {
			fmt.Fprintf(c.err, "error: %v\n", err)
			return 1
		}
		return 0

	case "help", "-h", "--help":
		c.printUsage()
		return 0

	default:
		fmt.Fprintf(c.err, "error: unknown command %q\n", args[0])
		c.printUsage()
		return 1
	}
}

func (c *CLI) printUsage() {
	_, _ = fmt.Fprintln(c.out, "Usage:")
	_, _ = fmt.Fprintln(c.out, "  pixelysia install [--split | --theme <name>]")
	_, _ = fmt.Fprintln(c.out, "  pixelysia set <theme>")
	_, _ = fmt.Fprintln(c.out, "  pixelysia list")
	_, _ = fmt.Fprintln(c.out, "  pixelysia current")
	_, _ = fmt.Fprintln(c.out, "  pixelysia remove <theme>")
	_, _ = fmt.Fprintln(c.out, "  pixelysia doctor")
}

func (c *CLI) printInstallUsage() {
	_, _ = fmt.Fprintln(c.out, "Usage: pixelysia install [--split | --theme <name>]")
}

func requireNoMutuallyExclusive(split bool, theme string) error {
	if split && theme != "" {
		return errors.New("--split and --theme cannot be used together")
	}
	return nil
}
