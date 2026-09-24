// Package install writes the embedded payload into a target repo (§5.7),
// ported from harness-experimental: plan every write first (zero writes on
// a refused plan, I8), protected-path conflict model, symlink refusal, and
// umask-independent modes for newly created paths only.
//
// Records under .harness/<kind>/ are never payload files and are never
// moved, overwritten or backed up by any mode.
package install

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	justharness "github.com/sting8k/just-harness"
)

const (
	defaultFileMode fs.FileMode = 0o644
	defaultDirMode  fs.FileMode = 0o755
	execBinaryMode  fs.FileMode = 0o755

	shimBegin = "<!-- HARNESS:BEGIN -->"
	shimEnd   = "<!-- HARNESS:END -->"
)

// protectedPaths refuse an install unless --merge or --override.
var protectedPaths = []string{"AGENTS.md", "docs", ".harness"}

// overridePaths are moved to backup by --override. .harness/ is not: it holds records.
var overridePaths = []string{"AGENTS.md", "docs"}

// lineMergeFiles are never replaced when they exist; missing lines are appended.
var lineMergeFiles = map[string][]string{
	".gitignore":     {".harness/bin/"},
	".gitattributes": {".harness/** text eol=lf"},
}

const lineMergeMarker = "# just-harness"

// BinaryRel is where the CLI copies itself inside the target.
func BinaryRel() string {
	if runtime.GOOS == "windows" {
		return ".harness/bin/just-harness-cli.exe"
	}
	return ".harness/bin/just-harness-cli"
}

// Options configures one install invocation.
type Options struct {
	Target           string
	Merge            bool
	Override         bool
	Force            bool
	DryRun           bool
	RefreshAgentShim bool
	Claude           bool
	SelfBinary       string // self-copy source override (tests); default os.Executable()
}

type actionKind int

const (
	kindCreateFile actionKind = iota
	kindOverwriteFile
	kindSkipExisting
	kindMergeLines
	kindCreateDir
	kindMoveToBackup
	kindReplaceBinary
	kindRefreshShim
)

type action struct {
	rel    string // path relative to target (slash-separated)
	kind   actionKind
	backup string // backup path relative to target
	src    []byte // content to write
	mode   fs.FileMode
}

// Plan is a fully computed install plan; nothing is written before Execute.
type Plan struct {
	opts       Options
	actions    []action
	created    int
	updated    int
	skipped    int
	backup     string
	movedRoots []string
}

// RefusedError reports protected-path conflicts with actionable guidance.
type RefusedError struct {
	Conflicts []string
	Target    string
}

func (e *RefusedError) Error() string {
	return fmt.Sprintf("%s already contains %s. Refusing to install so existing instructions, docs or records are not mixed or overwritten. "+
		"Use --merge to keep existing files, add missing ones and upgrade (records are never touched), "+
		"or --override to back up AGENTS.md and docs/ and install fresh.",
		e.Target, strings.Join(e.Conflicts, ", "))
}

