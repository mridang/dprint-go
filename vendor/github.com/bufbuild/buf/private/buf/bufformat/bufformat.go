package bufformat

import (
	"bytes"
	"io"

	"github.com/bufbuild/protocompile/ast"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
)

// Format is the main entry point.
// It takes raw bytes, formats them, and returns raw bytes.
func Format(filename string, input []byte) ([]byte, error) {
	// 1. Create a Reader for the input
	reader := bytes.NewReader(input)

	// 2. Parse the proto content into an AST
	// We use a nil reporter to silence standard warnings, we only care about errors.
	handler := reporter.NewHandler(nil)

	// Parse acts synchronously.
	fileNode, err := parser.Parse(filename, reader, handler)
	if err != nil {
		return nil, err
	}

	// 3. Create a buffer to write the result to
	var output bytes.Buffer

	// 4. Run the formatting logic
	if err := FormatFileNode(&output, fileNode); err != nil {
		return nil, err
	}

	return output.Bytes(), nil
}

// FormatFileNode is the bridge to the internal formatter struct.
func FormatFileNode(dest io.Writer, fileNode *ast.FileNode) error {
	// Optional: Validate the AST before formatting to ensure integrity
	if _, err := parser.ResultFromAST(fileNode, true, reporter.NewHandler(nil)); err != nil {
		return err
	}

	// Initialize the formatter from formatter.go
	formatter := newFormatter(dest, fileNode)
	return formatter.Run()
}
