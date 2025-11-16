package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unsafe"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"

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
// This must be called before any other plugin operations.
func ensureInit() {
	if !initialized {
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
//goland:noinspection GoSnakeCaseUsage,GoUnusedFunction,GoUnusedFunction
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

// get_plugin_info serializes and returns the plugin information as JSON.
// This includes the plugin name, version, configuration key, and supported
// file extensions. See: https://dprint.dev/plugins/wasm/#get_plugin_info
//
//go:wasmexport get_plugin_info
//go:noinline
//goland:noinspection GoUnusedFunction,GoSnakeCaseUsage
func get_plugin_info() uint32 { //nolint:revive,unused,staticcheck // because it is exported
	ensureInit()

	version := strings.TrimSpace(versionFile)
	info := dprint.PluginInfo{
		Name:            "dprint-plugin-gohcl",
		Version:         version,
		ConfigKey:       "go-hcl",
		FileExtensions:  []string{"tf", "tfvars", "tftest.hcl", "tfmock.hcl", "hcl"},
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
//goland:noinspection GoSnakeCaseUsage,GoSnakeCaseUsage,GoUnusedFunction,GoUnusedParameter
func get_config_file_matching(_ uint32) uint32 { //nolint:revive,unused,staticcheck // because it is exported
	ensureInit()
	_gE ^= 1
	matching := dprint.FileMatchingInfo{
		FileExtensions: []string{"tf", "tfvars", "tftest.hcl", "tfmock.hcl", "hcl"},
		FileNames:      []string{},
	}
	data, err := json.Marshal(matching)
	if err != nil {
		return putShared([]byte(dprint.SupportedFiles))
	}
	return putShared(data)
}

// Config for the HCL formatter. Currently there are no user-exposed
// options, but this struct is kept for future extensibility and to
// mirror the shfmt plugin's configuration pattern.
type Config struct {
	// Reserved for future configuration options.
}

func defaultConfig() Config {
	return Config{}
}

//go:wasmexport register_config
//go:noinline
//goland:noinspection GoSnakeCaseUsage,GoSnakeCaseUsage,GoUnusedFunction,GoUnusedParameter
func register_config(_ uint32) { //nolint:revive,unused,staticcheck // because it is exported
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
//goland:noinspection GoSnakeCaseUsage,GoSnakeCaseUsage,GoUnusedFunction,GoUnusedParameter
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

	formatted, err := formatHCL(input, currentConfig)
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

// formatHCL formats HCL source code using logic adapted from Terraform's
// "terraform fmt" implementation, via the hclwrite and hclsyntax packages.
func formatHCL(src []byte, _ Config) ([]byte, error) {
	// First check that the file is parseable as native HCL syntax.
	_, syntaxDiags := hclsyntax.ParseConfig(src, "", hcl.Pos{Line: 1, Column: 1})
	if syntaxDiags.HasErrors() {
		return nil, fmt.Errorf("%s", syntaxDiags.Error())
	}

	// Then parse with hclwrite so we can manipulate tokens and regenerate
	// the formatted output.
	f, diags := hclwrite.ParseConfig(src, "", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, fmt.Errorf("%s", diags.Error())
	}
	if f == nil {
		return nil, errors.New("failed to parse HCL config")
	}

	formatter := &hclFormatter{}
	formatter.formatBody(f.Body(), nil)

	return f.Bytes(), nil
}

// hclFormatter adapts Terraform's fmt formatting logic for use in this plugin.
type hclFormatter struct{}

const (
	// minInterpolationTokens is the minimum number of tokens required for a "${ ... }" sequence.
	minInterpolationTokens = 5
	// parenPairTokens is the number of tokens needed for adding parentheses (open and close).
	parenPairTokens = 2
	// legacyTypeTokens is the number of tokens in a legacy quoted type expression like "string".
	legacyTypeTokens = 3
)

func (f *hclFormatter) formatBody(body *hclwrite.Body, inBlocks []string) {
	attrs := body.Attributes()
	for name, attr := range attrs {
		if len(inBlocks) == 1 && inBlocks[0] == "variable" && name == "type" {
			cleanedExprTokens := f.formatTypeExpr(attr.Expr().BuildTokens(nil))
			body.SetAttributeRaw(name, cleanedExprTokens)
			continue
		}
		cleanedExprTokens := f.formatValueExpr(attr.Expr().BuildTokens(nil))
		body.SetAttributeRaw(name, cleanedExprTokens)
	}

	blocks := body.Blocks()
	for _, block := range blocks {
		// Normalize the label formatting, removing any weird stuff like
		// interleaved inline comments and using the idiomatic quoted
		// label syntax.
		block.SetLabels(block.Labels())

		inBlocks = append(inBlocks, block.Type())
		f.formatBody(block.Body(), inBlocks)
	}
}

func (f *hclFormatter) formatValueExpr(tokens hclwrite.Tokens) hclwrite.Tokens {
	if len(tokens) < minInterpolationTokens {
		// Can't possibly be a "${ ... }" sequence without at least enough
		// tokens for the delimiters and one token inside them.
		return tokens
	}

	if !f.isInterpolationSequence(tokens) {
		return tokens
	}

	inside := tokens[2 : len(tokens)-2]

	if !f.isSingleInterpolation(inside) {
		return tokens
	}

	// If we got down here without an early return then this looks like
	// an unwrappable sequence, but we'll trim any leading and trailing
	// newlines that might result in an invalid result if we were to
	// naively trim something like this:
	// "${
	//    foo
	// }"
	trimmed := f.trimNewlines(inside)

	// Finally, we check if the unwrapped expression is on multiple lines. If
	// so, we ensure that it is surrounded by parenthesis to make sure that it
	// parses correctly after unwrapping. This may be redundant in some cases,
	// but is required for at least multi-line ternary expressions.
	return f.wrapMultiLineIfNeeded(trimmed)
}

// isInterpolationSequence checks if tokens represent a "${ ... }" interpolation sequence.
func (f *hclFormatter) isInterpolationSequence(tokens hclwrite.Tokens) bool {
	oQuote := tokens[0]
	oBrace := tokens[1]
	cBrace := tokens[len(tokens)-2]
	cQuote := tokens[len(tokens)-1]
	return oQuote.Type == hclsyntax.TokenOQuote &&
		oBrace.Type == hclsyntax.TokenTemplateInterp &&
		cBrace.Type == hclsyntax.TokenTemplateSeqEnd &&
		cQuote.Type == hclsyntax.TokenCQuote
}

// isSingleInterpolation checks if the interior tokens represent a single interpolation.
func (f *hclFormatter) isSingleInterpolation(inside hclwrite.Tokens) bool {
	// We're only interested in sequences that are provable to be single
	// interpolation sequences, which we'll determine by hunting inside
	// the interior tokens for any other interpolation sequences. This is
	// likely to produce false negatives sometimes, but that's better than
	// false positives and we're mainly interested in catching the easy cases
	// here.
	quotes := 0
	for _, token := range inside {
		if token.Type == hclsyntax.TokenOQuote {
			quotes++
			continue
		}
		if token.Type == hclsyntax.TokenCQuote {
			quotes--
			continue
		}
		if quotes > 0 {
			// Interpolation sequences inside nested quotes are okay, because
			// they are part of a nested expression.
			// "${foo("${bar}")}"
			continue
		}
		if token.Type == hclsyntax.TokenTemplateInterp || token.Type == hclsyntax.TokenTemplateSeqEnd {
			// We've found another template delimiter within our interior
			// tokens, which suggests that we've found something like this:
			// "${foo}${bar}"
			// That isn't unwrappable, so we'll leave the whole expression alone.
			return false
		}
		if token.Type == hclsyntax.TokenQuotedLit {
			// If there's any literal characters in the outermost
			// quoted sequence then it is not unwrappable.
			return false
		}
	}
	return true
}

// wrapMultiLineIfNeeded wraps multi-line expressions in parentheses if not already wrapped.
func (f *hclFormatter) wrapMultiLineIfNeeded(trimmed hclwrite.Tokens) hclwrite.Tokens {
	isMultiLine := false
	hasLeadingParen := false
	hasTrailingParen := false
	for i, token := range trimmed {
		switch {
		case i == 0 && token.Type == hclsyntax.TokenOParen:
			hasLeadingParen = true
		case token.Type == hclsyntax.TokenNewline:
			isMultiLine = true
		case i == len(trimmed)-1 && token.Type == hclsyntax.TokenCParen:
			hasTrailingParen = true
		}
	}
	if isMultiLine && (!hasLeadingParen || !hasTrailingParen) {
		wrapped := make(hclwrite.Tokens, 0, len(trimmed)+parenPairTokens)
		wrapped = append(wrapped, &hclwrite.Token{
			Type:  hclsyntax.TokenOParen,
			Bytes: []byte("("),
		})
		wrapped = append(wrapped, trimmed...)
		wrapped = append(wrapped, &hclwrite.Token{
			Type:  hclsyntax.TokenCParen,
			Bytes: []byte(")"),
		})

		return wrapped
	}

	return trimmed
}

func (f *hclFormatter) formatTypeExpr(tokens hclwrite.Tokens) hclwrite.Tokens {
	switch len(tokens) {
	case 1:
		kwTok := tokens[0]
		if kwTok.Type != hclsyntax.TokenIdent {
			// Not a single type keyword, then.
			return tokens
		}

		// Collection types without an explicit element type mean
		// the element type is "any", so we'll normalize that.
		switch string(kwTok.Bytes) {
		case "list", "map", "set":
			return hclwrite.Tokens{
				kwTok,
				{
					Type:  hclsyntax.TokenOParen,
					Bytes: []byte("("),
				},
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("any"),
				},
				{
					Type:  hclsyntax.TokenCParen,
					Bytes: []byte(")"),
				},
			}
		default:
			return tokens
		}

	case legacyTypeTokens:
		// A pre-0.12 legacy quoted string type, like "string".
		oQuote := tokens[0]
		strTok := tokens[1]
		cQuote := tokens[2]
		if oQuote.Type != hclsyntax.TokenOQuote ||
			strTok.Type != hclsyntax.TokenQuotedLit ||
			cQuote.Type != hclsyntax.TokenCQuote {
			// Not a quoted string sequence, then.
			return tokens
		}

		// Because this quoted syntax is from Terraform 0.11 and
		// earlier, which didn't have the idea of "any" as an,
		// element type, we use string as the default element
		// type. That will avoid oddities if somehow the configuration
		// was relying on numeric values being auto-converted to
		// string, as 0.11 would do. This mimicks what terraform
		// 0.12upgrade used to do, because we'd found real-world
		// modules that were depending on the auto-stringing.)
		switch string(strTok.Bytes) {
		case "string":
			return hclwrite.Tokens{
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("string"),
				},
			}
		case "list":
			return hclwrite.Tokens{
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("list"),
				},
				{
					Type:  hclsyntax.TokenOParen,
					Bytes: []byte("("),
				},
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("string"),
				},
				{
					Type:  hclsyntax.TokenCParen,
					Bytes: []byte(")"),
				},
			}
		case "map":
			return hclwrite.Tokens{
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("map"),
				},
				{
					Type:  hclsyntax.TokenOParen,
					Bytes: []byte("("),
				},
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("string"),
				},
				{
					Type:  hclsyntax.TokenCParen,
					Bytes: []byte(")"),
				},
			}
		default:
			// Something else we're not expecting, then.
			return tokens
		}
	default:
		return tokens
	}
}