// PayloadFiles returns the sorted relative paths of every shipped file.
func PayloadFiles() ([]string, error) {
	var files []string
	err := fs.WalkDir(justharness.Payload, "harness", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, strings.TrimPrefix(p, "harness/"))
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func readPayload(rel string) ([]byte, error) {
	return justharness.Payload.ReadFile("harness/" + rel)
}

// NewPlan computes the complete install plan without writing anything.
func NewPlan(opts Options) (*Plan, error) {
	p := &Plan{opts: opts}
	root, err := resolveRoot(opts.Target)
	if err != nil {
		return nil, err
	}
	p.opts.Target = root
	if _, err := os.Lstat(root); errors.Is(err, os.ErrNotExist) {
		p.actions = append(p.actions, action{rel: root, kind: kindCreateDir, mode: defaultDirMode})
	}

	conflicts, err := existing(root, protectedPaths)
	if err != nil {
		return nil, err
	}
	if len(conflicts) > 0 && !opts.Merge && !opts.Override {
		return nil, &RefusedError{Conflicts: conflicts, Target: root}
	}
	if opts.Override {
		moved, err := existing(root, overridePaths)
		if err != nil {
			return nil, err
		}
		p.movedRoots = moved
		for _, rel := range moved {
			p.actions = append(p.actions, action{rel: rel, kind: kindMoveToBackup, backup: p.backupOrNew() + "/" + rel})
		}
	}

	files, err := PayloadFiles()
	if err != nil {
		return nil, err
	}
	for _, d := range payloadDirsParentFirst(append(files, BinaryRel())) {
		if err := p.planDir(d); err != nil {
			return nil, err
		}
	}
	for _, rel := range files {
		content, err := readPayload(rel)
		if err != nil {
			return nil, err
		}
		if err := p.planPayloadFile(rel, content); err != nil {
			return nil, err
		}
	}
	if opts.Claude {
		if err := p.planFile("CLAUDE.md", []byte("@AGENTS.md\n"), defaultFileMode); err != nil {
			return nil, err
		}
	}
	if err := p.planSelfCopy(); err != nil {
		return nil, err
	}

	// I8: every planned destination, its ancestors and its backup path must
	// be symlink-free, so writes and backups cannot escape the target.
	for _, a := range p.actions {
		if a.kind == kindSkipExisting {
			continue
		}
		for _, rel := range []string{a.rel, a.backup} {
			if err := p.assertNoSymlinkPath(rel); err != nil {
				return nil, err
			}
		}
	}
	return p, nil
}

func existing(root string, rels []string) ([]string, error) {
	var out []string
	for _, rel := range rels {
		if _, err := os.Lstat(filepath.Join(root, rel)); err == nil {
			out = append(out, rel)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	return out, nil
}

// underMovedRoot: paths moved away by --override count as absent at plan time.
func (p *Plan) underMovedRoot(rel string) bool {
	for _, root := range p.movedRoots {
		if rel == root || strings.HasPrefix(rel, root+"/") {
			return true
		}
	}
	return false
}

// exists reports whether rel exists in the post-move view of the target.
func (p *Plan) exists(rel string) (bool, error) {
	if p.underMovedRoot(rel) {
		return false, nil
	}
	_, err := os.Lstat(filepath.Join(p.opts.Target, filepath.FromSlash(rel)))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func (p *Plan) planDir(rel string) error {
	ok, err := p.exists(rel)
	if err != nil || ok {
		return err
	}
	p.actions = append(p.actions, action{rel: rel, kind: kindCreateDir, mode: defaultDirMode})
	return nil
}

func (p *Plan) planPayloadFile(rel string, content []byte) error {
	if lines, ok := lineMergeFiles[rel]; ok {
		return p.planLineMerge(rel, content, lines)
	}
	if rel == "AGENTS.md" {
		return p.planAgents(content)
	}
	return p.planFile(rel, content, defaultFileMode)
}

// planFile: create when absent; with --force overwrite after backup; else skip.
func (p *Plan) planFile(rel string, content []byte, mode fs.FileMode) error {
	ok, err := p.exists(rel)
	if err != nil {
		return err
	}
	switch {
	case !ok:
		p.actions = append(p.actions, action{rel: rel, kind: kindCreateFile, src: content, mode: mode})
		p.created++
	case p.opts.Force:
		p.actions = append(p.actions, action{rel: rel, kind: kindOverwriteFile, src: content, mode: mode, backup: p.backupOrNew() + "/" + rel})
		p.updated++
	default:
		p.actions = append(p.actions, action{rel: rel, kind: kindSkipExisting})
		p.skipped++
	}
	return nil
}

// planAgents: an existing AGENTS.md (--merge upgrade) gets its HARNESS block
// refreshed in place, keeping the user's text; --force replaces it wholesale.
func (p *Plan) planAgents(content []byte) error {
	ok, err := p.exists("AGENTS.md")
	if err != nil {
		return err
	}
	if !ok || p.opts.Force {
		return p.planFile("AGENTS.md", content, defaultFileMode)
	}
	current, err := os.ReadFile(filepath.Join(p.opts.Target, "AGENTS.md"))
	if err != nil {
		return err
	}
	updated := refreshShim(current, content)
	if bytes.Equal(updated, current) {
		p.actions = append(p.actions, action{rel: "AGENTS.md", kind: kindSkipExisting})
		p.skipped++
		return nil
	}
	p.actions = append(p.actions, action{rel: "AGENTS.md", kind: kindRefreshShim, src: updated, mode: defaultFileMode, backup: p.backupOrNew() + "/AGENTS.md"})
	p.updated++
	return nil
}

// refreshShim replaces the BEGIN..END span of current with the payload's,
// or appends the payload block when current has no markers.
func refreshShim(current, payload []byte) []byte {
	block := payload
	if i, j := bytes.Index(payload, []byte(shimBegin)), bytes.LastIndex(payload, []byte(shimEnd)); i >= 0 && j > i {
		block = payload[i : j+len(shimEnd)]
	}
	i, j := bytes.Index(current, []byte(shimBegin)), bytes.LastIndex(current, []byte(shimEnd))
	var out []byte
	if i >= 0 && j > i {
		out = append(out, current[:i]...)
		out = append(out, block...)
		return append(out, current[j+len(shimEnd):]...)
	}
	out = append(out, current...)
	if len(out) > 0 && out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	if len(out) > 0 {
		out = append(out, '\n')
	}
	out = append(out, block...)
	return append(out, '\n')
}

// planLineMerge never replaces a user's .gitignore/.gitattributes; it appends
// the harness lines that are missing.
func (p *Plan) planLineMerge(rel string, content []byte, lines []string) error {
	ok, err := p.exists(rel)
	if err != nil {
		return err
	}
	if !ok {
		p.actions = append(p.actions, action{rel: rel, kind: kindCreateFile, src: content, mode: defaultFileMode})
		p.created++
		return nil
	}
	current, err := os.ReadFile(filepath.Join(p.opts.Target, rel))
	if err != nil {
		return err
	}
	have := map[string]bool{}
	for _, l := range strings.Split(string(current), "\n") {
		have[strings.TrimSpace(l)] = true
	}
	var missing []string
	for _, l := range lines {
		if !have[l] {
			missing = append(missing, l)
		}
	}
	if len(missing) == 0 {
		p.actions = append(p.actions, action{rel: rel, kind: kindSkipExisting})
		p.skipped++
		return nil
	}
	var b bytes.Buffer
	b.Write(current)
	if len(current) > 0 && current[len(current)-1] != '\n' {
		b.WriteByte('\n')
	}
	b.WriteString(lineMergeMarker + "\n" + strings.Join(missing, "\n") + "\n")
	p.actions = append(p.actions, action{rel: rel, kind: kindMergeLines, src: b.Bytes(), mode: defaultFileMode})
	p.updated++
	return nil
}

// planSelfCopy installs this binary into .harness/bin/. The binary is
// gitignored and reproducible, so it is replaced (tmp + rename) without backup.
func (p *Plan) planSelfCopy() error {
	src := p.opts.SelfBinary
	if src == "" {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		src = exe
	}
	rel := BinaryRel()
	dest := filepath.Join(p.opts.Target, filepath.FromSlash(rel))
	if a, errA := filepath.EvalSymlinks(src); errA == nil {
		if b, errB := filepath.EvalSymlinks(dest); errB == nil && a == b {
			p.actions = append(p.actions, action{rel: rel, kind: kindSkipExisting})
			p.skipped++
			return nil
		}
	}
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("self binary: %w", err)
	}
	ok, err := p.exists(rel)
	if err != nil {
		return err
	}
	if ok {
		if current, err := os.ReadFile(dest); err == nil && bytes.Equal(current, content) {
			p.actions = append(p.actions, action{rel: rel, kind: kindSkipExisting})
			p.skipped++
			return nil
		}
		p.updated++
	} else {
		p.created++
	}
	p.actions = append(p.actions, action{rel: rel, kind: kindReplaceBinary, src: content, mode: execBinaryMode})
	return nil
}

func (p *Plan) backupOrNew() string {
	if p.backup == "" {
		p.backup = ".harness-backup/" + time.Now().Format("20060102150405")
	}
	return p.backup
}

// assertNoSymlinkPath refuses rel when it or any ancestor below the target
// is a symlink. The root-creation action (absolute) is skipped.
func (p *Plan) assertNoSymlinkPath(rel string) error {
	if rel == "" || filepath.IsAbs(rel) {
		return nil
	}
	parts := strings.Split(rel, "/")
	for i := 1; i <= len(parts); i++ {
		sub := filepath.Join(p.opts.Target, filepath.Join(parts[:i]...))
		info, err := os.Lstat(sub)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to install through symlink: %s", sub)
		}
	}
	return nil
}

// resolveRoot canonicalizes the target without creating anything. Symlinks
// in the target itself or its prefix are followed (macOS /tmp); an absent
// target needs an existing parent.
func resolveRoot(target string) (string, error) {
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(abs); err == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("target is not a directory: %s", abs)
		}
		return filepath.EvalSymlinks(abs)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	parent := filepath.Dir(abs)
	if info, err := os.Stat(parent); err != nil || !info.IsDir() {
		return "", fmt.Errorf("parent directory does not exist: %s", parent)
	}
	canon, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(canon, filepath.Base(abs)), nil
}

// payloadDirsParentFirst lists every directory holding a file, shallow first.
func payloadDirsParentFirst(files []string) []string {
	set := map[string]bool{}
	for _, rel := range files {
		for d := pathDir(rel); d != ""; d = pathDir(d) {
			set[d] = true
		}
	}
	out := make([]string, 0, len(set))
	for d := range set {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		di, dj := strings.Count(out[i], "/"), strings.Count(out[j], "/")
		if di != dj {
			return di < dj
		}
		return out[i] < out[j]
	})
	return out
}

