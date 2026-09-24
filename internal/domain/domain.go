// Package domain holds record types, enums and invariants. Pure: no I/O.
package domain

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// SchemaVersion is the "v" field every record carries.
const SchemaVersion = 1

// Kind names a record family; it is also the .harness/<kind>/ directory.
type Kind string

const (
	KindStory     Kind = "stories"
	KindDecision  Kind = "decisions"
	KindGuardrail Kind = "guardrails"
	KindTrace     Kind = "traces"
)

// Kinds lists every record kind in display order.
var Kinds = []Kind{KindStory, KindDecision, KindGuardrail, KindTrace}

// Prefix returns the ID prefix for a kind.
func (k Kind) Prefix() string {
	switch k {
	case KindStory:
		return "US"
	case KindDecision:
		return "D"
	case KindGuardrail:
		return "G"
	case KindTrace:
		return "T"
	}
	return ""
}

// Enums.
var (
	Lanes             = []string{"tiny", "normal", "high_risk"}
	StoryStatuses     = []string{"planned", "in_progress", "implemented", "changed", "retired"}
	DecisionStatuses  = []string{"proposed", "accepted", "superseded", "rejected"}
	GuardrailStatuses = []string{"active", "superseded"}
	TraceOutcomes     = []string{"completed", "partial", "blocked", "failed"}
	VerifyResults     = []string{"pass", "fail"}
)

// CheckEnum returns an error when v is not one of allowed.
func CheckEnum(field, v string, allowed []string) error {
	for _, a := range allowed {
		if v == a {
			return nil
		}
	}
	return fmt.Errorf("%s: invalid value %q (want %s)", field, v, strings.Join(allowed, "|"))
}

// IDAlphabet is lowercase Crockford base32 (no i, l, o, u).
const IDAlphabet = "0123456789abcdefghjkmnpqrstvwxyz"

// IDLen is the number of random characters after the prefix.
const IDLen = 4

var idPattern = regexp.MustCompile(`^(US|D|G|T)-[0-9a-hjkmnp-tv-z]{4}$`)

// CheckID validates an ID against the expected kind.
func CheckID(k Kind, id string) error {
	if !idPattern.MatchString(id) || !strings.HasPrefix(id, k.Prefix()+"-") {
		return fmt.Errorf("invalid %s id %q (want %s-xxxx)", k, id, k.Prefix())
	}
	return nil
}

// Verify is the result of the last `story verify` run.
type Verify struct {
	Result   string `json:"result"`
	ExitCode int    `json:"exit_code"`
	Command  string `json:"command"`
	Commit   string `json:"commit,omitempty"`
	Dirty    bool   `json:"dirty,omitempty"`
	At       string `json:"at"`
}

