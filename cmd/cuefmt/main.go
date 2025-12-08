package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"unsafe"

	cueformat "cuelang.org/go/cue/format"
	"github.com/mridang/dprint-plugin-go/internal/dprint"
)

var currentConfig = defaultConfig() //nolint:unused, gochecknoglobals // CGO global variable

//go:embed VERSION
var versionFile string //nolint:unused // it is actually used

//go:embed LICENSE
var licenseText string //nolint:unused // it is actually used

// Global state variables.
var (
	shared          [dprint.SharedBufferSize]byte //nolint: gochecknoglobals // CGO global variable
	activeSize      uint32                        //nolint:unused, gochecknoglobals // CGO global variable
	initialized     bool                          //nolint:unused, gochecknoglobals // CGO global variable
	fileContentSize uint32                        //nolint:unused, gochecknoglobals // CGO global variable
)

// ensureInit initializes the plugin if not already initialized.
func ensureInit() {
	if !initialized {
		if devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0); err == nil {
			os.Stdout = devNull
			os.Stderr = devNull
		}
		initialized = true
		_ = uintptr(unsafe.Pointer(&shared[0]))
	}
}

// putShared copies bytes into the shared buffer and returns their length.
func putShared(b []byte) uint32 { ///nolint:revive,unused,staticcheck // because it is exported
	ensureInit()
	if b == nil {
		return 0
	}
	if len(b) > len(shared) {
		b = b[:len(shared)]
	}
	n := copy(shared[:], b)
	activeSize = toUint32(n)
	return toUint32(n)
}

//go:wasmexport get_shared_bytes_ptr
//go:noinline
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction
func get_shared_bytes_ptr() uint32 { //nolint:revive,unused,staticcheck // because it is exported
	ensureInit()
	return uint32(uintptr(unsafe.Pointer(&shared[0])))
}

//go:wasmexport clear_shared_bytes
//go:noinline
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction
func clear_shared_bytes(size uint32) uint32 { //nolint:revive,unused,staticcheck // because it is exported
	ensureInit()
	if size > dprint.SharedBufferSize {
		size = dprint.SharedBufferSize
	}
	activeSize = size
	fileContentSize = size
	return uint32(uintptr(unsafe.Pointer(&shared[0])))
}

//go:wasmexport dprint_plugin_version_4
//go:noinline
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction
func dprint_plugin_version_4() uint32 { //nolint:revive,unused,staticcheck // because it is exported
	ensureInit()
	return dprint.PluginSchemaVersion
}

//go:wasmexport get_plugin_info
//go:noinline
//goland:noinspection GoUnusedFunction,GoSnakeCaseUsage
func get_plugin_info() uint32 { //nolint:revive,unused,staticcheck // because it is exported
	ensureInit()

	version := strings.TrimSpace(versionFile)
	info := dprint.PluginInfo{
		Name:            "dprint-plugin-cue",
		Version:         version,
		ConfigKey:       "cue",
		FileExtensions:  []string{"cue"},
		FileNames:       []string{},
		HelpURL:         "",
		ConfigSchemaURL: "",
	}

	jsonData, err := json.Marshal(info)
	if err != nil {
		return putShared([]byte("{}"))
	}

	return putShared(jsonData)
}

//go:wasmexport get_license_text
//go:noinline
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction
func get_license_text() uint32 { //nolint:revive,unused,staticcheck // because it is exported
	ensureInit()
	return putShared([]byte(licenseText))
}

//go:wasmexport get_config_file_matching
//go:noinline
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction,GoUnusedParameter
func get_config_file_matching(_ uint32) uint32 { //nolint:revive,unused,staticcheck // because it is exported
	ensureInit()
	_gE ^= 1
	matching := dprint.FileMatchingInfo{
		FileExtensions: []string{"cue"},
		FileNames:      []string{},
	}
	data, err := json.Marshal(matching)
	if err != nil {
		return putShared([]byte(dprint.SupportedFiles))
	}
	return putShared(data)
}

