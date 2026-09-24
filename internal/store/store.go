// Package store finds the repo root and reads/writes .harness/ records:
// deterministic JSON, atomic replace, collision-free random IDs (I5, I7).
package store

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	justharness "github.com/sting8k/just-harness"
	"github.com/sting8k/just-harness/internal/domain"
)

// Dir is the record directory at the repo root.
const Dir = ".harness"

// ErrNoRoot means no .harness/ was found from cwd upward.
var ErrNoRoot = errors.New("no .harness/ directory found from the current directory upward; run `just-harness-cli install .` first")

// FindRoot returns $HARNESS_REPO_ROOT or the nearest ancestor of start that
// contains .harness/ (§5.2).
func FindRoot(start string) (string, error) {
	if env := os.Getenv("HARNESS_REPO_ROOT"); env != "" {
		return filepath.Abs(env)
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, Dir)); err == nil && info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNoRoot
		}
		dir = parent
	}
}

// Store reads and writes records under Root.
type Store struct {
	Root string
}

// Now is the record clock (UTC, second precision).
func Now() string { return time.Now().UTC().Format(time.RFC3339) }

// Path returns the record file path for id.
func (s Store) Path(k domain.Kind, id string) string {
	return filepath.Join(s.Root, Dir, string(k), id+".json")
}

// Marshal serializes deterministically: struct key order, 2-space indent,
// no HTML escaping, LF, trailing newline, empty fields omitted (I7).
func Marshal(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Decode parses a record strictly (unknown fields rejected). CRLF is
// accepted because JSON treats \r as whitespace (Q18).
func Decode(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("trailing data after JSON object")
	}
	return nil
}

// Load reads one record into v.
func (s Store) Load(k domain.Kind, id string, v any) error {
	data, err := os.ReadFile(s.Path(k, id))
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s not found", id)
	}
	if err != nil {
		return err
	}
	if err := Decode(data, v); err != nil {
		return fmt.Errorf("%s: %w", s.Path(k, id), err)
	}
	return nil
}

// Exists reports whether a record file exists.
func (s Store) Exists(k domain.Kind, id string) bool {
	_, err := os.Lstat(s.Path(k, id))
	return err == nil
}

// Save atomically replaces (or creates) the record for id.
func (s Store) Save(k domain.Kind, id string, v any) error {
	data, err := Marshal(v)
	if err != nil {
		return err
	}
	return writeAtomic(s.Path(k, id), data)
}

// Create writes a new record and refuses to overwrite an existing one (I5).
func (s Store) Create(k domain.Kind, id string, v any) error {
	if s.Exists(k, id) {
		return fmt.Errorf("refusing to overwrite existing record %s", id)
	}
	return s.Save(k, id, v)
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// NewID returns an unused random ID "<prefix>-xxxx" for kind k (Q12, I5).
func (s Store) NewID(k domain.Kind) (string, error) {
	max := big.NewInt(int64(len(domain.IDAlphabet)))
	for attempt := 0; attempt < 100; attempt++ {
		var b strings.Builder
		b.WriteString(k.Prefix() + "-")
		for i := 0; i < domain.IDLen; i++ {
			n, err := rand.Int(rand.Reader, max)
			if err != nil {
				return "", err
			}
			b.WriteByte(domain.IDAlphabet[n.Int64()])
		}
		if id := b.String(); !s.Exists(k, id) {
			return id, nil
		}
	}
	return "", errors.New("could not allocate a free id")
}

// RawRecord is one record file as found on disk, for listing and `check`.
type RawRecord struct {
	Kind domain.Kind
	File string // file stem (expected to equal the id)
	Data []byte
}

// ReadAll returns every *.json file under .harness/<kind>/, sorted by name.
func (s Store) ReadAll(k domain.Kind) ([]RawRecord, error) {
	dir := filepath.Join(s.Root, Dir, string(k))
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []RawRecord
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, RawRecord{Kind: k, File: strings.TrimSuffix(e.Name(), ".json"), Data: data})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out, nil
}

// ProsePath returns the repo-relative path docs/<dir>/<ID>-<slug>.md (Q19).
func ProsePath(dir, id, title string) string {
	name := id
	if slug := domain.Slug(title); slug != "" {
		name += "-" + slug
	}
	return "docs/" + dir + "/" + name + ".md"
}

// CreateProse renders docs/templates/<template> (repo copy, else embedded
// payload) with {{id}}/{{title}} into rel. Never overwrites (I5).
func (s Store) CreateProse(rel, template, id, title string) error {
	tpl, err := os.ReadFile(filepath.Join(s.Root, "docs", "templates", template))
	if errors.Is(err, fs.ErrNotExist) {
		tpl, err = justharness.Payload.ReadFile("harness/docs/templates/" + template)
	}
	if err != nil {
		return fmt.Errorf("template %s: %w", template, err)
	}
	body := strings.NewReplacer("{{id}}", id, "{{title}}", title).Replace(string(tpl))
	path := filepath.Join(s.Root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("refusing to overwrite existing %s", rel)
		}
		return err
	}
	if _, err := f.WriteString(body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
