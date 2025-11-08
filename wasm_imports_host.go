//go:build tinygo

package main

// --- host imports (declared for ABI; not used in no-op) ---
//
//go:wasmimport dprint host_write_buffer
func host_write_buffer(ptr uint32)

//go:wasmimport dprint host_format
func host_format(filePathPtr, filePathLen, rangeStart, rangeEnd, overridePtr, overrideLen, fileBytesPtr, fileBytesLen uint32) uint32

//go:wasmimport dprint host_get_formatted_text
func host_get_formatted_text() uint32

//go:wasmimport dprint host_get_error_text
func host_get_error_text() uint32

//go:wasmimport dprint host_has_cancelled
func host_has_cancelled() uint32
