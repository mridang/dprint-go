package main

import (
	"bytes"
	"encoding/json"
	gofmt "go/format"
	"strings"
	"testing"

	"github.com/mridang/dprint-plugin-go/internal/dprint"
)

//goland:noinspection DuplicatedCode,DuplicatedCode,DuplicatedCode
func TestAPI_CoversExports(t *testing.T) {
	resetState()

	if got := get_shared_bytes_ptr(); got == 0 {
		t.Fatalf("get_shared_bytes_ptr returned 0")
	}
	if got := dprint_plugin_version_4(); got != dprint.PluginSchemaVersion {
		t.Fatalf("dprint_plugin_version_4 = %d; want %d", got, dprint.PluginSchemaVersion)
	}

	infoSize := get_plugin_info()
	info := decodePluginInfo(t, infoSize)
	if info.Version != strings.TrimSpace(versionFile) {
		t.Fatalf("plugin version = %q; want %q", info.Version, strings.TrimSpace(versionFile))
	}
	if info.ConfigKey != "go" {
		t.Fatalf("plugin config key = %q; want %q", info.ConfigKey, "go")
	}

	licenseSize := get_license_text()
	if licenseSize == 0 {
		t.Fatalf("license text is empty")
	}

	matchSize := get_config_file_matching(0)
	match := decodeFileMatching(t, matchSize)
	if len(match.FileExtensions) != 1 || match.FileExtensions[0] != "go" {
		t.Fatalf("file extensions = %v; want [go]", match.FileExtensions)
	}

	loadShared([]byte(`{}`))
	register_config(0)
	resolvedSize := get_resolved_config(0)
	if got := string(shared[:resolvedSize]); got != "{}" {
		t.Fatalf("resolved config = %q; want {}", got)
	}
	diagSize := get_config_diagnostics(0)
	if got := string(shared[:diagSize]); got != "[]" {
		t.Fatalf("config diagnostics = %q; want []", got)
	}

	set_file_path()
	set_override_config()
	release_config(0)

	bad := []byte(`package main
import "fmt"
func main(){fmt.Println("ok")}
`)
	want, err := gofmt.Source(bad)
	if err != nil {
		t.Fatalf("go/format failed: %v", err)
	}
	loadShared(bad)
	if got := format(0); got != dprint.FormatResultChanged {
		t.Fatalf("format result = %d; want changed", got)
	}
	outSize := get_formatted_text()
	if !bytes.Equal(shared[:outSize], want) {
		t.Fatalf("formatted output did not match go/format")
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
