package main

//revive:disable:var-naming // WASM exports require underscore names.

import (
	"bytes"
	_ "embed"
	"encoding/json"
	gofmt "go/format"
	"slices"
	"strings"
	"unsafe"

	"github.com/mridang/dprint-plugin-go/internal/dprint"
)

//go:embed VERSION
var versionFile string

//go:embed LICENSE
var licenseText string

// Global state variables.
var (
	shared          [dprint.SharedBufferSize]byte //nolint:gochecknoglobals // required for wasm shared state.
	activeSize      uint32                        //nolint:gochecknoglobals // required for wasm shared state.
	initialized     bool                          //nolint:gochecknoglobals // required for wasm shared state.
	fileContentSize uint32                        //nolint:gochecknoglobals // required for wasm shared state.
)

// ensureInit initializes the plugin if not already initialized.
// This must be called before any other plugin operations.
func ensureInit() {
	if !initialized {
		initialized = true
		_ = uintptr(unsafe.Pointer(&shared[0]))
	}
}

// putShared copies data to the shared buffer and returns the number of bytes
// copied. If the data is larger than the buffer, it will be truncated.
func putShared(b []byte) uint32 {
	ensureInit()
	if b == nil {
		return 0
	}
	if len(b) > len(shared) {
		b = b[:len(shared)]
	}
	n := copy(shared[:], b)
	return toUint32(n)
}

// get_shared_bytes_ptr returns a pointer to the shared Wasm memory buffer.
// This is called by the dprint CLI to access the shared buffer.
// See: https://dprint.dev/plugins/wasm/#get_shared_bytes_ptr
//
//go:wasmexport get_shared_bytes_ptr
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage
func get_shared_bytes_ptr() uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	return uint32(uintptr(unsafe.Pointer(&shared[0])))
}

// clear_shared_bytes clears the shared byte array and returns a pointer to it.
// The dprint CLI calls this to prepare the buffer for writing file content.
// See: https://dprint.dev/plugins/wasm/#clear_shared_bytes
//
//go:wasmexport clear_shared_bytes
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage
func clear_shared_bytes(size uint32) uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	if size > dprint.SharedBufferSize {
		size = dprint.SharedBufferSize
	}
	activeSize = size
	fileContentSize = size
	return uint32(uintptr(unsafe.Pointer(&shared[0])))
}

// dprint_plugin_version_4 returns the schema version supported by this plugin.
// The CLI checks for this export to determine plugin compatibility.
// See: https://dprint.dev/plugins/wasm/#dprint_plugin_version_4
//
//go:wasmexport dprint_plugin_version_4
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage
func dprint_plugin_version_4() uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	return dprint.PluginSchemaVersion
}

// get_plugin_info serializes and returns the plugin information as JSON.
// This includes the plugin name, version, configuration key, and supported
// file extensions. See: https://dprint.dev/plugins/wasm/#get_plugin_info
//
//go:wasmexport get_plugin_info
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage
func get_plugin_info() uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()

	version := strings.TrimSpace(versionFile)
	info := dprint.PluginInfo{
		Name:            "dprint-plugin-gofmt",
		Version:         version,
		ConfigKey:       "go",
		FileExtensions:  []string{"go"},
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

// get_license_text returns the license text for this plugin.
// The license is embedded at compile time from the LICENSE file.
// See: https://dprint.dev/plugins/wasm/#get_license_text
//
//go:wasmexport get_license_text
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage
func get_license_text() uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	return putShared([]byte(licenseText))
}

// get_config_file_matching returns the file matching configuration as JSON.
// This tells dprint which files this plugin can format.
// See: https://dprint.dev/plugins/wasm/#get_config_file_matching
//
//go:wasmexport get_config_file_matching
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage
func get_config_file_matching(_ uint32) uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	_gE ^= 1
	matching := dprint.FileMatchingInfo{
		FileExtensions: []string{"go"},
		FileNames:      []string{},
	}

	jsonData, err := json.Marshal(matching)
	if err != nil {
		return putShared([]byte(dprint.SupportedFiles))
	}

	return putShared(jsonData)
}