func pathDir(rel string) string {
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		return rel[:i]
	}
	return ""
}

// Execute applies the plan. A refused plan never reaches Execute; a crash
// mid-way may leave partial files (rerun with --merge recovers).
func (p *Plan) Execute(stdout io.Writer) error {
	if p.opts.DryRun {
		fmt.Fprintln(stdout, "Dry run: no files will be written.")
	}
	for _, a := range p.actions {
		if !p.opts.DryRun {
			if err := p.apply(a); err != nil {
				return err
			}
		}
		p.log(stdout, a)
	}
	fmt.Fprintf(stdout, "\nDone. Created: %d, updated: %d, skipped: %d.\n", p.created, p.updated, p.skipped)
	return nil
}

func (p *Plan) abs(rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(p.opts.Target, filepath.FromSlash(rel))
}

func (p *Plan) apply(a action) error {
	target := p.abs(a.rel)
	if a.backup != "" && a.kind != kindMoveToBackup {
		if err := copyFile(target, p.abs(a.backup)); err != nil {
			return err
		}
	}
	switch a.kind {
	case kindMoveToBackup:
		if err := os.MkdirAll(filepath.Dir(p.abs(a.backup)), defaultDirMode); err != nil {
			return err
		}
		return os.Rename(target, p.abs(a.backup))
	case kindCreateDir:
		if err := os.MkdirAll(target, a.mode); err != nil {
			return err
		}
		return os.Chmod(target, a.mode) // only newly created dirs are planned
	case kindCreateFile, kindOverwriteFile, kindMergeLines, kindRefreshShim:
		return writeFile(target, a.src, a.mode)
	case kindReplaceBinary:
		return replaceFile(target, a.src, a.mode)
	}
	return nil
}