func (f *hclFormatter) trimNewlines(tokens hclwrite.Tokens) hclwrite.Tokens {
	if len(tokens) == 0 {
		return nil
	}
	var start, end int
	for start = range tokens {
		if tokens[start].Type != hclsyntax.TokenNewline {
			break
		}
	}
	for end = len(tokens); end > 0; end-- {
		if tokens[end-1].Type != hclsyntax.TokenNewline {
			break
		}
	}
	return tokens[start:end]
}

// The main is the entry point for the WASM module.
func main() {
	ensureInit()
}

// Dummy globals to prevent Identical Code Folding optimization from
// merging these placeholder functions.
var (
	_gA uint8 //nolint:unused, gochecknoglobals, var-naming // CGO global variable
	_gB uint8 //nolint:unused, gochecknoglobals, var-naming // CGO global variable
	_gC uint8 //nolint:unused, gochecknoglobals, var-naming // CGO global variable
	_gD uint8 //nolint:unused, gochecknoglobals, var-naming // CGO global variable
	_gE uint8 //nolint:unused, gochecknoglobals, var-naming // CGO global variable
	_gF uint8 //nolint:unused, gochecknoglobals, var-naming // CGO global variable
	_gG uint8 //nolint:unused, gochecknoglobals, var-naming // CGO global variable
)

