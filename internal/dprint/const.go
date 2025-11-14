package dprint

const SupportedFiles = `{"fileExtensions":[],"fileNames":[]}`

// Constants for the dprint WASM ABI
const (
	// PluginSchemaVersion Schema version supported by this plugin
	PluginSchemaVersion = 4

	// SharedBufferSize Shared buffer size (1MB) for communication between host and plugin
	SharedBufferSize = 1 << 20

	// FormatResultNoChange Format return values as defined by dprint WASM ABI
	FormatResultNoChange = 0 // No formatting changes needed
	FormatResultChanged  = 1 // Content was formatted and changed
	FormatResultError    = 2 // Formatting error occurred
)
