package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// newRepo creates an empty harness root and points the CLI at it.
func newRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".harness"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HARNESS_REPO_ROOT", root)
	return root
}

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	return code, out.String(), errb.String()
}

// mustRun runs a command expecting exit 0 and returns the created id, if any.
func mustRun(t *testing.T, args ...string) string {
	t.Helper()
	code, out, errs := run(t, args...)
	if code != 0 {
		t.Fatalf("%v: exit %d\nstdout: %s\nstderr: %s", args, code, out, errs)
	}
	if rest, ok := strings.CutPrefix(out, "created "); ok {
		return strings.Fields(rest)[0]
	}
	return ""
}

func readFile(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestVersion(t *testing.T) {
	if code, out, _ := run(t, "version"); code != 0 || out != "dev\n" {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestInputErrorsExit2(t *testing.T) {
	newRepo(t)
	for _, args := range [][]string{
		{"story", "add", "--title", "x"},                  // missing --lane
		{"story", "add", "--title", "x", "--lane", "big"}, // bad enum
		{"story", "update", "--id", "US-zzzz", "--title", "y"},
		{"story", "add", "--title", "x", "--lane", "normal", "--contract", "docs/missing.md"},
		{"decision", "update", "--id", "D-0000", "--status", "superseded", "--by", "D-1111"},
		{"trace", "--summary", "s", "--outcome", "done"},
		{"nope"},
	} {
		if code, _, _ := run(t, args...); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
}

func TestStoryGateAndVerify(t *testing.T) {
	root := newRepo(t)
	id := mustRun(t, "story", "add", "--title", "Đăng nhập", "--lane", "high_risk")
	rec := ".harness/stories/" + id + ".json"
	packet := "docs/stories/" + id + "-dang-nhap.md"
	if !strings.Contains(readFile(t, root, rec), `"packet": "`+packet+`"`) {
		t.Fatal("packet not linked")
	}
	readFile(t, root, packet)

	// I2: high_risk without verify is refused; the record stays byte-identical.
	before := readFile(t, root, rec)
	if code, _, errs := run(t, "story", "update", "--id", id, "--status", "implemented"); code != 1 || !strings.Contains(errs, "I2") {
		t.Fatalf("I2 not enforced: code=%d %s", code, errs)
	}
	if readFile(t, root, rec) != before {
		t.Fatal("refused update modified the record")
	}

	// I1: verify fails → exit code propagates → gate still closed.
	mustRun(t, "story", "update", "--id", id, "--verify", "exit 3")
	if code, _, _ := run(t, "story", "verify", "--id", id); code != 3 {
		t.Fatalf("verify exit code not propagated: %d", code)
	}
	if !strings.Contains(readFile(t, root, rec), `"result": "fail"`) {
		t.Fatal("failed verify not recorded")
	}
	if code, _, _ := run(t, "story", "update", "--id", id, "--status", "implemented"); code != 1 {
		t.Fatal("closed on a failing verify")
	}

	// Pass with the current command opens the gate.
	mustRun(t, "story", "update", "--id", id, "--verify", "exit 0")
	mustRun(t, "story", "verify", "--id", id)
	mustRun(t, "story", "update", "--id", id, "--status", "implemented")

	// I3 on an implemented story: changing verify without --waive is refused.
	if code, _, _ := run(t, "story", "update", "--id", id, "--verify", "true"); code != 1 {
		t.Fatal("verify change on implemented story not gated")
	}
	mustRun(t, "story", "update", "--id", id, "--verify", "true", "--waive", "tooling moved")
	if got := readFile(t, root, rec); !strings.Contains(got, `"waiver": "tooling moved"`) || strings.Contains(got, "last_verify") {
		t.Fatalf("waiver/I3 not applied:\n%s", got)
	}
}

// A timed-out verify must kill the whole process tree: an orphaned child
// would keep running and hold the repo dir (Windows TempDir cleanup fails).
func TestStoryVerifyTimeout(t *testing.T) {
	root := newRepo(t)
	slow := "sleep 30 | sleep 30"
	if runtime.GOOS == "windows" {
		slow = "ping -n 30 127.0.0.1 >NUL & ping -n 30 127.0.0.1 >NUL"
	}
	id := mustRun(t, "story", "add", "--title", "slow", "--lane", "tiny", "--no-packet", "--verify", slow)
	start := time.Now()
	if code, _, _ := run(t, "story", "verify", "--id", id, "--timeout", "300ms"); code != timeoutExitCode {
		t.Fatalf("timeout exit %d", code)
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Fatalf("verify returned after %s; children kept it alive", d)
	}
	if got := readFile(t, root, ".harness/stories/"+id+".json"); !strings.Contains(got, `"result": "fail"`) || !strings.Contains(got, `"exit_code": 124`) {
		t.Fatalf("timeout not recorded as fail/124:\n%s", got)
	}
}

func TestDecisionGuardrailTrace(t *testing.T) {
	root := newRepo(t)
	d1 := mustRun(t, "decision", "add", "--title", "JWT auth")
	d2 := mustRun(t, "decision", "add", "--title", "Session auth", "--status", "accepted")
	mustRun(t, "decision", "update", "--id", d1, "--status", "superseded", "--by", d2)
	if got := readFile(t, root, ".harness/decisions/"+d1+".json"); !strings.Contains(got, `"superseded_by": "`+d2+`"`) {
		t.Fatalf("superseded_by missing:\n%s", got)
	}
	readFile(t, root, "docs/decisions/"+d1+"-jwt-auth.md")

	g := mustRun(t, "guardrail", "add", "--rule", "No raw SQL", "--why", "injection")
	mustRun(t, "guardrail", "supersede", "--id", g)
	tr := mustRun(t, "trace", "--summary", "did it", "--outcome", "partial", "--unverified", "windows")
	if got := readFile(t, root, ".harness/traces/"+tr+".json"); !strings.Contains(got, `"unverified": "windows"`) {
		t.Fatalf("trace:\n%s", got)
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckFlagsHandEdits(t *testing.T) {
	root := newRepo(t)
	id := mustRun(t, "story", "add", "--title", "Pagination", "--lane", "normal", "--verify", "exit 0")
	d := mustRun(t, "decision", "add", "--title", "Keyset", "--story", id)
	if code, out, _ := run(t, "check"); code != 0 {
		t.Fatalf("clean repo flagged:\n%s", out)
	}

	rec := ".harness/stories/" + id + ".json"
	// Hand edit: implemented without a pass (I1), with CRLF line endings.
	orig := readFile(t, root, rec)
	edited := strings.ReplaceAll(strings.Replace(orig, `"planned"`, `"implemented"`, 1), "\n", "\r\n")
	writeFile(t, root, rec, edited)
	// Duplicate id (e.g. merged copy under another name), broken doc path,
	// dangling reference, and invalid JSON (merge conflict markers).
	writeFile(t, root, ".harness/stories/US-zzzz.json", orig)
	os.Remove(filepath.Join(root, "docs", "decisions", d+"-keyset.md"))
	writeFile(t, root, ".harness/traces/T-0000.json",
		`{"v":1,"id":"T-0000","summary":"s","outcome":"completed","story":"US-9999","created_at":"x"}`)
	writeFile(t, root, ".harness/guardrails/G-1111.json", "<<<<<<< HEAD\n{}\n")

	code, out, _ := run(t, "check")
	if code != 1 {
		t.Fatalf("check exit %d, want 1\n%s", code, out)
	}
	for _, want := range []string{
		id + ".json: implemented but no verify run recorded (I1)",
		"US-zzzz.json: id \"" + id + "\" does not match the file name",
		"duplicate id " + id,
		"doc docs/decisions/" + d + "-keyset.md does not exist",
		"T-0000.json: story US-9999 does not exist",
		"G-1111.json: invalid JSON",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("check output missing %q:\n%s", want, out)
		}
	}
	if _, status, _ := run(t, "query", "status"); !strings.Contains(status, "Needs attention:     "+id+" [normal] implemented — no verify run recorded (I1)") {
		t.Errorf("status does not surface the I1 violation:\n%s", status)
	}
}

func TestQueryStatusAndMarkdown(t *testing.T) {
	newRepo(t)
	mustRun(t, "guardrail", "add", "--rule", "Routes must re-check the user", "--why", "deleted users")
	open := mustRun(t, "story", "add", "--title", "Pagination", "--lane", "normal")
	mustRun(t, "story", "update", "--id", open, "--status", "in_progress")
	waived := mustRun(t, "story", "add", "--title", "a | b", "--lane", "high_risk", "--no-packet")
	mustRun(t, "story", "update", "--id", waived, "--status", "implemented", "--waive", "no e2e env")
	mustRun(t, "decision", "add", "--title", "JWT auth strategy")

	_, out, _ := run(t, "query", "status")
	for _, want := range []string{
		"Guardrails (active): G-",
		"Open stories:        " + open + " [normal] in_progress — Pagination   (docs/stories/" + open + "-pagination.md)",
		"Needs attention:     " + waived + ` [high_risk] implemented — WAIVED: "no e2e env"`,
		"Proposed decisions:  D-",
		"Problems (check):    none",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("status missing %q:\n%s", want, out)
		}
	}

	_, md, _ := run(t, "query", "stories", "--status", "implemented", "--md")
	want := "| " + waived + ` | high_risk | implemented | WAIVED: "no e2e env" | a \| b |  |`
	if !strings.Contains(md, want) || strings.Contains(md, open) {
		t.Fatalf("markdown:\n%s\nwant row %s", md, want)
	}
	if code, _, _ := run(t, "query", "stories", "--status", "done"); code != 2 {
		t.Fatal("bad --status accepted")
	}
}
