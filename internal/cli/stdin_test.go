package cli

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/magichourhq/magic-hour-cli/internal/catalog"
)

type failReader struct{}

func (failReader) Read([]byte) (int, error) { panic("stdin read without '-' input") }

func TestResolveStdinOnlyWhenExplicit(t *testing.T) {
	op := catalog.Operation{Fields: []catalog.Field{{Flag: "image", Type: "string", FileKind: "image"}}}
	values := map[string][]string{"image": {"local.png"}}
	if err := resolveStdin(context.Background(), failReader{}, op, values); err != nil {
		t.Fatal(err)
	}
	values["image"] = []string{"-"}
	input := `{"id":"job-1","type":"image","status":"complete","outputs":[{"url":"https://example.com/image.png","path":"/tmp/image.png","media_type":"image"}]}`
	if err := resolveStdin(context.Background(), strings.NewReader(input), op, values); err != nil {
		t.Fatal(err)
	}
	if values["image"][0] != "/tmp/image.png" {
		t.Fatalf("expected downloaded path, got %q", values["image"][0])
	}
}

func TestResolveStdinRejectsUnsafeResults(t *testing.T) {
	op := catalog.Operation{Fields: []catalog.Field{{Flag: "image", Type: "string", FileKind: "image"}}}
	for _, tc := range []struct {
		name, input, want string
	}{
		{"unfinished", `{"status":"queued","outputs":[{"url":"https://example.com/a.png"}]}`, "not complete"},
		{"failed upstream", `{"type":"image","status":"complete","error":"download failed","outputs":[{"url":"https://example.com/a.png"}]}`, "upstream command failed"},
		{"wrong media", `{"type":"video","status":"complete","outputs":[{"url":"https://example.com/a.mp4"}]}`, "expects image"},
		{"ambiguous", `{"type":"image","status":"complete","outputs":[{"url":"https://example.com/a.png"},{"url":"https://example.com/b.png"}]}`, "expects one output"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := map[string][]string{"image": {"-"}}
			err := resolveStdin(context.Background(), strings.NewReader(tc.input), op, values)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want error containing %q", err, tc.want)
			}
		})
	}
}

func TestResolveStdinRespectsDeadline(t *testing.T) {
	op := catalog.Operation{Fields: []catalog.Field{{Flag: "image", Type: "string", FileKind: "image"}}}
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := resolveStdin(ctx, reader, op, map[string][]string{"image": {"-"}})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v, want deadline exceeded", err)
	}
}
