package main

//revive:disable:var-naming // WASM exports require underscore names.

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"slices"
	"strings"
	"unsafe"

	"github.com/mridang/dprint-plugin-go/internal/dprint"
	"mvdan.cc/sh/v3/syntax"
)

var currentConfig = defaultConfig() //nolint:gochecknoglobals // required for wasm config storage.

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

// putShared copies bytes into the shared buffer and returns their length.
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
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction,GoUnusedFunction
func get_shared_bytes_ptr() uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	return uint32(uintptr(unsafe.Pointer(&shared[0])))
}

//go:wasmexport clear_shared_bytes
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction
func clear_shared_bytes(size uint32) uint32 { //nolint:staticcheck // required for wasm export name.
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
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction
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
//goland:noinspection GoUnusedFunction,GoSnakeCaseUsage
func get_plugin_info() uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()

	version := strings.TrimSpace(versionFile)
	info := dprint.PluginInfo{
		Name:            "dprint-plugin-shfmt",
		Version:         version,
		ConfigKey:       "go-shfmt",
		FileExtensions:  []string{"sh", "bash"},
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
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction
func get_license_text() uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	return putShared([]byte(licenseText))
}

//go:wasmexport get_config_file_matching
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage,GoSnakeCaseUsage,GoUnusedFunction,GoUnusedParameter
func get_config_file_matching(_ uint32) uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	_gE ^= 1
	matching := dprint.FileMatchingInfo{
		FileExtensions: []string{"sh", "bash"},
		FileNames:      []string{},
	}
	data, err := json.Marshal(matching)
	if err != nil {
		return putShared([]byte(dprint.SupportedFiles))
	}
	return putShared(data)
}

// Config maps a subset of shfmt options. Defaults aim to match shfmt defaults.
// Extend as needed.
type Config struct {
	Indent           int    `json:"indent"`           // spaces (0 means shfmt default=0 -> tabs)
	BinaryNextLine   bool   `json:"binaryNextLine"`   // place binary ops at line start
	SpaceRedirects   bool   `json:"spaceRedirects"`   // space before redirects
	KeepPadding      bool   `json:"keepPadding"`      // keep alignment spaces
	FunctionNextLine bool   `json:"functionNextLine"` // place function body on next line
	SwitchCaseIndent bool   `json:"switchCaseIndent"` // indent switch cases
	KeepComments     bool   `json:"keepComments"`     // preserve comments
	Language         string `json:"language"`         // "auto" (default), "posix", "bash", "mksh"
}

func defaultConfig() Config {
	return Config{
		Indent:           0,
		BinaryNextLine:   false,
		SpaceRedirects:   false,
		KeepPadding:      false,
		FunctionNextLine: false,
		SwitchCaseIndent: false,
		KeepComments:     true,
		Language:         "auto",
	}
}

//go:wasmexport register_config
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage,GoSnakeCaseUsage,GoUnusedFunction,GoUnusedParameter
func register_config(_ uint32) { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	_gA ^= 1
	buf := make([]byte, activeSize)
	copy(buf, shared[:activeSize])
	cfg := defaultConfig()
	if len(buf) != 0 {
		_ = json.Unmarshal(buf, &cfg) // tolerate unknown fields
	}
	currentConfig = cfg
	// v2 ABI doesn't return a _ from this function
}

//go:wasmexport get_resolved_config
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage,GoSnakeCaseUsage,GoUnusedFunction,GoUnusedParameter
func get_resolved_config(_ uint32) uint32 { //nolint:staticcheck // required for wasm export name.
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
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction,GoUnusedParameter,DuplicatedCode
func format(_ uint32) uint32 {
	ensureInit()

	contentSize := max(activeSize, fileContentSize)
	if contentSize == 0 || contentSize > dprint.SharedBufferSize {
		return dprint.FormatResultNoChange
	}

	input := slices.Clone(shared[:contentSize])

	formatted, err := formatShell(input, currentConfig)
	if err != nil {
		errMsg := []byte(err.Error())
		if len(errMsg) > dprint.SharedBufferSize {
			errMsg = errMsg[:dprint.SharedBufferSize]
		}
		copy(shared[:], errMsg)
		activeSize = toUint32(len(errMsg))
		return dprint.FormatResultError
	}

	// unchanged fast path
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
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction
func get_formatted_text() uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	return activeSize
}

//go:wasmexport get_error_text
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction
func get_error_text() uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	return activeSize
}

