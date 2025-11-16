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
)

// TestDprint_Formats_Tf_File verifies end-to-end formatting using dprint
// and the TinyGo-built plugin.
//
// The test builds the WebAssembly plugin with TinyGo, injects a start
// section via the repository helper, writes a deliberately malformed Terraform
// file in a temporary directory, invokes `dprint fmt` with the repo
// configuration, and asserts the resulting file bytes match the
// canonical output from hclwrite. It then runs dprint a second time to
// assert idempotence.
//
// Preconditions:
//
//   - `tinygo` is available in PATH.
//   - `dprint` is available in PATH.
//   - `dprint.json` exists in the repository root and references the
//     plugin artifact at `build/tffmt.wasm`.
//   - The helper `cmd/addstart/main.go` exists to inject a start section
//     that calls `_initialize`.
//
// The test streams tool output on failures and uses timeouts to avoid
// hanging in CI. It fails fast on any unmet precondition.
func TestDprint_Formats_Tf_File(t *testing.T) {
	requireInPath(t, "tinygo")
	requireInPath(t, "dprint")

	pkgDir := getwdOrFatal(t)
	repoRoot := filepath.Join(pkgDir, "..", "..")
	requireFile(t, filepath.Join(pkgDir, "dprint.json"))
	requireFile(t, filepath.Join(repoRoot, "cmd", "addstart", "main.go"))

	buildPluginWasm(t)
	injectStartSection(t, repoRoot)

	td := t.TempDir()
	srcPath := filepath.Join(td, "main.tf")

	bad := []byte(`resource "aws_instance" "example" {
  ami = "${var.ami_id}"
  instance_type = "t2.micro"
  tags = {
    Name = "test"
  }
}
`)

	if err := os.WriteFile(srcPath, bad, 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	want, err := formatHCL(bad, defaultConfig())
	if err != nil {
		t.Fatalf("formatHCL failed on input: %v", err)
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

// buildPluginWasm compiles the plugin to build/tffmt.wasm using the
// same flags as production. A timeout is applied to prevent hangs.
func buildPluginWasm(t *testing.T) {
	t.Helper()

	if err := os.MkdirAll("build", 0o755); err != nil {
		t.Fatalf("mkdir build: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"tinygo", "build",
		"-o=build/tffmt.wasm",
		"-target=wasm-unknown",
		"-scheduler=none",
		"-no-debug",
		"-opt=2",
		"main.go",
	)
	runCmd(t, cmd, "tinygo build")
}

// injectStartSection runs the helper to inject a start section and
// replaces build/tffmt.wasm with the fixed output.
func injectStartSection(t *testing.T, repoRoot string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	addstartPath := filepath.Join(repoRoot, "cmd", "addstart", "main.go")
	cmd := exec.CommandContext(
		ctx,
		"go", "run", addstartPath,
		"build/tffmt.wasm",
		"build/dprint-fixed.wasm",
	)
	runCmd(t, cmd, "addstart")

	if err := os.Rename("build/dprint-fixed.wasm", "build/tffmt.wasm"); err != nil {
		t.Fatalf("rename fixed wasm: %v", err)
	}
}

// runDprintFmt executes `dprint fmt` inside workDir with explicit
// configuration and debug logging. A timeout is applied.
func runDprintFmt(t *testing.T, workDir, configPath string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"dprint", "fmt", "main.tf",
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
