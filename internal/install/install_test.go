package install

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// goldenManifest is the review gate: any payload change must edit this list.
var goldenManifest = []string{
	".gitattributes",
	".gitignore",
	".harness/README.md",
	"AGENTS.md",
	"docs/HARNESS.md",
	"docs/templates/decision.md",
	"docs/templates/story.md",
}

func TestGoldenManifest(t *testing.T) {
	files, err := PayloadFiles()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(files, "\n") != strings.Join(goldenManifest, "\n") {
		t.Fatalf("payload:\n%s\nwant:\n%s", strings.Join(files, "\n"), strings.Join(goldenManifest, "\n"))
	}
	agents, _ := readPayload("AGENTS.md")
	if !bytes.Contains(agents, []byte(shimBegin)) || !bytes.Contains(agents, []byte(shimEnd)) {
		t.Fatal("payload AGENTS.md lacks HARNESS markers")
	}
}

func fakeSelf(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "just-harness-cli")
	if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func install(t *testing.T, opts Options) (string, error) {
	t.Helper()
	if opts.SelfBinary == "" {
		opts.SelfBinary = fakeSelf(t, "binary-v1")
	}
	plan, err := NewPlan(opts)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	err = plan.Execute(&out)
	return out.String(), err
}

// snapshot maps every path under root to its content ("<dir>" for dirs).
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	m := map[string]string{}
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			t.Fatal(err)
		}
		rel, _ := filepath.Rel(root, p)
		if d.IsDir() {
			m[filepath.ToSlash(rel)] = "<dir>"
		} else if d.Type()&fs.ModeSymlink != 0 {
			m[filepath.ToSlash(rel)] = "<symlink>"
		} else {
			b, _ := os.ReadFile(p)
			m[filepath.ToSlash(rel)] = string(b)
		}
		return nil
	})
	return m
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestFreshInstall(t *testing.T) {
	target := filepath.Join(t.TempDir(), "repo") // absent: created by the plan
	if _, err := install(t, Options{Target: target, Claude: true}); err != nil {
		t.Fatal(err)
	}
	for _, rel := range goldenManifest {
		want, _ := readPayload(rel)
		if read(t, target, rel) != string(want) {
			t.Errorf("%s differs from payload", rel)
		}
	}
	if read(t, target, BinaryRel()) != "binary-v1" || read(t, target, "CLAUDE.md") != "@AGENTS.md\n" {
		t.Fatal("binary or CLAUDE.md missing")
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(filepath.Join(target, BinaryRel()))
		if info.Mode().Perm() != execBinaryMode {
			t.Fatalf("binary mode %v", info.Mode())
		}
	}
}

func TestRefuseWritesNothing(t *testing.T) {
	target := t.TempDir()
	write(t, target, "docs/README.md", "user docs")
	before := snapshot(t, target)
	_, err := install(t, Options{Target: target})
	var refused *RefusedError
	if !errors.As(err, &refused) || refused.Conflicts[0] != "docs" {
		t.Fatalf("expected refusal, got %v", err)
	}
	if _, err := install(t, Options{Target: target, Merge: true, DryRun: true}); err != nil {
		t.Fatal(err)
	}
	if after := snapshot(t, target); len(after) != len(before) {
		t.Fatalf("refused/dry-run plan wrote files: %v", after)
	}
}