//go:wasmexport release_config
//go:noinline
//goland:noinspection GoUnusedFunction,GoUnusedParameter,GoSnakeCaseUsage
func release_config(_ uint32) { //nolint:revive,unused,staticcheck // because it is exported
	ensureInit()
	_gB ^= 1
}

//go:wasmexport get_config_diagnostics
//go:noinline
//goland:noinspection GoUnusedFunction,GoUnusedParameter,GoSnakeCaseUsage
func get_config_diagnostics(_ uint32) uint32 { //nolint:revive,unused,staticcheck // because it is exported
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
//goland:noinspection GoUnusedFunction, GoSnakeCaseUsage
func set_file_path() { //nolint:revive,unused,staticcheck // because it is exported
	ensureInit()
	_gF ^= 1
}

// set_override_config is called by the CLI to set override configuration.
// This allows per-file or per-directory configuration overrides.
// See: https://dprint.dev/plugins/wasm/#set_override_config
//
//go:wasmexport set_override_config
//go:noinline
//goland:noinspection GoUnusedFunction,GoSnakeCaseUsage
func set_override_config() { //nolint:revive,unused,staticcheck // because it is exported
	ensureInit()
	_gG ^= 1
}

// toUint32 converts an int to uint32, suppressing the G115 overflow warning.
func toUint32(val int) uint32 { //nolint:unused // because it is exported
	// This cast from int (64-bit) to uint32 (32-bit) could
	// overflow, so we suppress the gosec linter.
	return uint32(val) //nolint:gosec // since dprint really needs uint32
}
