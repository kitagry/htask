package main

import (
	"bytes"
	"runtime/debug"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "version prints the embedded version",
			args:       []string{"version"},
			wantCode:   0,
			wantStdout: "htask dev\n",
		},
		{
			name:       "no arguments is a usage error",
			args:       nil,
			wantCode:   2,
			wantStderr: "Usage: htask",
		},
		{
			name:       "unknown command is a usage error",
			args:       []string{"nope"},
			wantCode:   2,
			wantStderr: `unknown command "nope"`,
		},
		{
			name:       "help prints usage to stdout",
			args:       []string{"help"},
			wantCode:   0,
			wantStdout: "Usage: htask",
		},
		{
			name:       "-h prints usage to stdout",
			args:       []string{"-h"},
			wantCode:   0,
			wantStdout: "Usage: htask",
		},
		{
			name:       "version rejects extra arguments",
			args:       []string{"version", "extra"},
			wantCode:   2,
			wantStderr: "Usage: htask version",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			got := run(tt.args, &stdout, &stderr)
			if got != tt.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %q)", got, tt.wantCode, stderr.String())
			}
			if !strings.Contains(stdout.String(), tt.wantStdout) {
				t.Errorf("stdout = %q, want it to contain %q", stdout.String(), tt.wantStdout)
			}
			if tt.wantStdout == "" && stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty", stdout.String())
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}

func TestResolveVersion(t *testing.T) {
	info := func(v string) func() (*debug.BuildInfo, bool) {
		return func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Version: v}}, true
		}
	}
	noInfo := func() (*debug.BuildInfo, bool) { return nil, false }
	tests := []struct {
		name string
		v    string
		read func() (*debug.BuildInfo, bool)
		want string
	}{
		{name: "ldflags version wins", v: "0.1.0", read: info("v9.9.9"), want: "0.1.0"},
		{name: "go install version", v: "dev", read: info("v0.1.0"), want: "0.1.0"},
		{name: "local build", v: "dev", read: info("(devel)"), want: "dev"},
		{name: "no build info", v: "dev", read: noInfo, want: "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.v, tt.read); got != tt.want {
				t.Errorf("resolveVersion(%q) = %q, want %q", tt.v, got, tt.want)
			}
		})
	}
}
