//go:build !tinygo

package main

import (
	_ "embed"
	"encoding/json"
	gofmt "go/format"
	"strings"
	"unsafe"
)

//go:embed VERSION
var versionFile string

type PluginInfo struct {
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	ConfigKey       string   `json:"configKey"`
	FileExtensions  []string `json:"fileExtensions"`
	FileNames       []string `json:"fileNames"`
	HelpUrl         string   `json:"helpUrl"`
	ConfigSchemaUrl string   `json:"configSchemaUrl"`
}

//go:embed LICENSE
var licenseText string

const bufSize = 1 << 20

var shared [bufSize]byte
var activeSize uint32
var initialized bool
var fileContentSize uint32 // Track the file content size separately

// distinct globals to prevent ICF.
var _gA uint8
var _gB uint8
var _gC uint8
var _gD uint8
var _gE uint8
var _gF uint8

func ensureInit() {
	if !initialized {
		initialized = true
		// Initialize shared buffer
		_ = uintptr(unsafe.Pointer(&shared[0]))
	}
}

func putShared(b []byte) uint32 {
	ensureInit()
	if b == nil {
		return 0
	}
	if len(b) > len(shared) {
		b = b[:len(shared)]
	}
	n := copy(shared[:], b)
	return uint32(n)
}

//go:wasmexport get_shared_bytes_ptr
//go:noinline
//goland:noinspection ALL
func get_shared_bytes_ptr() uint32 {
	ensureInit()
	return uint32(uintptr(unsafe.Pointer(&shared[0])))
}

//go:wasmexport clear_shared_bytes
//go:noinline
//goland:noinspection ALL
func clear_shared_bytes(size uint32) uint32 {
	ensureInit()
	if size > bufSize {
		size = bufSize
	}
	// Store the size - this is the size of the file content that will be written
	activeSize = size
	fileContentSize = size // Remember this for format()
	// Don't actually clear - dprint will write the content
	return uint32(uintptr(unsafe.Pointer(&shared[0])))
}

//go:wasmexport dprint_plugin_version_4
//go:noinline
//goland:noinspection ALL
func dprint_plugin_version_4() uint32 {
	ensureInit()
	return 4
}

//go:wasmexport get_plugin_info
//go:noinline
//goland:noinspection ALL
func get_plugin_info() uint32 {
	ensureInit()

	version := strings.TrimSpace(versionFile)

	info := PluginInfo{
		Name:            "dprint-plugin-go-noop",
		Version:         version,
		ConfigKey:       "go-noop",
		FileExtensions:  []string{"go"},
		FileNames:       []string{},
		HelpUrl:         "",
		ConfigSchemaUrl: "",
	}

	jsonData, err := json.Marshal(info)
	if err != nil {
		// Fallback to empty JSON object if marshal fails
		return putShared([]byte("{}"))
	}

	return putShared(jsonData)
}

//go:wasmexport get_license_text
//go:noinline
func get_license_text() uint32 {
	ensureInit()
	return putShared([]byte(licenseText))
}

//go:wasmexport register_config
//go:noinline
//goland:noinspection ALL
func register_config(config_id uint32) {
	ensureInit()
	_gA = _gA ^ 1
}

//go:wasmexport release_config
//go:noinline
//goland:noinspection ALL
func release_config(config_id uint32) {
	ensureInit()
	_gB = _gB ^ 1
}

//go:wasmexport get_config_diagnostics
//go:noinline
//goland:noinspection ALL
func get_config_diagnostics(config_id uint32) uint32 {
	ensureInit()
	_gC = _gC ^ 1
	return putShared([]byte("[]"))
}

//go:wasmexport get_resolved_config
//go:noinline
//goland:noinspection ALL
func get_resolved_config(config_id uint32) uint32 {
	ensureInit()
	_gD = _gD ^ 1
	return putShared([]byte("{}"))
}

//go:wasmexport get_config_file_matching
//go:noinline
//goland:noinspection ALL
func get_config_file_matching(config_id uint32) uint32 {
	ensureInit()
	_gE = _gE ^ 1
	// Return the file matching info as JSON
	matching := []byte(`{"fileExtensions":["go"],"fileNames":[]}`)
	return putShared(matching)
}

//go:wasmexport set_file_path
//go:noinline
//goland:noinspection ALL
func set_file_path() {
	ensureInit()
	_gF = _gF ^ 1
	// dprint writes the file path to shared buffer
	// We could read it here if needed, but for now just note it was called
}

//go:wasmexport set_override_config
//go:noinline
//goland:noinspection ALL
func set_override_config() {
	ensureInit()
	// do nothing - valid no-op
}

//go:wasmexport format
//go:noinline
//goland:noinspection ALL
func format(config_id uint32) uint32 {
	ensureInit()

	// Get the content size
	contentSize := fileContentSize
	if activeSize > contentSize {
		contentSize = activeSize
	}

	if contentSize == 0 || contentSize > bufSize {
		return 0 // no content or too large
	}

	// Read the original content from shared buffer
	originalContent := make([]byte, contentSize)
	copy(originalContent, shared[:contentSize])

	// Format using go/format
	formatted, err := gofmt.Source(originalContent)
	if err != nil {
		// If there's a formatting error, return the error
		// Store error message in shared buffer for get_error_text
		errMsg := []byte(err.Error())
		if len(errMsg) > bufSize {
			errMsg = errMsg[:bufSize]
		}
		copy(shared[:], errMsg)
		activeSize = uint32(len(errMsg))
		return 2 // error
	}

	// Check if content changed
	if len(formatted) == len(originalContent) {
		same := true
		for i := range len(formatted) {
			if formatted[i] != originalContent[i] {
				same = false
				break
			}
		}
		if same {
			return 0 // no change
		}
	}

	// Store formatted content back in shared buffer
	if len(formatted) > bufSize {
		formatted = formatted[:bufSize]
	}

	activeSize = uint32(len(formatted))
	copy(shared[:], formatted)

	// Return 1 to indicate a change was made
	return 1
}

//go:wasmexport get_formatted_text
//go:noinline
//goland:noinspection ALL
func get_formatted_text() uint32 {
	ensureInit()
	// Return the size of the formatted text in the shared buffer
	return activeSize
}

//go:wasmexport get_error_text
//go:noinline
//goland:noinspection ALL
func get_error_text() uint32 {
	ensureInit()
	// Return the size of the error text in the shared buffer
	// (it was already written there by format() when it returned 2)
	return activeSize
}

func main() {
	ensureInit()
}
