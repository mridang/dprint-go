// Command addstart injects a WebAssembly start section that calls the
// exported function named "_initialize" in a TinyGo-produced module.
//
// It reads an input module, validates the header, parses sections, and
// finds the export named "_initialize" to obtain its function index. It
// removes any existing start section, creates a new start section with
// that index, reinserts sections preserving canonical order, and writes
// the updated module to the output path.
//
// This is intended for a dprint plugin compiled to WebAssembly where an
// explicit start section is desired. TinyGo often exports an initializer
// without emitting a start section. This tool ensures the module starts
// by invoking "_initialize" when instantiated.
//
// Usage:
//
//	addstart <input.wasm> <output.wasm>
//
// Exit status is non-zero on error. The implementation avoids external
// dependencies and performs minimal parsing of the binary format.
package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

type section struct {
	id   byte
	body []byte
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: addstart <in.wasm> <out.wasm>")
		os.Exit(2)
	}

	inPath := os.Args[1]
	outPath := os.Args[2]

	inBytes, err := os.ReadFile(inPath)
	if err != nil {
		fail(err)
	}

	if err := ensureMagic(inBytes); err != nil {
		fail(err)
	}

	secs, err := parseSections(inBytes[8:])
	if err != nil {
		fail(err)
	}

	fnIndex, err := findInitializeIndex(secs)
	if err != nil {
		fail(err)
	}

	secs = dropStartSection(secs)
	start := buildStartSection(fnIndex)
	secs = insertStartSection(secs, start)
	out := rebuildModule(inBytes[:8], secs)

	if err := atomicWrite(outPath, out); err != nil {
		fail(err)
	}

	fmt.Printf("Added start calling _initialize (func index %d)\n", fnIndex)
}

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

func findInitializeIndex(secs []section) (uint32, error) {
	for _, s := range secs {
		if s.id != 7 {
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

func scanExportsForInitialize(b []byte) (*uint32, error) {
	off := 0

	count, n := readU32(b[off:])
	if n == 0 {
		return nil, errors.New("bad export count")
	}
	off += n

	for i := uint32(0); i < count; i++ {
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

		if kind == 0x00 && name == "_initialize" {
			return &idx, nil
		}
	}
	return nil, nil
}

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

func buildStartSection(fnIndex uint32) section {
	payload := writeU32(fnIndex)
	return section{id: 8, body: payload}
}

func insertStartSection(secs []section, start section) []section {
	i := 0
	for i < len(secs) && secs[i].id <= 8 {
		i++
	}
	out := make([]section, 0, len(secs)+1)
	out = append(out, secs[:i]...)
	out = append(out, start)
	out = append(out, secs[i:]...)
	return out
}

func rebuildModule(header []byte, secs []section) []byte {
	var out []byte
	out = append(out, header...)
	for _, s := range secs {
		out = append(out, s.id)
		sz := writeU32(uint32(len(s.body)))
		out = append(out, sz...)
		out = append(out, s.body...)
	}
	return out
}

func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

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

func fail(err error) {
	if err == nil {
		return
	}
	io.WriteString(os.Stderr, err.Error()+"\n")
	os.Exit(1)
}
