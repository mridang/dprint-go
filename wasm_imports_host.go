//go:build tinygo

package main

// This file defines the host import functions provided by the dprint CLI
// for WASM plugins to communicate back to the host environment.
// These are part of the dprint WASM ABI Schema Version 4.
// See: https://dprint.dev/plugins/wasm/#wasm-imports

// host_write_buffer tells the host to write data to the provided WASM memory
// address. This is used for low-level communication between plugin and host.
// See: https://dprint.dev/plugins/wasm/#host_write_buffer
//
//go:wasmimport dprint host_write_buffer
//goland:noinspection GoUnusedFunction,GoSnakeCaseUsage
func host_write_buffer(ptr uint32)

// host_format tells the host to format code using another plugin. This allows
// plugins to delegate formatting to other specialized plugins (e.g., markdown
// plugin formatting code blocks within markdown files).
// Parameters:
//   - filePathPtr, filePathLen: pointer and length of file path string
//   - rangeStart, rangeEnd: byte range to format (0 and fileLen for full file)
//   - overridePtr, overrideLen: pointer and length of override config JSON
//   - fileBytesPtr, fileBytesLen: pointer and length of file content
//
// Returns:
//   - 0: no change needed
//   - 1: file was changed (call host_get_formatted_text)
//   - 2: formatting error (call host_get_error_text)
//
// See: https://dprint.dev/plugins/wasm/#host_format
//
//go:wasmimport dprint host_format
//goland:noinspection GoUnusedFunction,GoSnakeCaseUsage,GoUnusedParameter
func host_format(filePathPtr, filePathLen, rangeStart, rangeEnd, overridePtr, overrideLen, fileBytesPtr, fileBytesLen uint32) uint32

// host_get_formatted_text tells the host to store the formatted text in its
// local byte array and returns the byte length of that text. Call this after
// host_format returns 1 (indicating successful formatting with changes).
// See: https://dprint.dev/plugins/wasm/#host_get_formatted_text
//
//go:wasmimport dprint host_get_formatted_text
//goland:noinspection GoUnusedFunction,GoSnakeCaseUsage
func host_get_formatted_text() uint32

// host_get_error_text tells the host to store the error text in its local
// byte array and returns the byte length of that error message. Call this
// after host_format returns 2 (indicating a formatting error occurred).
// See: https://dprint.dev/plugins/wasm/#host_get_error_text
//
//go:wasmimport dprint host_get_error_text
//goland:noinspection GoUnusedFunction,GoSnakeCaseUsage
func host_get_error_text() uint32

// host_has_cancelled checks if the host has cancelled the formatting request.
// This allows long-running formatting operations to be interrupted gracefully.
// Returns 1 if cancelled, 0 if still active.
// See: https://dprint.dev/plugins/wasm/#host_has_cancelled
//
//go:wasmimport dprint host_has_cancelled
//goland:noinspection GoUnusedFunction,GoSnakeCaseUsage
func host_has_cancelled() uint32
