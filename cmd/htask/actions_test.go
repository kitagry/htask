package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestActionsLint(t *testing.T) {
	dir := t.TempDir()
	clean := writeFile(t, dir, "clean.md", "- **🟠 1 [htask#1](https://github.com/kitagry/htask/pull/1) を merge してよい?**\n")
	dirty := writeFile(t, dir, "dirty.md", "- **🟠 1 PR #42 を merge してよい?**\n  - #7 と gokart#715\n")

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name:     "no violations",
			args:     []string{"actions", "lint", clean},
			wantCode: 0,
		},
		{
			name:     "one line per violation across files",
			args:     []string{"actions", "lint", clean, dirty},
			wantCode: 1,
			wantStdout: dirty + ":1: PR #42 is not a link\n" +
				dirty + ":2: #7 is not a link\n" +
				dirty + ":2: gokart#715 is not a link\n",
		},
		{
			name:     "json output",
			args:     []string{"actions", "lint", "--json", dirty},
			wantCode: 1,
			wantStdout: `[{"file":"` + dirty + `","line":1,"col":12,"ref":"PR #42"},` +
				`{"file":"` + dirty + `","line":2,"col":5,"ref":"#7"},` +
				`{"file":"` + dirty + `","line":2,"col":12,"ref":"gokart#715"}]` + "\n",
		},
		{
			name:       "json output without violations is an empty array",
			args:       []string{"actions", "lint", "--json", clean},
			wantCode:   0,
			wantStdout: "[]\n",
		},
		{
			name:       "no files is a usage error",
			args:       []string{"actions", "lint"},
			wantCode:   2,
			wantStderr: "Usage: htask actions lint",
		},
		{
			name:       "unknown flag is a usage error",
			args:       []string{"actions", "lint", "--nope", clean},
			wantCode:   2,
			wantStderr: "flag provided but not defined",
		},
		{
			name:       "unreadable file is a usage error",
			args:       []string{"actions", "lint", filepath.Join(dir, "missing.md")},
			wantCode:   2,
			wantStderr: "missing.md",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			got := run(tt.args, &stdout, &stderr)
			if got != tt.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %q)", got, tt.wantCode, stderr.String())
			}
			if diff := cmp.Diff(tt.wantStdout, stdout.String()); diff != "" {
				t.Errorf("stdout mismatch (-want +got):\n%s", diff)
			}
			// strings.Contains always holds for an empty want, so check emptiness explicitly.
			if tt.wantStderr == "" && stderr.Len() != 0 {
				t.Errorf("stderr = %q, want empty", stderr.String())
			} else if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}
