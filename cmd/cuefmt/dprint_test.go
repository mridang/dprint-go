package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	cuefmt "cuelang.org/go/cue/format"
)

// TestDprint_Formats_Cue_File verifies end-to-end formatting using dprint
// and the TinyGo-built plugin.
//
// The test builds the WebAssembly plugin using the local Makefile target,
// writes a deliberately malformed CUE file in a temporary directory,
// invokes `dprint fmt` with the repo configuration, and asserts the
// resulting file bytes match the canonical output from cue/format. It
// then runs dprint a second time to assert idempotence.
//
// Preconditions:
//
//   - `make` is available in PATH.
//   - `tinygo` is available in PATH.
//   - `dprint` is available in PATH.
//   - `dprint.json` exists in the repository root and references the
//     plugin artifact at `build/cuefmt.wasm`.
//
// The test streams tool output on failures and uses timeouts to avoid
// hanging in CI. It fails fast on any unmet precondition.
//
//goland:noinspection DuplicatedCode,DuplicatedCode
func TestDprint_Formats_Cue_File(t *testing.T) {
	requireInPath(t, "make")
	requireInPath(t, "tinygo")
	requireInPath(t, "dprint")

	pkgDir := getwdOrFatal(t)
	requireFile(t, filepath.Join(pkgDir, "dprint.json"))

	buildPluginWasm(t)

	td := t.TempDir()
	srcPath := filepath.Join(td, "test.cue")

	bad := []byte(`package main
import "list"
foo:  {
	bar: "baz"
	num: 1
}`)

	if err := os.WriteFile(srcPath, bad, 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	want, err := cuefmt.Source(bad)
	if err != nil {
		t.Fatalf("cue/format failed on input: %v", err)
	}
	if bytes.Equal(bad, want) {
		t.Fatalf("test input not malformed; no change would be observed")
	}

	runDprintFmt(t, td, filepath.Join(pkgDir, "dprint.json"))

	got, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("read formatted file: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf(
			"file not formatted as expected\n--- got ---\n%s\n--- want ---\n%s",
			string(got),
			string(want),
		)
	}

	before := slices.Clone(got)
	runDprintFmt(t, td, filepath.Join(pkgDir, "dprint.json"))
	after, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("read file after second pass: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("dprint not idempotent on second pass")
	}
}

// buildPluginWasm compiles the plugin to build/cuefmt.wasm using the
// same flags as production. A timeout is applied to prevent hangs.
func buildPluginWasm(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("make"); err != nil {
		t.Fatalf("make not found in PATH: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "make", "build")
	runCmd(t, cmd, "make build")
}

// runDprintFmt executes `dprint fmt` inside workDir with explicit
// configuration and debug logging. A timeout is applied.
func runDprintFmt(t *testing.T, workDir, configPath string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"dprint", "fmt", "test.cue",
		"--log-level=debug",
		"--config="+configPath,
	)
	cmd.Dir = workDir
	runCmd(t, cmd, "dprint fmt")
}

// runCmd executes a command and fails the test with the captured output
// when the command exits non-zero or the context times out.
func runCmd(t *testing.T, cmd *exec.Cmd, label string) {
	t.Helper()

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		_, _ = os.Stderr.WriteString(buf.String())
		t.Fatalf("%s failed: %v", label, err)
	}
}

// requireInPath skips the test if the named binary is not found.
func requireInPath(t *testing.T, bin string) {
	t.Helper()
	if _, err := exec.LookPath(bin); err != nil {
		t.Skipf("%s not found in PATH: %v", bin, err)
	}
}

// requireFile fails if the path does not exist.
func requireFile(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("required file missing: %s: %v", path, err)
	}
}

// getwdOrFatal returns the current working directory or fails.
func getwdOrFatal(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return wd
}
