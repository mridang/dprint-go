package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	cueformat "cuelang.org/go/cue/format"
	"github.com/mridang/dprint-plugin-go/internal/dprint"
)

func TestAPI_CoversExports(t *testing.T) {
	resetState()
	assertRuntimeExports(t)
	assertMetadata(t)
	assertConfigRoundTrip(t)
	assertFormatRoundTrip(t)
}

func assertRuntimeExports(t *testing.T) {
	t.Helper()

	if got := get_shared_bytes_ptr(); got == 0 {
		t.Fatalf("get_shared_bytes_ptr returned 0")
	}
	if got := dprint_plugin_version_4(); got != dprint.PluginSchemaVersion {
		t.Fatalf("dprint_plugin_version_4 = %d; want %d", got, dprint.PluginSchemaVersion)
	}
}

func assertMetadata(t *testing.T) {
	t.Helper()

	infoSize := get_plugin_info()
	info := decodePluginInfo(t, infoSize)
	if info.Version != strings.TrimSpace(versionFile) {
		t.Fatalf("plugin version = %q; want %q", info.Version, strings.TrimSpace(versionFile))
	}
	if info.ConfigKey != "cue" {
		t.Fatalf("plugin config key = %q; want %q", info.ConfigKey, "cue")
	}

	licenseSize := get_license_text()
	if licenseSize == 0 {
		t.Fatalf("license text is empty")
	}

	matchSize := get_config_file_matching(0)
	match := decodeFileMatching(t, matchSize)
	if len(match.FileExtensions) != 1 || match.FileExtensions[0] != "cue" {
		t.Fatalf("file extensions = %v; want [cue]", match.FileExtensions)
	}
}

func assertConfigRoundTrip(t *testing.T) {
	t.Helper()

	loadShared([]byte(`{"useTabs":true,"indentWidth":2,"simplify":true}`))
	register_config(0)
	resolvedSize := get_resolved_config(0)
	var resolved map[string]any
	if err := json.Unmarshal(shared[:resolvedSize], &resolved); err != nil {
		t.Fatalf("unmarshal resolved config: %v", err)
	}
	if got, ok := resolved["useTabs"].(bool); !ok || !got {
		t.Fatalf("resolved useTabs = %v; want true", resolved["useTabs"])
	}
	if got, ok := resolved["indentWidth"].(float64); !ok || got != 2 {
		t.Fatalf("resolved indentWidth = %v; want 2", resolved["indentWidth"])
	}
	if got, ok := resolved["simplify"].(bool); !ok || !got {
		t.Fatalf("resolved simplify = %v; want true", resolved["simplify"])
	}
	diagSize := get_config_diagnostics(0)
	if got := string(shared[:diagSize]); got != "[]" {
		t.Fatalf("config diagnostics = %q; want []", got)
	}

	set_file_path()
	set_override_config()
	release_config(0)
}

//goland:noinspection DuplicatedCode
func assertFormatRoundTrip(t *testing.T) {
	t.Helper()

	bad := []byte(`package main
import "list"
foo:  {
	bar: "baz"
	num: 1
}`)
	want, err := cueformat.Source(bad)
	if err != nil {
		t.Fatalf("cue/format failed: %v", err)
	}
	loadShared(bad)
	if got := format(0); got != dprint.FormatResultChanged {
		t.Fatalf("format result = %d; want changed", got)
	}
	outSize := get_formatted_text()
	if !bytes.Equal(shared[:outSize], want) {
		t.Fatalf("formatted output did not match cue/format")
	}

	loadShared([]byte("package"))
	if got := format(0); got != dprint.FormatResultError {
		t.Fatalf("format result = %d; want error", got)
	}
	if errSize := get_error_text(); errSize == 0 {
		t.Fatalf("expected error text")
	}

	loadShared(want)
	if got := format(0); got != dprint.FormatResultNoChange {
		t.Fatalf("format result = %d; want no change", got)
	}
}

func resetState() {
	initialized = false
	activeSize = 0
	fileContentSize = 0
	currentConfig = defaultConfig()
	for i := range shared {
		shared[i] = 0
	}
}

func loadShared(b []byte) {
	clear_shared_bytes(uint32(len(b)))
	copy(shared[:], b)
}

func decodePluginInfo(t *testing.T, size uint32) dprint.PluginInfo {
	t.Helper()
	var info dprint.PluginInfo
	if err := json.Unmarshal(shared[:size], &info); err != nil {
		t.Fatalf("unmarshal plugin info: %v", err)
	}
	return info
}

func decodeFileMatching(t *testing.T, size uint32) dprint.FileMatchingInfo {
	t.Helper()
	var info dprint.FileMatchingInfo
	if err := json.Unmarshal(shared[:size], &info); err != nil {
		t.Fatalf("unmarshal file matching: %v", err)
	}
	return info
}
