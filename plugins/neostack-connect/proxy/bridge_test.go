package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestWriteStdoutLineCompactsPrettyJSON(t *testing.T) {
	oldStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writePipe
	t.Cleanup(func() {
		os.Stdout = oldStdout
		_ = readPipe.Close()
		_ = writePipe.Close()
	})

	bridge := newHTTPBridge("http://127.0.0.1:1/mcp")
	if err := bridge.writeStdoutLine([]byte("{\n  \"jsonrpc\": \"2.0\",\n  \"id\": 1\n}")); err != nil {
		t.Fatal(err)
	}

	if err := writePipe.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatal(err)
	}

	want := "{\"jsonrpc\":\"2.0\",\"id\":1}\n"
	if string(output) != want {
		t.Fatalf("output = %q, want %q", output, want)
	}
	if bytes.Count(output, []byte("\n")) != 1 || strings.Contains(string(output[:len(output)-1]), "\n") {
		t.Fatalf("output is not newline-delimited JSON: %q", output)
	}
}
