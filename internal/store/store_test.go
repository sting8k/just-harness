package store

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sting8k/just-harness/internal/domain"
)

func newStore(t *testing.T) Store {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, Dir), 0o755); err != nil {
		t.Fatal(err)
	}
	return Store{Root: root}
}

func TestRoundTripDeterministic(t *testing.T) {
	s := newStore(t)
	in := domain.Story{V: 1, ID: "US-k3f9", Title: "Pagination", Lane: "normal", Status: "implemented",
		Verify: "npm test && echo <ok>", LastVerify: &domain.Verify{Result: "pass", Command: "npm test && echo <ok>", At: "t"},
		CreatedAt: "t", UpdatedAt: "t"}
	if err := s.Save(domain.KindStory, in.ID, &in); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(s.Path(domain.KindStory, in.ID))
	var out domain.Story
	if err := s.Load(domain.KindStory, in.ID, &out); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(domain.KindStory, out.ID, &out); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(s.Path(domain.KindStory, in.ID))
	if !bytes.Equal(first, second) {
		t.Fatalf("not deterministic:\n%s\n---\n%s", first, second)
	}
	text := string(first)
	for _, want := range []string{"{\n  \"v\": 1,\n  \"id\": \"US-k3f9\"", "\"exit_code\": 0", "&& echo <ok>"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in\n%s", want, text)
		}
	}
	if strings.Contains(text, "packet") || strings.Contains(text, "\r") || !strings.HasSuffix(text, "}\n") {
		t.Errorf("empty field kept, CRLF or no trailing newline:\n%s", text)
	}
	// Reader accepts CRLF (Q18).
	crlf := bytes.ReplaceAll(first, []byte("\n"), []byte("\r\n"))
	if err := Decode(crlf, &out); err != nil {
		t.Fatalf("CRLF rejected: %v", err)
	}
}

func TestNewIDAndNoOverwrite(t *testing.T) {
	s := newStore(t)
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		id, err := s.NewID(domain.KindDecision)
		if err != nil {
			t.Fatal(err)
		}
		if err := domain.CheckID(domain.KindDecision, id); err != nil {
			t.Fatal(err)
		}
		if seen[id] {
			t.Fatalf("NewID returned an existing id %s", id)
		}
		seen[id] = true
		if err := s.Create(domain.KindDecision, id, map[string]string{"id": id}); err != nil {
			t.Fatal(err)
		}
	}
	for id := range seen {
		if err := s.Create(domain.KindDecision, id, map[string]string{}); err == nil {
			t.Fatal("Create overwrote an existing record")
		}
		break
	}
	rel := ProsePath("stories", "US-k3f9", "Đăng nhập")
	if rel != "docs/stories/US-k3f9-dang-nhap.md" {
		t.Fatalf("ProsePath=%s", rel)
	}
	if err := s.CreateProse(rel, "story.md", "US-k3f9", "Đăng nhập"); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateProse(rel, "story.md", "US-k3f9", "x"); err == nil {
		t.Fatal("prose overwritten")
	}
	body, _ := os.ReadFile(filepath.Join(s.Root, rel))
	if !strings.HasPrefix(string(body), "# US-k3f9 Đăng nhập") {
		t.Fatalf("template not rendered: %q", body)
	}
}

func TestFindRoot(t *testing.T) {
	s := newStore(t)
	deep := filepath.Join(s.Root, "a", "b")
	os.MkdirAll(deep, 0o755)
	t.Setenv("HARNESS_REPO_ROOT", "")
	got, err := FindRoot(deep)
	want, _ := filepath.EvalSymlinks(s.Root)
	gotReal, _ := filepath.EvalSymlinks(got)
	if err != nil || gotReal != want {
		t.Fatalf("FindRoot=%s err=%v want %s", got, err, want)
	}
	if _, err := FindRoot(t.TempDir()); err != ErrNoRoot {
		t.Fatalf("expected ErrNoRoot, got %v", err)
	}
	t.Setenv("HARNESS_REPO_ROOT", deep)
	if got, _ := FindRoot("."); got != deep {
		t.Fatalf("env override ignored: %s", got)
	}
}
