package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/sting8k/just-harness/internal/domain"
	"github.com/sting8k/just-harness/internal/store"
)

// snapshot is every valid record plus the problems found while loading and
// cross-checking them. It backs `check`, `query status` and the list views.
type snapshot struct {
	stories    []domain.Story
	decisions  []domain.Decision
	guardrails []domain.Guardrail
	traces     []domain.Trace
	problems   []string
}

func (s *snapshot) problem(file, format string, a ...any) {
	s.problems = append(s.problems, file+": "+fmt.Sprintf(format, a...))
}

// load reads, strictly decodes and validates every record, then runs the
// cross-record checks (§5.5). Only valid records land in the lists.
func load(st store.Store) (*snapshot, error) {
	snap := &snapshot{}
	ids := map[string]string{} // id -> file that claimed it first
	for _, k := range domain.Kinds {
		raws, err := st.ReadAll(k)
		if err != nil {
			return nil, err
		}
		for _, raw := range raws {
			file := filepath.ToSlash(filepath.Join(store.Dir, string(k), raw.File+".json"))
			id, err := decodeRecord(snap, k, raw.Data)
			if err != nil {
				snap.problem(file, "%v", err)
				continue
			}
			if id != raw.File {
				snap.problem(file, "id %q does not match the file name", id)
			}
			if first, dup := ids[id]; dup {
				snap.problem(file, "duplicate id %s (also in %s)", id, first)
			} else {
				ids[id] = file
			}
		}
	}
	snap.crossCheck(st.Root, ids)
	snap.sort()
	return snap, nil
}

// decodeRecord appends a valid record of kind k to snap and returns its id.
func decodeRecord(snap *snapshot, k domain.Kind, data []byte) (string, error) {
	switch k {
	case domain.KindStory:
		var r domain.Story
		if err := decodeValid(data, &r, r.Validate); err != nil {
			return "", err
		}
		snap.stories = append(snap.stories, r)
		return r.ID, nil
	case domain.KindDecision:
		var r domain.Decision
		if err := decodeValid(data, &r, r.Validate); err != nil {
			return "", err
		}
		snap.decisions = append(snap.decisions, r)
		return r.ID, nil
	case domain.KindGuardrail:
		var r domain.Guardrail
		if err := decodeValid(data, &r, r.Validate); err != nil {
			return "", err
		}
		snap.guardrails = append(snap.guardrails, r)
		return r.ID, nil
	default:
		var r domain.Trace
		if err := decodeValid(data, &r, r.Validate); err != nil {
			return "", err
		}
		snap.traces = append(snap.traces, r)
		return r.ID, nil
	}
}

// decodeValid decodes into v, then runs validate. validate is a method value
// bound to the same pointer, so it sees the decoded fields.
func decodeValid(data []byte, v any, validate func() error) error {
	if err := store.Decode(data, v); err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}
	return validate()
}

func (s *snapshot) crossCheck(root string, ids map[string]string) {
	pathOK := func(rel string) bool {
		if rel == "" {
			return true
		}
		if filepath.IsAbs(rel) {
			return false
		}
		_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		return err == nil
	}
	refOK := func(id string) bool { return id == "" || ids[id] != "" }
	file := func(k domain.Kind, id string) string { return store.Dir + "/" + string(k) + "/" + id + ".json" }

	for i := range s.stories {
		st := &s.stories[i]
		f := file(domain.KindStory, st.ID)
		if st.Status == "implemented" {
			if issue := st.GateIssue(); issue != "" {
				s.problem(f, "implemented but %s", issue)
			}
		}
		for field, rel := range map[string]string{"packet": st.Packet, "contract": st.Contract} {
			if !pathOK(rel) {
				s.problem(f, "%s %s does not exist", field, rel)
			}
		}
	}
	for _, d := range s.decisions {
		f := file(domain.KindDecision, d.ID)
		if !pathOK(d.Doc) {
			s.problem(f, "doc %s does not exist", d.Doc)
		}
		if !refOK(d.Story) {
			s.problem(f, "story %s does not exist", d.Story)
		}
		if !refOK(d.SupersededBy) {
			s.problem(f, "superseded_by %s does not exist", d.SupersededBy)
		}
	}
	for _, t := range s.traces {
		if !refOK(t.Story) {
			s.problem(file(domain.KindTrace, t.ID), "story %s does not exist", t.Story)
		}
	}
	sort.Strings(s.problems)
}

// sort orders every list by created_at, then id (§5.3).
func (s *snapshot) sort() {
	sort.Slice(s.stories, func(i, j int) bool {
		return less(s.stories[i].CreatedAt, s.stories[i].ID, s.stories[j].CreatedAt, s.stories[j].ID)
	})
	sort.Slice(s.decisions, func(i, j int) bool {
		return less(s.decisions[i].CreatedAt, s.decisions[i].ID, s.decisions[j].CreatedAt, s.decisions[j].ID)
	})
	sort.Slice(s.guardrails, func(i, j int) bool {
		return less(s.guardrails[i].CreatedAt, s.guardrails[i].ID, s.guardrails[j].CreatedAt, s.guardrails[j].ID)
	})
	sort.Slice(s.traces, func(i, j int) bool {
		return less(s.traces[i].CreatedAt, s.traces[i].ID, s.traces[j].CreatedAt, s.traces[j].ID)
	})
}

func less(at1, id1, at2, id2 string) bool {
	if at1 != at2 {
		return at1 < at2
	}
	return id1 < id2
}

func cmdCheck(e *env, args []string) error {
	if err := newFlags("check").parse(args); err != nil {
		return err
	}
	snap, err := load(e.st)
	if err != nil {
		return err
	}
	for _, p := range snap.problems {
		fmt.Fprintln(e.stdout, p)
	}
	if n := len(snap.problems); n > 0 {
		return fmt.Errorf("check: %d problem(s)", n)
	}
	total := len(snap.stories) + len(snap.decisions) + len(snap.guardrails) + len(snap.traces)
	fmt.Fprintf(e.stdout, "ok: %d record(s), no problems\n", total)
	return nil
}
