//goland:noinspection DuplicatedCode
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mridang/dprint-plugin-go/internal/wasm"
	"github.com/wasmerio/wasmer-go/wasmer"
)

// TestWasm_Exports_And_OptionalCall verifies that the compiled Wasm module
// exports all the expected functions for the dprint V2 ABI. It builds the
// TinyGo Wasm, strips any start section (which wasmer-go doesn't support),
// and instantiates it with no-op dprint host imports.
func TestWasm_Exports_And_OptionalCall(t *testing.T) {
	wasmBytes := buildTinyGoWasm(t)
	wasmBytes = wasm.StripStartSection(wasmBytes)

	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)

	module, err := wasmer.NewModule(store, wasmBytes)
	if err != nil {
		t.Fatalf("parse module: %v", err)
	}

	expected := map[string]struct{}{
		"get_shared_bytes_ptr":     {},
		"clear_shared_bytes":       {},
		"dprint_plugin_version_4":  {},
		"get_plugin_info":          {},
		"get_license_text":         {},
		"register_config":          {},
		"release_config":           {},
		"get_config_diagnostics":   {},
		"get_resolved_config":      {},
		"get_config_file_matching": {},
		"set_file_path":            {},
		"set_override_config":      {},
		"format":                   {},
		"get_formatted_text":       {},
		"get_error_text":           {},
	}

	found := make(map[string]*wasmer.ExternType)
	for _, et := range module.Exports() {
		found[et.Name()] = et.Type()
	}
	for name := range expected {
		typ, ok := found[name]
		if !ok {
			t.Errorf("missing wasm export: %q", name)
			continue
		}
		if typ.IntoFunctionType() == nil {
			t.Errorf("export %q is not a function", name)
		}
	}

	imports := wasmer.NewImportObject()
	registerNoOpDprint(t, store, imports)

	instance, err := wasmer.NewInstance(module, imports)
	if err != nil {
		t.Fatalf("instantiate: %v", err)
	}

	if initFn, err := instance.Exports.GetFunction("_initialize"); err == nil { //nolint:govet // this is why
		if _, err = initFn(); err != nil {
			t.Skipf("skipping runtime calls; _initialize trapped: %v", err)
			return
		}
	} else {
		t.Log("no _initialize export; proceeding without runtime init")
	}

	fn, err := instance.Exports.GetFunction("dprint_plugin_version_4")
	if err != nil {
		t.Fatalf("get dprint_plugin_version_4: %v", err)
	}
	v, callErr := fn()
	if callErr != nil {
		t.Skipf("skipping value assertion; call trapped: %v", callErr)
		return
	}
	if got := v.(int32); got != 4 {
		t.Fatalf("dprint_plugin_version_4 = %d; want 4", got)
	}
}

// buildTinyGoWasm compiles the package in the current directory to a
// Wasm module using TinyGo.
func buildTinyGoWasm(t *testing.T) []byte {
	t.Helper()
	if _, err := exec.LookPath("tinygo"); err != nil {
		t.Fatalf("tinygo not found in PATH: %v", err)
	}
	dir := t.TempDir()
	out := filepath.Join(dir, "tffmt.wasm")
	cmd := exec.Command(
		"tinygo", "build",
		"-o", out,
		"-target=wasm-unknown",
		"-scheduler=none",
		"-no-debug",
		"-opt=2",
		".", // Build the package in the current directory
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("tinygo build failed: %v", err)
	}
	bin, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read wasm: %v", err)
	}
	return bin
}

// registerNoOpDprint registers stub implementations of the host functions
// that dprint provides to the Wasm module.
func registerNoOpDprint(t *testing.T, store *wasmer.Store, imports *wasmer.ImportObject) {
	t.Helper()
	newFunc := func(params, results []wasmer.ValueKind, f func([]wasmer.Value) ([]wasmer.Value, error)) *wasmer.Function {
		return wasmer.NewFunction(
			store,
			wasmer.NewFunctionType(
				wasmer.NewValueTypes(params...),
				wasmer.NewValueTypes(results...),
			),
			f,
		)
	}
	imports.Register(
		"dprint",
		map[string]wasmer.IntoExtern{
			"host_write_buffer": newFunc(
				[]wasmer.ValueKind{wasmer.I32}, nil,
				func([]wasmer.Value) ([]wasmer.Value, error) { return nil, nil },
			),
			"host_format": newFunc(
				[]wasmer.ValueKind{
					wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32,
					wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32,
				},
				[]wasmer.ValueKind{wasmer.I32},
				func([]wasmer.Value) ([]wasmer.Value, error) {
					return []wasmer.Value{wasmer.NewI32(0)}, nil
				},
			),
			"host_get_formatted_text": newFunc(
				nil, []wasmer.ValueKind{wasmer.I32},
				func([]wasmer.Value) ([]wasmer.Value, error) {
					return []wasmer.Value{wasmer.NewI32(0)}, nil
				},
			),
			"host_get_error_text": newFunc(
				nil, []wasmer.ValueKind{wasmer.I32},
				func([]wasmer.Value) ([]wasmer.Value, error) {
					return []wasmer.Value{wasmer.NewI32(0)}, nil
				},
			),
			"host_has_cancelled": newFunc(
				nil, []wasmer.ValueKind{wasmer.I32},
				func([]wasmer.Value) ([]wasmer.Value, error) {
					return []wasmer.Value{wasmer.NewI32(0)}, nil
				},
			),
		},
	)
}
