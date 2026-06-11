// Package trust is twig's direnv-style approval store. Executing scripts
// declared in a repo on "enter" is an attack vector — checking out a PR
// branch could otherwise run attacker code — so twig refuses to use any
// twig.toml until the user approves that exact content at that exact path.
// Approvals are keyed by sha256(path NUL content): editing or moving the
// manifest invalidates them.
package trust

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/josh-padnick/twig/internal/xdg"
)

// Store persists approvals as one JSON file per approved manifest.
type Store struct {
	Dir string
}

// NewStore returns the store at $XDG_DATA_HOME/twig/trust.
func NewStore() (*Store, error) {
	data, err := xdg.DataDir()
	if err != nil {
		return nil, err
	}
	return &Store{Dir: filepath.Join(data, "trust")}, nil
}

// Entry records one approval.
type Entry struct {
	Version       int       `json:"version"`
	Path          string    `json:"path"`
	ContentSHA256 string    `json:"content_sha256"`
	ApprovedAt    time.Time `json:"approved_at"`
}

// key derives the approval filename. The NUL separator is unambiguous
// because POSIX paths cannot contain NUL bytes.
func key(manifestPath string, content []byte) string {
	h := sha256.New()
	h.Write([]byte(manifestPath))
	h.Write([]byte{0})
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

func (s *Store) entryPath(manifestPath string, content []byte) string {
	return filepath.Join(s.Dir, key(manifestPath, content)+".json")
}

// Check reports whether this exact manifest content at this exact path has
// been approved.
func (s *Store) Check(manifestPath string, content []byte) (bool, error) {
	_, err := os.Stat(s.entryPath(manifestPath, content))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Approve records an approval atomically (temp file + rename) so a crash
// can't leave a half-written entry that parses as approval.
func (s *Store) Approve(manifestPath string, content []byte) error {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	sum := sha256.Sum256(content)
	entry := Entry{
		Version:       1,
		Path:          manifestPath,
		ContentSHA256: hex.EncodeToString(sum[:]),
		ApprovedAt:    time.Now().UTC(),
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.Dir, ".approve-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.entryPath(manifestPath, content))
}

// Revoke removes every approval recorded for manifestPath (there can be
// several from successive content versions) and returns how many.
func (s *Store) Revoke(manifestPath string) (int, error) {
	files, err := s.entryFiles()
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, f := range files {
		var e Entry
		if readEntry(f, &e) != nil || e.Path != manifestPath {
			continue
		}
		if err := os.Remove(f); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

// List returns every approval, newest first.
func (s *Store) List() ([]Entry, error) {
	files, err := s.entryFiles()
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, f := range files {
		var e Entry
		if err := readEntry(f, &e); err != nil {
			continue // tolerate stray files in the store dir
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ApprovedAt.After(entries[j].ApprovedAt) })
	return entries, nil
}

func (s *Store) entryFiles() ([]string, error) {
	dirEntries, err := os.ReadDir(s.Dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var files []string
	for _, de := range dirEntries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".json") {
			continue
		}
		files = append(files, filepath.Join(s.Dir, de.Name()))
	}
	return files, nil
}

func readEntry(path string, e *Entry) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, e); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}
	return nil
}