func writeFile(path string, content []byte, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), defaultDirMode); err != nil {
		return err
	}
	if err := os.WriteFile(path, content, mode); err != nil {
		return err
	}
	return os.Chmod(path, mode) // umask-independent
}

// replaceFile writes via tmp + rename so a running binary is never truncated.
func replaceFile(path string, content []byte, mode fs.FileMode) error {
	tmp := path + ".tmp"
	if err := writeFile(tmp, content, mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func copyFile(src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return writeFile(dst, content, defaultFileMode)
}

func (p *Plan) log(w io.Writer, a action) {
	verb := map[actionKind][2]string{ // {done, dry-run}
		kindCreateFile:    {"created ", "create  "},
		kindOverwriteFile: {"updated ", "update  "},
		kindSkipExisting:  {"skip    ", "skip    "},
		kindMergeLines:    {"merged  ", "merge   "},
		kindCreateDir:     {"mkdir   ", "mkdir   "},
		kindMoveToBackup:  {"moved   ", "move    "},
		kindReplaceBinary: {"binary  ", "binary  "},
		kindRefreshShim:   {"refresh ", "refresh "},
	}[a.kind]
	v := verb[0]
	if p.opts.DryRun {
		v = verb[1]
	}
	line := v + " " + a.rel
	if a.backup != "" {
		line += " (backup: " + a.backup + ")"
	}
	fmt.Fprintln(w, line)
}
