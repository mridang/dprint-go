package main

//revive:disable:var-naming // WASM exports require underscore names.

import (
	"bytes"
	_ "embed"
	"slices"
	"strconv"
	"strings"
	"unsafe"

	cueformat "cuelang.org/go/cue/format"
	"github.com/mridang/dprint-plugin-go/internal/dprint"
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
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction
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

//go:wasmexport get_plugin_info
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoUnusedFunction,GoSnakeCaseUsage
func get_plugin_info() uint32 { //nolint:staticcheck // required for wasm export name.
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
	return putShared(pluginInfoJSON(info))
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
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction,GoUnusedParameter
func get_config_file_matching(_ uint32) uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	_gE ^= 1
	matching := dprint.FileMatchingInfo{
		FileExtensions: []string{"cue"},
		FileNames:      []string{},
	}
	return putShared(fileMatchingJSON(matching))
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
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction,GoUnusedParameter
func register_config(_ uint32) { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	_gA ^= 1
	buf := make([]byte, activeSize)
	copy(buf, shared[:activeSize])
	cfg := defaultConfig()
	if len(buf) != 0 {
		parseConfigJSON(buf, &cfg)
	}
	currentConfig = cfg
}

//go:wasmexport get_resolved_config
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction,GoUnusedParameter
func get_resolved_config(_ uint32) uint32 { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	_gD ^= 1
	return putShared(configJSON(currentConfig))
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

//go:wasmexport set_file_path
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoUnusedFunction, GoSnakeCaseUsage
func set_file_path() { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	_gF ^= 1
}

//go:wasmexport set_override_config
//go:noinline
//nolint:revive,staticcheck // WASM exports require snake_case names.
//goland:noinspection GoUnusedFunction,GoSnakeCaseUsage
func set_override_config() { //nolint:staticcheck // required for wasm export name.
	ensureInit()
	_gG ^= 1
}

func toUint32(val int) uint32 {
	return uint32(val) //nolint:gosec // required for wasm pointer sizes.
}

const (
	pluginInfoBufSize = 128
	fileMatchBufSize  = 64
	configBufSize     = 64
	jsonBase10        = 10
	configKeyUseTabs  = "useTabs"
	configKeyIndent   = "indentWidth"
	configKeySimplify = "simplify"
)

func pluginInfoJSON(info dprint.PluginInfo) []byte {
	buf := make([]byte, 0, pluginInfoBufSize)
	buf = append(buf, '{')
	buf = appendJSONField(buf, "name", func(b []byte) []byte {
		return appendJSONString(b, info.Name)
	})
	buf = appendJSONField(buf, "version", func(b []byte) []byte {
		return appendJSONString(b, info.Version)
	})
	buf = appendJSONField(buf, "configKey", func(b []byte) []byte {
		return appendJSONString(b, info.ConfigKey)
	})
	buf = appendJSONField(buf, "fileExtensions", func(b []byte) []byte {
		return appendJSONArrayStrings(b, info.FileExtensions)
	})
	buf = appendJSONField(buf, "fileNames", func(b []byte) []byte {
		return appendJSONArrayStrings(b, info.FileNames)
	})
	buf = appendJSONField(buf, "helpUrl", func(b []byte) []byte {
		return appendJSONString(b, info.HelpURL)
	})
	buf = appendJSONField(buf, "configSchemaUrl", func(b []byte) []byte {
		return appendJSONString(b, info.ConfigSchemaURL)
	})
	buf = append(buf, '}')
	return buf
}

func fileMatchingJSON(info dprint.FileMatchingInfo) []byte {
	buf := make([]byte, 0, fileMatchBufSize)
	buf = append(buf, '{')
	buf = appendJSONField(buf, "fileExtensions", func(b []byte) []byte {
		return appendJSONArrayStrings(b, info.FileExtensions)
	})
	buf = appendJSONField(buf, "fileNames", func(b []byte) []byte {
		return appendJSONArrayStrings(b, info.FileNames)
	})
	buf = append(buf, '}')
	return buf
}

func configJSON(cfg Config) []byte {
	buf := make([]byte, 0, configBufSize)
	buf = append(buf, '{')
	first := true
	if cfg.UseTabs != nil {
		buf = appendJSONFieldConditional(buf, &first, configKeyUseTabs, func(b []byte) []byte {
			return appendJSONBool(b, *cfg.UseTabs)
		})
	}
	if cfg.IndentWidth != nil {
		buf = appendJSONFieldConditional(buf, &first, configKeyIndent, func(b []byte) []byte {
			return strconv.AppendInt(b, int64(*cfg.IndentWidth), jsonBase10)
		})
	}
	if cfg.Simplify != nil {
		buf = appendJSONFieldConditional(buf, &first, configKeySimplify, func(b []byte) []byte {
			return appendJSONBool(b, *cfg.Simplify)
		})
	}
	buf = append(buf, '}')
	return buf
}

func parseConfigJSON(b []byte, cfg *Config) {
	i := 0
	skipWS(b, &i)
	if !consumeByte(b, &i, '{') {
		return
	}
	for {
		skipWS(b, &i)
		if consumeByte(b, &i, '}') {
			return
		}
		key, ok := readString(b, &i)
		if !ok {
			return
		}
		skipWS(b, &i)
		if !consumeByte(b, &i, ':') {
			return
		}
		skipWS(b, &i)
		if !applyConfigValue(b, &i, cfg, key) {
			return
		}
		skipWS(b, &i)
		if consumeByte(b, &i, ',') {
			continue
		}
		if consumeByte(b, &i, '}') {
			return
		}
		return
	}
}

func skipWS(b []byte, i *int) {
	for *i < len(b) {
		switch b[*i] {
		case ' ', '\t', '\n', '\r':
			*i++
		default:
			return
		}
	}
}

func consumeByte(b []byte, i *int, want byte) bool {
	if *i >= len(b) || b[*i] != want {
		return false
	}
	*i++
	return true
}

func readString(b []byte, i *int) (string, bool) {
	if !consumeByte(b, i, '"') {
		return "", false
	}
	start := *i
	for *i < len(b) {
		if b[*i] == '\\' {
			*i += 2
			continue
		}
		if b[*i] == '"' {
			s := string(b[start:*i])
			*i++
			return s, true
		}
		*i++
	}
	return "", false
}

func applyConfigValue(b []byte, i *int, cfg *Config, key string) bool {
	switch key {
	case configKeyUseTabs, configKeySimplify:
		val, ok := readBool(b, i)
		if !ok {
			return false
		}
		if key == configKeyUseTabs {
			cfg.UseTabs = &val
		} else {
			cfg.Simplify = &val
		}
		return true
	case configKeyIndent:
		val, ok := readInt(b, i)
		if !ok {
			return false
		}
		cfg.IndentWidth = &val
		return true
	default:
		return false
	}
}

func readBool(b []byte, i *int) (bool, bool) {
	switch {
	case matchLiteral(b, *i, "true"):
		*i += len("true")
		return true, true
	case matchLiteral(b, *i, "false"):
		*i += len("false")
		return false, true
	default:
		return false, false
	}
}

func readInt(b []byte, i *int) (int, bool) {
	start := *i
	if *i < len(b) && (b[*i] == '-' || b[*i] == '+') {
		*i++
	}
	for *i < len(b) && b[*i] >= '0' && b[*i] <= '9' {
		*i++
	}
	if start == *i {
		return 0, false
	}
	val, err := strconv.Atoi(string(b[start:*i]))
	if err != nil {
		return 0, false
	}
	return val, true
}

func matchLiteral(b []byte, i int, lit string) bool {
	if i < 0 || i+len(lit) > len(b) {
		return false
	}
	return string(b[i:i+len(lit)]) == lit
}

func appendJSONField(buf []byte, key string, appendValue func([]byte) []byte) []byte {
	buf = appendJSONFieldConditional(buf, nil, key, appendValue)
	return buf
}

func appendJSONFieldConditional(buf []byte, first *bool, key string, appendValue func([]byte) []byte) []byte {
	if first != nil {
		if !*first {
			buf = append(buf, ',')
		}
		*first = false
	} else if len(buf) > 1 {
		buf = append(buf, ',')
	}
	buf = appendJSONString(buf, key)
	buf = append(buf, ':')
	buf = appendValue(buf)
	return buf
}

func appendJSONString(buf []byte, s string) []byte {
	return strconv.AppendQuote(buf, s)
}

func appendJSONArrayStrings(buf []byte, items []string) []byte {
	buf = append(buf, '[')
	for i, item := range items {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = appendJSONString(buf, item)
	}
	buf = append(buf, ']')
	return buf
}

func appendJSONBool(buf []byte, v bool) []byte {
	if v {
		return append(buf, 't', 'r', 'u', 'e')
	}
	return append(buf, 'f', 'a', 'l', 's', 'e')
}
