package main

import (
	"log"
	"os"

	"github.com/mridang/dprint-plugin-go/internal/wasm"
)

// This tool injects a WebAssembly "start" section (section 8) into a
// Wasm module. This is required by the dprint CLI.
// It finds the function index of "_initialize" and sets that as the
// start function.
//
// This is necessary because TinyGo does not add this section, but wasmer-go
// (used in tests) will crash if it's present.
func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Usage: %s <input.wasm> <output.wasm>", os.Args[0])
	}
	inPath, outPath := os.Args[1], os.Args[2]

	data, err := os.ReadFile(inPath)
	if err != nil {
		log.Fatal(err)
	}

	// Use the shared utility to add the start section
	output, err := wasm.AddStartSection(data)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(outPath, output, 0644); err != nil {
		log.Fatal(err)
	}
}
