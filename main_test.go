package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout runs fn while capturing everything written to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe writer: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read captured stdout: %v", err)
	}
	return string(out)
}

func TestRunSkillsCommandHelpFlag(t *testing.T) {
	// Isolate from the real home directory so an accidental execution
	// cannot touch the user's installed skills.
	t.Setenv("HOME", t.TempDir())

	subcommands := []string{"install", "list", "remove"}
	helpFlags := []string{"--help", "-h"}

	for _, sub := range subcommands {
		for _, flag := range helpFlags {
			t.Run(sub+" "+flag, func(t *testing.T) {
				out := captureStdout(t, func() {
					runSkillsCommand([]string{sub, flag})
				})

				if !strings.Contains(out, "Usage: logs-mcp-server skills") {
					t.Errorf("expected usage text, got:\n%s", out)
				}
				for _, executed := range []string{"Installed", "Removed", "skills available", "found to remove"} {
					if strings.Contains(out, executed) {
						t.Errorf("command executed instead of showing help (found %q):\n%s", executed, out)
					}
				}
			})
		}
	}
}
