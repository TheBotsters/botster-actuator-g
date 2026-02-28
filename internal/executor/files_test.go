package executor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheBotsters/botster-actuator-g/internal/protocol"
)

func TestRead_BasicFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("line1\nline2\nline3"), 0644)

	fe := NewFileExecutor(dir)
	res := fe.Read(protocol.ReadPayload{Path: "hello.txt"})
	if res.Error != "" {
		t.Fatalf("unexpected error: %s", res.Error)
	}
	if res.Content != "line1\nline2\nline3" {
		t.Fatalf("got %q", res.Content)
	}
	if res.LinesRead != 3 {
		t.Fatalf("expected 3 lines, got %d", res.LinesRead)
	}
}

func TestRead_OffsetLimit(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("a\nb\nc\nd\ne"), 0644)

	fe := NewFileExecutor(dir)
	offset := 2
	limit := 2
	res := fe.Read(protocol.ReadPayload{Path: "f.txt", Offset: &offset, Limit: &limit})
	if res.Error != "" {
		t.Fatal(res.Error)
	}
	if res.Content != "b\nc" {
		t.Fatalf("got %q", res.Content)
	}
	if res.LinesRead != 2 {
		t.Fatalf("lines: %d", res.LinesRead)
	}
	if !res.Truncated {
		t.Fatal("expected truncated=true")
	}
}

func TestRead_NotFound(t *testing.T) {
	fe := NewFileExecutor(t.TempDir())
	res := fe.Read(protocol.ReadPayload{Path: "nope.txt"})
	if res.Error == "" {
		t.Fatal("expected error")
	}
}

func TestWrite_Basic(t *testing.T) {
	dir := t.TempDir()
	fe := NewFileExecutor(dir)
	res := fe.Write(protocol.WritePayload{Path: "out.txt", Content: "hello"})
	if res.Error != "" {
		t.Fatal(res.Error)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "out.txt"))
	if string(data) != "hello" {
		t.Fatalf("got %q", string(data))
	}
}

func TestWrite_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	fe := NewFileExecutor(dir)
	res := fe.Write(protocol.WritePayload{Path: "a/b/c.txt", Content: "deep"})
	if res.Error != "" {
		t.Fatal(res.Error)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "a", "b", "c.txt"))
	if string(data) != "deep" {
		t.Fatalf("got %q", string(data))
	}
}

func TestEdit_Basic(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "e.txt"), []byte("foo bar baz"), 0644)

	fe := NewFileExecutor(dir)
	res := fe.Edit(protocol.EditPayload{Path: "e.txt", OldText: "bar", NewText: "QUX"})
	if res.Error != "" {
		t.Fatal(res.Error)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "e.txt"))
	if string(data) != "foo QUX baz" {
		t.Fatalf("got %q", string(data))
	}
}

func TestEdit_OldTextNotFound(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "e.txt"), []byte("abc"), 0644)

	fe := NewFileExecutor(dir)
	res := fe.Edit(protocol.EditPayload{Path: "e.txt", OldText: "xyz", NewText: "new"})
	if res.Error == "" {
		t.Fatal("expected error")
	}
}

func TestPathTraversal(t *testing.T) {
	dir := t.TempDir()
	fe := NewFileExecutor(dir)

	cases := []string{"../etc/passwd", "../../foo", "a/../../b"}
	for _, p := range cases {
		t.Run(p, func(t *testing.T) {
			res := fe.Read(protocol.ReadPayload{Path: p})
			if res.Error == "" || !strings.Contains(res.Error, "outside root") {
				t.Fatalf("expected path rejection for %q, got error=%q", p, res.Error)
			}
			wres := fe.Write(protocol.WritePayload{Path: p, Content: "x"})
			if wres.Error == "" || !strings.Contains(wres.Error, "outside root") {
				t.Fatalf("expected path rejection for write %q", p)
			}
		})
	}
}
