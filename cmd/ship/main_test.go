package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRunPrintsVersionWhenShortVersionFlagIsUsed(t *testing.T) {
	var output bytes.Buffer
	withStdout(t, &output, func() {
		if err := run(context.Background(), []string{"-v"}); err != nil {
			t.Fatal(err)
		}
	})

	if strings.TrimSpace(output.String()) != "ship dev" {
		t.Fatalf("unexpected version output %q", output.String())
	}
}

func withStdout(t *testing.T, writer io.Writer, run func()) {
	t.Helper()
	original := os.Stdout
	reader, pipeWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = pipeWriter
	defer func() {
		os.Stdout = original
	}()

	run()

	if err := pipeWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(writer, reader); err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
}
