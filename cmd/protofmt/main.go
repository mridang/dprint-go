package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"unsafe"

	// Importing Buf's private packages directly as requested.
	// NOTE: This assumes that the Go compiler allows importing 'private' folders
	// and that TinyGo can handle the massive dependency tree pulled in by these packages.
	"github.com/bufbuild/buf/private/buf/bufformat"
	"github.com/mridang/dprint-plugin-go/internal/dprint"
)

var currentConfig = defaultConfig() //nolint:unused, gochecknoglobals

//go:embed VERSION
var versionFile string

//go:embed LICENSE
var licenseText string

// Global state variables
var (
	shared          [dprint.SharedBufferSize]byte
	activeSize      uint32
	initialized     bool
	fileContentSize uint32
	filePath        string
)

func ensureInit() {
	if !initialized {
		initialized = true
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
	activeSize = toUint32(n)
	return toUint32(n)
}

//go:wasmexport get_shared_bytes_ptr
func get_shared_bytes_ptr() uint32 {
	ensureInit()
	return uint32(uintptr(unsafe.Pointer(&shared[0])))
}

//go:wasmexport clear_shared_bytes
func clear_shared_bytes(size uint32) uint32 {
	ensureInit()
	if size > dprint.SharedBufferSize {
		size = dprint.SharedBufferSize
	}
	activeSize = size
	fileContentSize = size
	return uint32(uintptr(unsafe.Pointer(&shared[0])))
}

//go:wasmexport dprint_plugin_version_4
func dprint_plugin_version_4() uint32 {
	ensureInit()
	return dprint.PluginSchemaVersion
}

//go:wasmexport get_plugin_info
func get_plugin_info() uint32 {
	ensureInit()
	version := strings.TrimSpace(versionFile)
	info := dprint.PluginInfo{
		Name:            "dprint-plugin-proto",
		Version:         version,
		ConfigKey:       "proto",
		FileExtensions:  []string{"proto"},
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
func get_license_text() uint32 {
	ensureInit()
	return putShared([]byte(licenseText))
}

//go:wasmexport get_config_file_matching
func get_config_file_matching(_ uint32) uint32 {
	ensureInit()
	_gE ^= 1
	matching := dprint.FileMatchingInfo{
		FileExtensions: []string{"proto"},
		FileNames:      []string{},
	}
	data, err := json.Marshal(matching)
	if err != nil {
		return putShared([]byte(dprint.SupportedFiles))
	}
	return putShared(data)
}

type Config struct {
	// Buf generally doesn't expose config options for formatting via API struct
}

func defaultConfig() Config {
	return Config{}
}

//go:wasmexport register_config
func register_config(_ uint32) {
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
func get_resolved_config(_ uint32) uint32 {
	ensureInit()
	_gD ^= 1
	data, err := json.Marshal(currentConfig)
	if err != nil {
		return putShared([]byte("{}"))
	}
	return putShared(data)
}

//go:wasmexport set_file_path
func set_file_path() {
	ensureInit()
	_gF ^= 1
	// dprint writes the file path into the shared buffer before calling this
	pathBytes := make([]byte, activeSize)
	copy(pathBytes, shared[:activeSize])
	filePath = string(pathBytes)
	if filePath == "" {
		filePath = "file.proto"
	}
}

//go:wasmexport format
func format(_ uint32) uint32 {
	ensureInit()

	contentSize := max(activeSize, fileContentSize)
	if contentSize == 0 || contentSize > dprint.SharedBufferSize {
		return dprint.FormatResultNoChange
	}

	input := slices.Clone(shared[:contentSize])

	// --- CHANGED SECTION START ---
	// No more buckets. No more context. No more OS calls.

	// We pass the filename (filePath) and the input bytes directly.
	formattedBytes, err := bufformat.Format(filePath, input)
	if err != nil {
		return handleError(err)
	}
	// --- CHANGED SECTION END ---

	// Check if content changed
	if len(formattedBytes) == len(input) && string(formattedBytes) == string(input) {
		return dprint.FormatResultNoChange
	}

	if len(formattedBytes) > dprint.SharedBufferSize {
		return handleError(errors.New("formatted content exceeds buffer size"))
	}

	activeSize = toUint32(len(formattedBytes))
	copy(shared[:], formattedBytes)
	return dprint.FormatResultChanged
}

func handleError(err error) uint32 {
	errMsg := []byte(err.Error())
	if len(errMsg) > dprint.SharedBufferSize {
		errMsg = errMsg[:dprint.SharedBufferSize]
	}
	copy(shared[:], errMsg)
	activeSize = toUint32(len(errMsg))
	return dprint.FormatResultError
}

//go:wasmexport get_formatted_text
func get_formatted_text() uint32 {
	ensureInit()
	return activeSize
}

//go:wasmexport get_error_text
func get_error_text() uint32 {
	ensureInit()
	return activeSize
}

func main() {
	ensureInit()
}

// Dummy globals to prevent optimization
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
func release_config(_ uint32) {
	ensureInit()
	_gB ^= 1
}

//go:wasmexport get_config_diagnostics
func get_config_diagnostics(_ uint32) uint32 {
	ensureInit()
	_gC ^= 1
	return putShared([]byte("[]"))
}

//go:wasmexport set_override_config
func set_override_config() {
	ensureInit()
	_gG ^= 1
}

func toUint32(val int) uint32 {
	return uint32(val) //nolint:gosec
}
