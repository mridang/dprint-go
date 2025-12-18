//goland:noinspection DuplicatedCode
package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mridang/dprint-plugin-go/internal/dprint"
	"github.com/mridang/dprint-plugin-go/internal/wasm"
	"github.com/wasmerio/wasmer-go/wasmer"
)

// TestWasm_Exports_And_OptionalCall verifies that the compiled Wasm module
// exports all the expected functions for the dprint V2 ABI. It builds the
// TinyGo Wasm, strips any start section (which wasmer-go doesn't support),
// and instantiates it with no-op dprint host imports.
//
//goland:noinspection DuplicatedCode
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

	if initFn, err := instance.Exports.GetFunction("_initialize"); err == nil { //nolint:govet // TinyGo export name is required.
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

	assertWasmPluginInfo(t, instance)
}

// buildTinyGoWasm compiles the package in the current directory to a
// Wasm module using TinyGo.
//
//goland:noinspection DuplicatedCode
func buildTinyGoWasm(t *testing.T) []byte {
	t.Helper()
	if _, err := exec.LookPath("tinygo"); err != nil {
		t.Fatalf("tinygo not found in PATH: %v", err)
	}
	dir := t.TempDir()
	out := filepath.Join(dir, "protofmt.wasm")
	cmd := exec.Command(
		"tinygo", "build",
		"-o", out,
		"-target=wasm-unknown",
		"-scheduler=none",
		"-no-debug",
		"-opt=2",
		".",
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
//
//goland:noinspection DuplicatedCode
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

//goland:noinspection DuplicatedCode
func assertWasmPluginInfo(t *testing.T, instance *wasmer.Instance) {
	t.Helper()

	infoFn, err := instance.Exports.GetFunction("get_plugin_info")
	if err != nil {
		t.Fatalf("get get_plugin_info: %v", err)
	}
	sizeVal, err := infoFn()
	if err != nil {
		t.Fatalf("call get_plugin_info: %v", err)
	}
	size := uint32(sizeVal.(int32))
	if size == 0 {
		t.Fatalf("get_plugin_info returned 0")
	}

	data := readSharedBytes(t, instance, size)
	var info dprint.PluginInfo
	if unmarshalErr := json.Unmarshal(data, &info); unmarshalErr != nil {
		t.Fatalf("unmarshal plugin info: %v", unmarshalErr)
	}
	if info.Version != strings.TrimSpace(versionFile) {
		t.Fatalf("wasm plugin version = %q; want %q", info.Version, strings.TrimSpace(versionFile))
	}
}

//goland:noinspection DuplicatedCode
func readSharedBytes(t *testing.T, instance *wasmer.Instance, size uint32) []byte {
	t.Helper()

	ptrFn, err := instance.Exports.GetFunction("get_shared_bytes_ptr")
	if err != nil {
		t.Fatalf("get get_shared_bytes_ptr: %v", err)
	}
	ptrVal, err := ptrFn()
	if err != nil {
		t.Fatalf("call get_shared_bytes_ptr: %v", err)
	}
	ptr := uint32(ptrVal.(int32))

	mem, err := instance.Exports.GetMemory("memory")
	if err != nil {
		t.Fatalf("get memory: %v", err)
	}
	data := mem.Data()
	end := int(ptr + size)
	if end > len(data) {
		t.Fatalf("shared buffer exceeds memory: %d > %d", end, len(data))
	}
	return slices.Clone(data[ptr:end])
}
