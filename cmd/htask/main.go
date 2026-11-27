// Command htask is the CLI behind the herdr-task skill.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

const (
	exitOK    = 0
	exitFail  = 1
	exitUsage = 2
)

type command struct {
	// name is "<noun> <verb>", or a single word for top-level commands.
	name    string
	summary string
	run     func(args []string, stdout, stderr io.Writer) int
}

var commands = []command{
	{name: "version", summary: "print the htask version", run: runVersion},
	{name: "actions lint", summary: "report PR / issue references in ACTIONS.md that are not links", run: runActionsLint},
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return exitUsage
	}
	switch args[0] {
	case "help", "-h", "-help", "--help":
		usage(stdout)
		return exitOK
	}
	if cmd, rest, ok := lookup(args); ok {
		return cmd.run(rest, stdout, stderr)
	}
	fmt.Fprintf(stderr, "htask: unknown command %q\n", strings.Join(args[:min(2, len(args))], " "))
	usage(stderr)
	return exitUsage
}

// lookup prefers the two-word "<noun> <verb>" form so that a noun can also
// exist as a top-level command later without shadowing its verbs.
func lookup(args []string) (command, []string, bool) {
	for n := min(2, len(args)); n >= 1; n-- {
		name := strings.Join(args[:n], " ")
		for _, c := range commands {
			if c.name == name {
				return c, args[n:], true
			}
		}
	}
	return command{}, nil, false
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "Usage: htask <command> [arguments]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	for _, c := range commands {
		fmt.Fprintf(w, "  %-16s %s\n", c.name, c.summary)
	}
}

// newFlagSet returns a FlagSet whose errors and usage go to stderr, so each
// command can return exitUsage itself instead of exiting the process.
func newFlagSet(name, args string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: htask %s %s\n", name, args)
		fs.PrintDefaults()
	}
	return fs
}

// parseFlags returns the exit code to use when the command must stop early.
func parseFlags(fs *flag.FlagSet, args []string) (int, bool) {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK, false
		}
		return exitUsage, false
	}
	return 0, true
}

func runVersion(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("version", "", stderr)
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return exitUsage
	}
	fmt.Fprintf(stdout, "htask %s\n", resolveVersion(version, debug.ReadBuildInfo))
	return exitOK
}

// resolveVersion falls back to the module version the go command records when
// -ldflags is not passed: the tag for `go install ...@vX.Y.Z`, or a VCS
// pseudo-version for a local `go build`. "(devel)" means no version was recorded.
func resolveVersion(v string, readBuildInfo func() (*debug.BuildInfo, bool)) string {
	if v != "dev" {
		return v
	}
	if info, ok := readBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return v
}