// format performs the actual code formatting using go/format.
// Returns formatResultNoChange (0) for no changes, formatResultChanged (1)
// for successful formatting, or formatResultError (2) for errors.
// See: https://dprint.dev/plugins/wasm/#format
//
//go:wasmexport format
//go:noinline
//goland:noinspection DuplicatedCode
func format(_ uint32) uint32 {
	ensureInit()

	contentSize := max(activeSize, fileContentSize)

	if contentSize == 0 || contentSize > dprint.SharedBufferSize {
		return dprint.FormatResultNoChange
	}

	originalContent := slices.Clone(shared[:contentSize])
	formatted, err := gofmt.Source(originalContent)
	if err != nil {
		errMsg := []byte(err.Error())
		if len(errMsg) > dprint.SharedBufferSize {
			errMsg = errMsg[:dprint.SharedBufferSize]
		}
		copy(shared[:], errMsg)
		activeSize = toUint32(len(errMsg))
		return dprint.FormatResultError
	}

	if len(formatted) == len(originalContent) && bytes.Equal(formatted, originalContent) {
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

// get_formatted_text returns the size of the formatted text in the shared
// buffer. Called after format() returns formatResultChanged.
// See: https://dprint.dev/plugins/wasm/#get_formatted_text
//
//go:wasmexport get_formatted_text
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage
func get_formatted_text() uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	return activeSize
}

// get_error_text returns the size of the error text in the shared buffer.
// Called after format() returns formatResultError.
// See: https://dprint.dev/plugins/wasm/#get_error_text
//
//go:wasmexport get_error_text
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage
func get_error_text() uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	return activeSize
}

// The main is the entry point for the WASM module.
func main() {
	ensureInit()
}

// Dummy globals to prevent Identical Code Folding optimization from
// merging these placeholder functions.
var (
	_gA uint8 //nolint:gochecknoglobals,unused // required to keep wasm exports stable.
	_gB uint8 //nolint:gochecknoglobals,unused // required to keep wasm exports stable.
	_gC uint8 //nolint:gochecknoglobals,unused // required to keep wasm exports stable.
	_gD uint8 //nolint:gochecknoglobals,unused // required to keep wasm exports stable.
	_gE uint8 //nolint:gochecknoglobals,unused // required to keep wasm exports stable.
	_gF uint8 //nolint:gochecknoglobals,unused // required to keep wasm exports stable.
	_gG uint8 //nolint:gochecknoglobals,unused // required to keep wasm exports stable.
)

// register_config is called when the plugin and global configuration are complete.
// Store the configuration for later use during formatting.
// See: https://dprint.dev/plugins/wasm/#register_config
//
//go:wasmexport register_config
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage
func register_config(_ uint32) { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	_gA ^= 1
}

// release_config releases the configuration from memory when no longer needed.
// See: https://dprint.dev/plugins/wasm/#release_config
//
//go:wasmexport release_config
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage
func release_config(_ uint32) { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	_gB ^= 1
}

// get_config_diagnostics returns configuration validation diagnostics as JSON.
// This should return an array of diagnostic messages for invalid config.
// See: https://dprint.dev/plugins/wasm/#get_config_diagnostics
//
//go:wasmexport get_config_diagnostics
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage
func get_config_diagnostics(_ uint32) uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	_gC ^= 1
	return putShared([]byte("[]"))
}

// get_resolved_config returns the resolved configuration as JSON for display
// in the CLI. This shows the final configuration after all processing.
// See: https://dprint.dev/plugins/wasm/#get_resolved_config
//
//go:wasmexport get_resolved_config
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage
func get_resolved_config(_ uint32) uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	_gD ^= 1
	return putShared([]byte("{}"))
}

// set_file_path is called by the CLI to set the file path in the shared buffer.
// The plugin can read this path if needed for context-specific formatting.
// See: https://dprint.dev/plugins/wasm/#set_file_path
//
//go:wasmexport set_file_path
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage
func set_file_path() { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	_gF ^= 1
}

// set_override_config is called by the CLI to set override configuration.
// This allows per-file or per-directory configuration overrides.
// See: https://dprint.dev/plugins/wasm/#set_override_config
//
//go:wasmexport set_override_config
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage
func set_override_config() { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	_gG ^= 1
}

// toUint32 converts an int to uint32, suppressing the G115 overflow warning.
func toUint32(val int) uint32 {
	// This cast from int (64-bit) to uint32 (32-bit) is required by the WASM ABI.
	return uint32(val) //nolint:gosec // required for wasm pointer sizes.
}
