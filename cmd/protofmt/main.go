package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"slices"
	"strings"
	"unsafe"

	"github.com/emicklei/proto"
	"github.com/emicklei/proto-contrib/pkg/protofmt"
	"github.com/mridang/dprint-plugin-go/internal/dprint"
)

//go:embed VERSION
var versionFile string

//go:embed LICENSE
var licenseText string

// Global state variables.
var (
	shared          [dprint.SharedBufferSize]byte
	activeSize      uint32
	initialized     bool
	fileContentSize uint32
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

//go:wasmexport get_shared_bytes_ptr
//go:noinline
func get_shared_bytes_ptr() uint32 {
	ensureInit()
	return uint32(uintptr(unsafe.Pointer(&shared[0])))
}

//go:wasmexport clear_shared_bytes
//go:noinline
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
//go:noinline
func dprint_plugin_version_4() uint32 {
	ensureInit()
	return dprint.PluginSchemaVersion
}

//go:wasmexport get_plugin_info
//go:noinline
func get_plugin_info() uint32 {
	ensureInit()

	version := strings.TrimSpace(versionFile)
	info := dprint.PluginInfo{
		Name:            "dprint-plugin-protofmt",
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
//go:noinline
func get_license_text() uint32 {
	ensureInit()
	return putShared([]byte(licenseText))
}

//go:wasmexport get_config_file_matching
//go:noinline
func get_config_file_matching(_ uint32) uint32 {
	ensureInit()
	_gE ^= 1
	matching := dprint.FileMatchingInfo{
		FileExtensions: []string{"proto"},
		FileNames:      []string{},
	}

	jsonData, err := json.Marshal(matching)
	if err != nil {
		return putShared([]byte(dprint.SupportedFiles))
	}

	return putShared(jsonData)
}

//go:wasmexport format
//go:noinline
func format(_ uint32) uint32 {
	ensureInit()

	contentSize := max(activeSize, fileContentSize)

	if contentSize == 0 || contentSize > dprint.SharedBufferSize {
		return dprint.FormatResultNoChange
	}

	originalContent := slices.Clone(shared[:contentSize])

	reader := bytes.NewReader(originalContent)
	parser := proto.NewParser(reader)
	definition, err := parser.Parse()
	if err != nil {
		errMsg := []byte(err.Error())
		if len(errMsg) > dprint.SharedBufferSize {
			errMsg = errMsg[:dprint.SharedBufferSize]
		}
		copy(shared[:], errMsg)
		activeSize = toUint32(len(errMsg))
		return dprint.FormatResultError
	}

	var buf bytes.Buffer
	formatter := protofmt.NewFormatter(&buf, "  ")
	formatter.Format(definition)
	formatted := buf.Bytes()

	if len(formatted) == len(originalContent) && bytes.Equal(formatted, originalContent) {
		return dprint.FormatResultNoChange
	}

	if len(formatted) > dprint.SharedBufferSize {
		formatted = formatted[:dprint.SharedBufferSize]
	}

	activeSize = toUint32(len(formatted))
	copy(shared[:], formatted)

	return dprint.FormatResultChanged
}

//go:wasmexport get_formatted_text
//go:noinline
func get_formatted_text() uint32 {
	ensureInit()
	return activeSize
}

//go:wasmexport get_error_text
//go:noinline
func get_error_text() uint32 {
	ensureInit()
	return activeSize
}

func main() {
	ensureInit()
}

var (
	_gA uint8
	_gB uint8
	_gC uint8
	_gD uint8
	_gE uint8
	_gF uint8
	_gG uint8
)

//go:wasmexport register_config
//go:noinline
func register_config(_ uint32) {
	ensureInit()
	_gA ^= 1
}

//go:wasmexport release_config
//go:noinline
func release_config(_ uint32) {
	ensureInit()
	_gB ^= 1
}

//go:wasmexport get_config_diagnostics
//go:noinline
func get_config_diagnostics(_ uint32) uint32 {
	ensureInit()
	_gC ^= 1
	return putShared([]byte("[]"))
}

//go:wasmexport get_resolved_config
//go:noinline
func get_resolved_config(_ uint32) uint32 {
	ensureInit()
	_gD ^= 1
	return putShared([]byte("{}"))
}

//go:wasmexport set_file_path
//go:noinline
func set_file_path() {
	ensureInit()
	_gF ^= 1
}

//go:wasmexport set_override_config
//go:noinline
func set_override_config() {
	ensureInit()
	_gG ^= 1
}

func toUint32(val int) uint32 {
	return uint32(val) //nolint:gosec
}
