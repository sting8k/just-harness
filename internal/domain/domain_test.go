package domain

import (
	"strings"
	"testing"
)

func ptr(s string) *string { return &s }

func story(lane, verify string, lv *Verify) *Story {
	return &Story{V: 1, ID: "US-k3f9", Title: "t", Lane: lane, Status: "in_progress", Verify: verify, LastVerify: lv, CreatedAt: "x", UpdatedAt: "x"}
}

func TestValidateEnums(t *testing.T) {
	s := story("normal", "", nil)
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	s.Lane = "huge"
	if err := s.Validate(); err == nil {
		t.Fatal("bad lane accepted")
	}
	d := &Decision{V: 1, ID: "US-k3f9", Title: "t", Status: "accepted", CreatedAt: "x", UpdatedAt: "x"}
	if err := d.Validate(); err == nil {
		t.Fatal("story id accepted as decision id")
	}
	tr := &Trace{V: 2, ID: "T-4hc2", Summary: "s", Outcome: "completed", CreatedAt: "x"}
	if err := tr.Validate(); err == nil {
		t.Fatal("v=2 accepted")
	}
}

func TestGate(t *testing.T) {
	pass := &Verify{Result: "pass", Command: "go test", At: "x"}
	fail := &Verify{Result: "fail", ExitCode: 1, Command: "go test", At: "x"}
	stale := &Verify{Result: "pass", Command: "old", At: "x"}
	cases := []struct {
		name  string
		s     *Story
		waive *string
		ok    bool
	}{
		{"I1 pass matches", story("normal", "go test", pass), nil, true},
		{"I1 no run", story("normal", "go test", nil), nil, false},
		{"I1 failed", story("normal", "go test", fail), nil, false},
		{"I1 command drift", story("normal", "go test", stale), nil, false},
		{"I1 waived", story("normal", "go test", fail), ptr("no env"), true},
		{"I2 normal without verify", story("normal", "", nil), nil, true},
		{"I2 high_risk without verify", story("high_risk", "", nil), nil, false},
		{"I2 high_risk waived", story("high_risk", "", nil), ptr("manual"), true},
	}
	for _, c := range cases {
		gate, in := ApplyStoryChange(c.s, StoryChange{Status: ptr("implemented"), Waive: c.waive}, "now")
		if in != nil || (gate == nil) != c.ok {
			t.Errorf("%s: gate=%v input=%v", c.name, gate, in)
		}
	}
}

func TestTransitions(t *testing.T) {
	// I3: changing verify drops last_verify; implemented story is re-gated.
	s := story("normal", "go test", &Verify{Result: "pass", Command: "go test", At: "x"})
	s.Status = "implemented"
	if gate, _ := ApplyStoryChange(s, StoryChange{Verify: ptr("go test ./...")}, "now"); gate == nil {
		t.Fatal("verify change on implemented story not gated")
	}
	s = story("normal", "go test", &Verify{Result: "pass", Command: "go test", At: "x"})
	ApplyStoryChange(s, StoryChange{Verify: ptr("go vet")}, "now")
	if s.LastVerify != nil {
		t.Fatal("I3: last_verify kept after verify change")
	}
	// Unrelated update on an implemented story that violates I1 is not blocked.
	s = story("normal", "go test", nil)
	s.Status = "implemented"
	if gate, in := ApplyStoryChange(s, StoryChange{Title: ptr("new")}, "now"); gate != nil || in != nil {
		t.Fatalf("title update blocked: %v %v", gate, in)
	}
	// Q17: changed clears last_verify; leaving implemented drops the waiver,
	// so it cannot reopen the gate later.
	s = story("normal", "go test", &Verify{Result: "pass", Command: "go test", At: "x"})
	s.Status, s.Waiver = "implemented", "old"
	ApplyStoryChange(s, StoryChange{Status: ptr("changed")}, "now")
	if s.LastVerify != nil || s.Waiver != "" {
		t.Fatalf("changed kept proof: %+v", s)
	}
	if gate, _ := ApplyStoryChange(s, StoryChange{Status: ptr("implemented")}, "now"); gate == nil {
		t.Fatal("re-close without proof accepted")
	}
	if _, in := ApplyStoryChange(story("normal", "", nil), StoryChange{Waive: ptr("r")}, "now"); in == nil {
		t.Fatal("--waive accepted on non-implemented story")
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Pagination":                  "pagination",
		"Đăng nhập bằng JWT — bước 2": "dang-nhap-bang-jwt-buoc-2",
		"  ---  ":                     "",
		"Crème brûlée!!":              "creme-brulee",
		"e\u0301":                     "e",
		strings.Repeat("ab ", 30):     strings.Repeat("ab-", 13) + "a",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q)=%q want %q", in, got, want)
		}
	}
}
