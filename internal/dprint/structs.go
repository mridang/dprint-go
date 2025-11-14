package dprint

// PluginInfo represents the JSON structure returned by get_plugin_info.
// See: https://dprint.dev/plugins/wasm/#get_plugin_info
type PluginInfo struct {
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	ConfigKey       string   `json:"configKey"`
	FileExtensions  []string `json:"fileExtensions"`
	FileNames       []string `json:"fileNames"`
	HelpUrl         string   `json:"helpUrl"`
	ConfigSchemaUrl string   `json:"configSchemaUrl"`
}

// FileMatchingInfo represents the JSON structure returned by
// get_config_file_matching.
type FileMatchingInfo struct {
	FileExtensions []string `json:"fileExtensions"`
	FileNames      []string `json:"fileNames"`
}
