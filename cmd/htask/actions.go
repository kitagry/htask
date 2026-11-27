package main

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"os"

	"github.com/kitagry/htask/ghref"
)

type lintViolation struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Col  int    `json:"col"`
	Ref  string `json:"ref"`
}

func runActionsLint(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("actions lint", "[--json] <ACTIONS.md>...", stderr)
	asJSON := fs.Bool("json", false, "print violations as a JSON array")
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return exitUsage
	}

	// Read every file before printing anything, so a typo in the last path
	// does not leave a partial report behind.
	srcs := make([]string, fs.NArg())
	for i, path := range fs.Args() {
		b, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "htask actions lint: %v\n", err)
			return exitUsage
		}
		srcs[i] = string(b)
	}

	violations := []lintViolation{}
	for i, path := range fs.Args() {
		for _, r := range ghref.Find(srcs[i]) {
			violations = append(violations, lintViolation{File: path, Line: r.Line, Col: r.Col, Ref: r.Text})
		}
	}

	if *asJSON {
		if err := json.MarshalWrite(stdout, violations); err != nil {
			fmt.Fprintf(stderr, "htask actions lint: %v\n", err)
			return exitFail
		}
		fmt.Fprintln(stdout)
	} else {
		for _, v := range violations {
			fmt.Fprintf(stdout, "%s:%d: %s is not a link\n", v.File, v.Line, v.Ref)
		}
	}
	if len(violations) > 0 {
		return exitFail
	}
	return exitOK
}