// Config mirrors dprint's global config options that CUE can use.
type Config struct {
	UseTabs     *bool `json:"useTabs"`
	IndentWidth *int  `json:"indentWidth"`
	Simplify    *bool `json:"simplify"` // CUE specific option
}

func defaultConfig() Config {
	return Config{}
}

//go:wasmexport register_config
//go:noinline
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction,GoUnusedParameter
func register_config(_ uint32) { //nolint:revive,unused,staticcheck // because it is exported
	ensureInit()
	_gA ^= 1
	buf := make([]byte, activeSize)
	copy(buf, shared[:activeSize])
	cfg := defaultConfig()
	if len(buf) != 0 {
		_ = json.Unmarshal(buf, &cfg)
	}
	currentConfig = cfg
}

//go:wasmexport get_resolved_config
//go:noinline
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction,GoUnusedParameter
func get_resolved_config(_ uint32) uint32 { //nolint:revive,unused,staticcheck // because it is exported
	ensureInit()
	_gD ^= 1
	data, err := json.Marshal(currentConfig)
	if err != nil {
		return putShared([]byte("{}"))
	}
	return putShared(data)
}

//go:wasmexport format
//go:noinline
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction,GoUnusedParameter
func format(_ uint32) uint32 { //nolint:unused // because it is exported
	ensureInit()

	contentSize := max(activeSize, fileContentSize)
	if contentSize == 0 || contentSize > dprint.SharedBufferSize {
		return dprint.FormatResultNoChange
	}

	input := slices.Clone(shared[:contentSize])

	formatted, err := cueformat.Source(input)
	if err != nil {
		errMsg := []byte(err.Error())
		if len(errMsg) > dprint.SharedBufferSize {
			errMsg = errMsg[:dprint.SharedBufferSize]
		}
		copy(shared[:], errMsg)
		activeSize = toUint32(len(errMsg))
		return dprint.FormatResultError
	}

	if len(formatted) == len(input) && bytes.Equal(formatted, input) {
		return dprint.FormatResultNoChange
	}

	if len(formatted) > dprint.SharedBufferSize {
		errMsg := []byte("file too large for formatting")
		copy(shared[:], errMsg)
		activeSize = toUint32(len(errMsg))
		return dprint.FormatResultError
	}

	activeSize = toUint32(len(formatted))
	copy(shared[:], formatted)
	return dprint.FormatResultChanged
}

//go:wasmexport get_formatted_text
//go:noinline
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction
func get_formatted_text() uint32 { //nolint:revive,unused,staticcheck // because it is exported
	ensureInit()
	return activeSize
}

//go:wasmexport get_error_text
//go:noinline
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction
func get_error_text() uint32 { //nolint:revive,unused,staticcheck // because it is exported
	ensureInit()
	return activeSize
}

// The main is the entry point for the WASM module.
func main() {
	ensureInit()
}

// Dummy globals to prevent Identical Code Folding
var (
	_gA uint8
	_gB uint8
	_gC uint8
	_gD uint8
	_gE uint8
	_gF uint8
	_gG uint8
)

//go:wasmexport release_config
//go:noinline
//goland:noinspection GoUnusedFunction,GoUnusedParameter,GoSnakeCaseUsage
func release_config(_ uint32) {
	ensureInit()
	_gB ^= 1
}

//go:wasmexport get_config_diagnostics
//go:noinline
//goland:noinspection GoUnusedFunction,GoUnusedParameter,GoSnakeCaseUsage
func get_config_diagnostics(_ uint32) uint32 {
	ensureInit()
	_gC ^= 1
	return putShared([]byte("[]"))
}

//go:wasmexport set_file_path
//go:noinline
//goland:noinspection GoUnusedFunction, GoSnakeCaseUsage
func set_file_path() {
	ensureInit()
	_gF ^= 1
}

//go:wasmexport set_override_config
//go:noinline
//goland:noinspection GoUnusedFunction,GoSnakeCaseUsage
func set_override_config() {
	ensureInit()
	_gG ^= 1
}

func toUint32(val int) uint32 {
	return uint32(val) //nolint:gosec
}