// Story is a work packet record. Field order = JSON key order (I7).
type Story struct {
	V          int     `json:"v"`
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Lane       string  `json:"lane"`
	Status     string  `json:"status"`
	Packet     string  `json:"packet,omitempty"`
	Contract   string  `json:"contract,omitempty"`
	Verify     string  `json:"verify,omitempty"`
	LastVerify *Verify `json:"last_verify,omitempty"`
	Waiver     string  `json:"waiver,omitempty"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

// Decision records a choice later work must inherit.
type Decision struct {
	V            int    `json:"v"`
	ID           string `json:"id"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	Doc          string `json:"doc,omitempty"`
	Story        string `json:"story,omitempty"`
	SupersededBy string `json:"superseded_by,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// Guardrail is a durable project rule.
type Guardrail struct {
	V         int    `json:"v"`
	ID        string `json:"id"`
	Rule      string `json:"rule"`
	Why       string `json:"why"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Trace is an append-only execution record.
type Trace struct {
	V          int    `json:"v"`
	ID         string `json:"id"`
	Summary    string `json:"summary"`
	Outcome    string `json:"outcome"`
	Story      string `json:"story,omitempty"`
	Evidence   string `json:"evidence,omitempty"`
	Unverified string `json:"unverified,omitempty"`
	Agent      string `json:"agent,omitempty"`
	Commit     string `json:"commit,omitempty"`
	CreatedAt  string `json:"created_at"`
}

func checkCommon(k Kind, v int, id string, required map[string]string) error {
	if v != SchemaVersion {
		return fmt.Errorf("v: unsupported schema version %d", v)
	}
	if err := CheckID(k, id); err != nil {
		return err
	}
	for field, val := range required {
		if strings.TrimSpace(val) == "" {
			return fmt.Errorf("%s: required", field)
		}
	}
	return nil
}

// Validate checks schema and enums of a story (not the gate; see CheckImplemented).
func (s *Story) Validate() error {
	if err := checkCommon(KindStory, s.V, s.ID, map[string]string{"title": s.Title, "created_at": s.CreatedAt, "updated_at": s.UpdatedAt}); err != nil {
		return err
	}
	if err := CheckEnum("lane", s.Lane, Lanes); err != nil {
		return err
	}
	if err := CheckEnum("status", s.Status, StoryStatuses); err != nil {
		return err
	}
	if lv := s.LastVerify; lv != nil {
		if err := CheckEnum("last_verify.result", lv.Result, VerifyResults); err != nil {
			return err
		}
		if lv.Command == "" || lv.At == "" {
			return fmt.Errorf("last_verify: command and at are required")
		}
		if (lv.Result == "pass") != (lv.ExitCode == 0) {
			return fmt.Errorf("last_verify: result %q disagrees with exit_code %d", lv.Result, lv.ExitCode)
		}
	}
	return nil
}

// Validate checks schema and enums of a decision.
func (d *Decision) Validate() error {
	if err := checkCommon(KindDecision, d.V, d.ID, map[string]string{"title": d.Title, "created_at": d.CreatedAt, "updated_at": d.UpdatedAt}); err != nil {
		return err
	}
	if err := CheckEnum("status", d.Status, DecisionStatuses); err != nil {
		return err
	}
	if d.SupersededBy != "" {
		if err := CheckID(KindDecision, d.SupersededBy); err != nil {
			return fmt.Errorf("superseded_by: %w", err)
		}
	}
	if d.Story != "" {
		if err := CheckID(KindStory, d.Story); err != nil {
			return fmt.Errorf("story: %w", err)
		}
	}
	return nil
}

// Validate checks schema and enums of a guardrail.
func (g *Guardrail) Validate() error {
	if err := checkCommon(KindGuardrail, g.V, g.ID, map[string]string{"rule": g.Rule, "why": g.Why, "created_at": g.CreatedAt, "updated_at": g.UpdatedAt}); err != nil {
		return err
	}
	return CheckEnum("status", g.Status, GuardrailStatuses)
}

// Validate checks schema and enums of a trace.
func (t *Trace) Validate() error {
	if err := checkCommon(KindTrace, t.V, t.ID, map[string]string{"summary": t.Summary, "created_at": t.CreatedAt}); err != nil {
		return err
	}
	if err := CheckEnum("outcome", t.Outcome, TraceOutcomes); err != nil {
		return err
	}
	if t.Story != "" {
		if err := CheckID(KindStory, t.Story); err != nil {
			return fmt.Errorf("story: %w", err)
		}
	}
	return nil
}

// PassMatches reports whether the last verify passed with the current command.
func (s *Story) PassMatches() bool {
	return s.Verify != "" && s.LastVerify != nil && s.LastVerify.Result == "pass" && s.LastVerify.Command == s.Verify
}

// GateIssue is the gate rule (I1/I2), shared by the CLI and `check`: an
// implemented story needs a waiver, a matching pass, or — outside high_risk —
// no verify command at all. Returns "" when satisfied, else a short reason.
func (s *Story) GateIssue() string {
	switch {
	case s.Waiver != "", s.PassMatches():
		return ""
	case s.Verify == "" && s.Lane == "high_risk":
		return "high_risk story has no verify command (I2)"
	case s.Verify == "":
		return ""
	case s.LastVerify == nil:
		return "no verify run recorded (I1)"
	case s.LastVerify.Command != s.Verify:
		return "verify changed since last run (I1)"
	default:
		return fmt.Sprintf("last verify failed, exit %d (I1)", s.LastVerify.ExitCode)
	}
}

// CheckImplemented wraps GateIssue with how to fix it.
func (s *Story) CheckImplemented() error {
	issue := s.GateIssue()
	if issue == "" {
		return nil
	}
	fix := fmt.Sprintf("run `story verify --id %s` until it passes", s.ID)
	if s.Verify == "" {
		fix = "set --verify and pass it"
	}
	return fmt.Errorf("%s: %s; %s, or pass --waive \"reason\"", s.ID, issue, fix)
}

// StoryChange is the requested mutation of `story update`; nil = unchanged.
type StoryChange struct {
	Status, Title, Lane, Verify, Contract, Waive *string
}

// ApplyStoryChange applies c to s, enforcing I1–I3 and the transition rules:
// sang changed xóa last_verify; leaving implemented drops the waiver; the gate
// re-runs only when status enters implemented, or an implemented story
// changes verify or lane. Returns an input error for bad enums.
func ApplyStoryChange(s *Story, c StoryChange, now string) (gateErr, inputErr error) {
	if c.Lane != nil {
		if err := CheckEnum("lane", *c.Lane, Lanes); err != nil {
			return nil, err
		}
	}
	if c.Status != nil {
		if err := CheckEnum("status", *c.Status, StoryStatuses); err != nil {
			return nil, err
		}
	}
	if c.Title != nil && strings.TrimSpace(*c.Title) == "" {
		return nil, fmt.Errorf("title: required")
	}
	if c.Waive != nil && strings.TrimSpace(*c.Waive) == "" {
		return nil, fmt.Errorf("waive: reason required")
	}

	oldStatus, oldVerify, oldLane := s.Status, s.Verify, s.Lane
	if c.Title != nil {
		s.Title = *c.Title
	}
	if c.Lane != nil {
		s.Lane = *c.Lane
	}
	if c.Contract != nil {
		s.Contract = *c.Contract
	}
	if c.Verify != nil && *c.Verify != s.Verify {
		s.Verify = *c.Verify
		s.LastVerify = nil // I3
	}
	if c.Status != nil {
		s.Status = *c.Status
	}
	if s.Status != "implemented" {
		if c.Waive != nil {
			return nil, fmt.Errorf("--waive only applies when the story ends up implemented")
		}
		s.Waiver = ""
	}
	if s.Status == "changed" && oldStatus != "changed" {
		s.LastVerify = nil
	}
	if s.Status == "implemented" {
		regate := oldStatus != "implemented" || s.Verify != oldVerify || s.Lane != oldLane
		if regate {
			s.Waiver = "" // an old waiver was given for other conditions; never reuse it
		}
		if c.Waive != nil {
			s.Waiver = *c.Waive
		}
		if regate {
			if err := s.CheckImplemented(); err != nil {
				return err, nil
			}
		}
	}
	s.UpdatedAt = now
	return nil, nil
}

var foldTable = func() map[rune]rune {
	groups := map[rune]string{
		'a': "àáảãạăằắẳẵặâầấẩẫậäåāą",
		'c': "çćč",
		'd': "đď",
		'e': "èéẻẽẹêềếểễệëēęě",
		'g': "ğ",
		'i': "ìíỉĩịîïī",
		'l': "ł",
		'n': "ñńň",
		'o': "òóỏõọôồốổỗộơờớởỡợöøōő",
		'r': "ř",
		's': "śšş",
		't': "ť",
		'u': "ùúủũụưừứửữựûüūůű",
		'y': "ỳýỷỹỵÿ",
		'z': "źżž",
	}
	m := map[rune]rune{}
	for base, variants := range groups {
		for _, r := range variants {
			m[r] = base
		}
	}
	return m
}()

// SlugMaxLen caps slug length (Q19).
const SlugMaxLen = 40

// Slug folds diacritics (incl. Vietnamese, đ→d), lowercases, maps anything
// outside [a-z0-9] to '-', collapses and trims dashes, cuts to 40 (Q19).
func Slug(title string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(title) {
		if unicode.Is(unicode.Mn, r) { // decomposed combining marks
			continue
		}
		if f, ok := foldTable[r]; ok {
			r = f
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	s := strings.TrimRight(b.String(), "-")
	if len(s) > SlugMaxLen {
		s = strings.TrimRight(s[:SlugMaxLen], "-")
	}
	return s
}