func TestMergeUpgrade(t *testing.T) {
	target := t.TempDir()
	if _, err := install(t, Options{Target: target}); err != nil {
		t.Fatal(err)
	}
	// User customizes files and owns text around the shim; records exist.
	write(t, target, "AGENTS.md", "# Mine\nkeep me\n<!-- HARNESS:BEGIN -->\nold block\n<!-- HARNESS:END -->\ntail\n")
	write(t, target, "docs/HARNESS.md", "customized")
	write(t, target, ".gitignore", "node_modules/")
	write(t, target, ".harness/stories/US-k3f9.json", "{record}")

	out, err := install(t, Options{Target: target, Merge: true, SelfBinary: fakeSelf(t, "binary-v2")})
	if err != nil {
		t.Fatal(err)
	}
	agents := read(t, target, "AGENTS.md")
	payload, _ := readPayload("AGENTS.md")
	block := string(payload[bytes.Index(payload, []byte(shimBegin)):])
	block = block[:strings.Index(block, shimEnd)+len(shimEnd)]
	if !strings.HasPrefix(agents, "# Mine\nkeep me\n"+block) || !strings.HasSuffix(agents, "\ntail\n") || strings.Contains(agents, "old block") {
		t.Fatalf("shim not refreshed in place:\n%s", agents)
	}
	if read(t, target, "docs/HARNESS.md") != "customized" || read(t, target, ".harness/stories/US-k3f9.json") != "{record}" {
		t.Fatal("merge overwrote an existing file or record")
	}
	if read(t, target, ".gitignore") != "node_modules/\n# just-harness\n.harness/bin/\n.harness-backup/\n" {
		t.Fatalf(".gitignore: %q", read(t, target, ".gitignore"))
	}
	if read(t, target, BinaryRel()) != "binary-v2" {
		t.Fatal("binary not upgraded")
	}
	if !strings.Contains(out, "refresh  AGENTS.md (backup: .harness-backup/") {
		t.Fatalf("no AGENTS.md backup logged:\n%s", out)
	}

	// Second merge is a no-op.
	before := snapshot(t, target)
	if _, err := install(t, Options{Target: target, Merge: true, SelfBinary: fakeSelf(t, "binary-v2")}); err != nil {
		t.Fatal(err)
	}
	if after := snapshot(t, target); len(after) != len(before) || after["AGENTS.md"] != before["AGENTS.md"] {
		t.Fatal("repeated merge changed the tree")
	}
}

func TestOverrideBacksUpAndKeepsRecords(t *testing.T) {
	target := t.TempDir()
	write(t, target, "AGENTS.md", "user agents")
	write(t, target, "docs/HARNESS.md", "old policy")
	write(t, target, ".harness/README.md", "old readme")
	write(t, target, ".harness/decisions/D-8m2q.json", "{record}")

	if _, err := install(t, Options{Target: target, Override: true, Force: true}); err != nil {
		t.Fatal(err)
	}
	payloadAgents, _ := readPayload("AGENTS.md")
	if read(t, target, "AGENTS.md") != string(payloadAgents) {
		t.Fatal("AGENTS.md not replaced")
	}
	if read(t, target, ".harness/decisions/D-8m2q.json") != "{record}" {
		t.Fatal("override touched a record")
	}
	backups, _ := filepath.Glob(filepath.Join(target, ".harness-backup", "*"))
	if len(backups) != 1 {
		t.Fatalf("backups: %v", backups)
	}
	b := backups[0]
	for rel, want := range map[string]string{"AGENTS.md": "user agents", "docs/HARNESS.md": "old policy", ".harness/README.md": "old readme"} {
		if got, _ := os.ReadFile(filepath.Join(b, filepath.FromSlash(rel))); string(got) != want {
			t.Errorf("backup %s = %q", rel, got)
		}
	}
	if _, err := os.Stat(filepath.Join(b, ".harness", "decisions")); err == nil {
		t.Fatal("records were copied into the backup")
	}
}

func TestSymlinkRefused(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	target, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(target, "docs")); err != nil {
		t.Fatal(err)
	}
	if _, err := install(t, Options{Target: target, Merge: true}); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink refusal, got %v", err)
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Fatal("wrote through symlink")
	}
	if _, err := os.Lstat(filepath.Join(target, "AGENTS.md")); err == nil {
		t.Fatal("refused plan wrote AGENTS.md")
	}
}

func TestRefreshShimAppendsWithoutMarkers(t *testing.T) {
	payload := []byte("# P\n" + shimBegin + "\nnew\n" + shimEnd + "\n")
	got := string(refreshShim([]byte("user text"), payload))
	if got != "user text\n\n"+shimBegin+"\nnew\n"+shimEnd+"\n" {
		t.Fatalf("%q", got)
	}
}
