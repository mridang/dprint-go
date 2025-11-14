package wasm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
)

// section represents a WebAssembly section.
type section struct {
	id   byte
	body []byte
}

// AddStartSection injects a "start" section (section 8) into a Wasm module.
// It finds the function index of "_initialize" and sets that as the
// start function, which is required by the dprint CLI.
//
// This function correctly parses the Wasm module to insert the start
// section in its canonical order (after section 7, Export).
func AddStartSection(data []byte) ([]byte, error) {
	if err := ensureMagic(data); err != nil {
		return nil, err
	}

	secs, err := parseSections(data[8:])
	if err != nil {
		return nil, err
	}

	fnIndex, err := findInitializeIndex(secs)
	if err != nil {
		return nil, err
	}

	secs = dropStartSection(secs)
	start := buildStartSection(fnIndex)
	secs = insertStartSection(secs, start)
	out := rebuildModule(data[:8], secs)

	fmt.Printf("Added start calling _initialize (func index %d)\n", fnIndex)
	return out, nil
}

// ensureMagic checks for the Wasm magic bytes and version.
func ensureMagic(b []byte) error {
	if len(b) < 8 {
		return errors.New("file too small")
	}
	if b[0] != 0x00 || b[1] != 0x61 || b[2] != 0x73 || b[3] != 0x6d {
		return errors.New("bad wasm magic")
	}
	if binary.LittleEndian.Uint32(b[4:8]) != 1 {
		return errors.New("unsupported wasm version")
	}
	return nil
}

// parseSections parses the Wasm module into its constituent sections.
func parseSections(b []byte) ([]section, error) {
	var secs []section
	off := 0
	for off < len(b) {
		if off >= len(b) {
			return nil, errors.New("truncated section id")
		}
		id := b[off]
		off++

		size, n := readU32(b[off:])
		if n == 0 {
			return nil, errors.New("invalid section size")
		}
		off += n

		if off+int(size) > len(b) {
			return nil, errors.New("section exceeds file")
		}

		body := make([]byte, int(size))
		copy(body, b[off:off+int(size)])
		off += int(size)

		secs = append(secs, section{id: id, body: body})
	}
	return secs, nil
}

// findInitializeIndex finds the function index of the "_initialize" export.
func findInitializeIndex(secs []section) (uint32, error) {
	for _, s := range secs {
		if s.id != 7 { // Export section
			continue
		}
		idx, err := scanExportsForInitialize(s.body)
		if err != nil {
			return 0, err
		}
		if idx != nil {
			return *idx, nil
		}
	}
	return 0, errors.New("export _initialize not found")
}

// scanExportsForInitialize scans the body of an export section.
func scanExportsForInitialize(b []byte) (*uint32, error) {
	off := 0

	count, n := readU32(b[off:])
	if n == 0 {
		return nil, errors.New("bad export count")
	}
	off += n

	for range count {
		name, nn := readName(b[off:])
		if nn == 0 {
			return nil, errors.New("bad export name")
		}
		off += nn

		if off >= len(b) {
			return nil, errors.New("truncated export kind")
		}
		kind := b[off]
		off++

		idx, ni := readU32(b[off:])
		if ni == 0 {
			return nil, errors.New("bad export index")
		}
		off += ni

		if kind == 0x00 && name == "_initialize" { // Kind: Function
			return &idx, nil
		}
	}
	return nil, nil // Not found
}

// dropStartSection removes any existing start section (ID 8).
func dropStartSection(secs []section) []section {
	out := make([]section, 0, len(secs))
	for _, s := range secs {
		if s.id == 8 {
			continue
		}
		out = append(out, s)
	}
	return out
}

// buildStartSection creates a new start section (ID 8).
func buildStartSection(fnIndex uint32) section {
	payload := writeU32(fnIndex)
	return section{id: 8, body: payload}
}

// insertStartSection inserts the start section in its canonical order.
func insertStartSection(secs []section, start section) []section {
	i := 0
	// Find the correct insertion point...
	for i < len(secs) && secs[i].id <= 8 {
		i++
	}
	// Slices.Insert handles allocation and copying
	return slices.Insert(secs, i, start)
}

// rebuildModule reconstructs the Wasm module from its header and sections.
func rebuildModule(header []byte, secs []section) []byte {
	// Use a bytes.Buffer for efficient appends
	var out bytes.Buffer
	out.Write(header)
	for _, s := range secs {
		out.WriteByte(s.id)
		sz := writeU32(uint32(len(s.body)))
		out.Write(sz)
		out.Write(s.body)
	}
	return out.Bytes()
}

// readU32 reads a LEB128-encoded unsigned 32-bit integer.
func readU32(b []byte) (uint32, int) {
	var x uint32
	var s uint
	for i := 0; i < len(b) && i < 5; i++ {
		c := b[i]
		x |= uint32(c&0x7f) << s
		if c&0x80 == 0 {
			return x, i + 1
		}
		s += 7
	}
	return 0, 0
}

// writeU32 writes x as a LEB128-encoded unsigned 32-bit integer.
func writeU32(x uint32) []byte {
	var out []byte
	for {
		c := byte(x & 0x7f)
		x >>= 7
		if x != 0 {
			c |= 0x80
		}
		out = append(out, c)
		if x == 0 {
			break
		}
	}
	return out
}

// readName reads a LEB128-encoded string.
func readName(b []byte) (string, int) {
	l, n := readU32(b)
	if n == 0 {
		return "", 0
	}
	if int(l)+n > len(b) {
		return "", 0
	}
	return string(b[n : n+int(l)]), n + int(l)
}

// StripStartSection removes the start section (id 8) if present.
// This is needed for wasmer-go, which doesn't support it.
func StripStartSection(b []byte) []byte {
	if len(b) < 8 {
		return b
	}
	header := b[:8]
	rest := b[8:]

	var out []byte
	out = append(out, header...)

	for off := 0; off < len(rest); {
		id := rest[off]
		off++
		size, n := readU32(rest[off:])
		if n == 0 || off+n+int(size) > len(rest) {
			return b
		}
		off += n
		bodyStart := off
		bodyEnd := off + int(size)

		if id != 8 { // If it's not the start section, keep it
			out = append(out, id)
			out = append(out, writeU32(size)...)
			out = append(out, rest[bodyStart:bodyEnd]...)
		}
		off = bodyEnd
	}
	return out
}