func formatShell(src []byte, cfg Config) ([]byte, error) {
	parser := syntax.NewParser(parserOptions(cfg)...)
	file, err := parser.Parse(bytes.NewReader(src), "")
	if err != nil {
		return nil, err
	}
	var out strings.Builder
	printer := syntax.NewPrinter(printerOptions(cfg)...)
	if err = printer.Print(&out, file); err != nil {
		return nil, err
	}
	return []byte(out.String()), nil
}

func parserOptions(cfg Config) []syntax.ParserOption {
	var opts []syntax.ParserOption
	switch strings.ToLower(strings.TrimSpace(cfg.Language)) {
	case "posix":
		opts = append(opts, syntax.Variant(syntax.LangPOSIX))
	case "bash":
		opts = append(opts, syntax.Variant(syntax.LangBash))
	case "mksh":
		opts = append(opts, syntax.Variant(syntax.LangMirBSDKorn))
	default:
		// auto: let parser detect; no variant option
	}
	if cfg.KeepComments {
		opts = append(opts, syntax.KeepComments(true))
	}
	return opts
}

//goland:noinspection GoDeprecation
func printerOptions(cfg Config) []syntax.PrinterOption {
	var opts []syntax.PrinterOption
	if cfg.Indent > 0 {
		opts = append(opts, syntax.Indent(uint(cfg.Indent)))
	}
	if cfg.BinaryNextLine {
		opts = append(opts, syntax.BinaryNextLine(true))
	}
	if cfg.SpaceRedirects {
		opts = append(opts, syntax.SpaceRedirects(true))
	}
	if cfg.KeepPadding {
		opts = append(opts, syntax.KeepPadding(true)) //nolint:staticcheck // since it is used
	}
	if cfg.FunctionNextLine {
		opts = append(opts, syntax.FunctionNextLine(true))
	}
	if cfg.SwitchCaseIndent {
		opts = append(opts, syntax.SwitchCaseIndent(true))
	}
	return opts
}

// The main is the entry point for the WASM module.
func main() {
	ensureInit()
}

// Dummy globals to prevent identical code folding.
var (
	_gA uint8 //nolint:gochecknoglobals,unused // required to keep wasm exports stable.
	_gB uint8 //nolint:gochecknoglobals,unused // required to keep wasm exports stable.
	_gC uint8 //nolint:gochecknoglobals,unused // required to keep wasm exports stable.
	_gD uint8 //nolint:gochecknoglobals,unused // required to keep wasm exports stable.
	_gE uint8 //nolint:gochecknoglobals,unused // required to keep wasm exports stable.
	_gF uint8 //nolint:gochecknoglobals,unused // required to keep wasm exports stable.
	_gG uint8 //nolint:gochecknoglobals,unused // required to keep wasm exports stable.
)

//go:wasmexport release_config
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoUnusedFunction,GoUnusedParameter,GoSnakeCaseUsage
func release_config(_ uint32) { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	_gB ^= 1
}

//go:wasmexport get_config_diagnostics
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoUnusedFunction,GoUnusedParameter,GoSnakeCaseUsage
func get_config_diagnostics(_ uint32) uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	_gC ^= 1
	return putShared([]byte("[]"))
}

// set_file_path is called by the CLI to set the file path in the shared buffer.
// The plugin can read this path if needed for context-specific formatting.
// See: https://dprint.dev/plugins/wasm/#set_file_path
//
//go:wasmexport set_file_path
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoUnusedFunction, GoSnakeCaseUsage
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
//goland:noinspection GoUnusedFunction,GoSnakeCaseUsage
func set_override_config() { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	_gG ^= 1
}

// toUint32 converts an int to uint32, suppressing the G115 overflow warning.
func toUint32(val int) uint32 {
	// This cast from int (64-bit) to uint32 (32-bit) is required by the WASM ABI.
	return uint32(val) //nolint:gosec // required for wasm pointer sizes.
}
