package codex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadIndex(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "session_index.jsonl")
	body := `{"id":"a","thread_name":"Frame PR 142 review","updated_at":"2026-06-15T20:30:42Z"}
not json, skipped
{"id":"b","thread_name":"Refactor auth store","updated_at":"2026-06-14T10:00:00Z"}
{"id":"c","updated_at":"2026-06-13T10:00:00Z"}
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := ReadIndex(path)
	if err != nil {
		t.Fatalf("ReadIndex: %v", err)
	}
	// "c" has no title and is dropped; the malformed line is skipped.
	if len(entries) != 2 {
		t.Fatalf("entries = %+v, want 2", entries)
	}
	if entries[0].ID != "a" || entries[0].Title != "Frame PR 142 review" {
		t.Errorf("entries[0] = %+v", entries[0])
	}
}

func TestReadIndexMissingFile(t *testing.T) {
	entries, err := ReadIndex(filepath.Join(t.TempDir(), "nope.jsonl"))
	if err != nil || entries != nil {
		t.Errorf("missing file: entries=%v err=%v, want nil,nil", entries, err)
	}
}

func TestSearchByTitle(t *testing.T) {
	entries := []IndexEntry{
		{ID: "1", Title: "Frame PR 142 review", UpdatedAt: "2026-06-15T20:00:00Z"},
		{ID: "2", Title: "Frame PR 145 review", UpdatedAt: "2026-06-15T22:00:00Z"},
		{ID: "3", Title: "Refactor the auth store", UpdatedAt: "2026-06-10T00:00:00Z"},
	}

	// Substring beats nothing, and there's only one — exact single match.
	if got := SearchByTitle(entries, "auth store"); len(got) != 1 || got[0].ID != "3" {
		t.Errorf("'auth store' = %+v, want [3]", got)
	}

	// Token match across two sessions; the stronger substring tier ("frame pr
	// 14") would tie both, so they're returned newest-first (145 before 142).
	got := SearchByTitle(entries, "frame review")
	if len(got) != 2 || got[0].ID != "2" || got[1].ID != "1" {
		t.Errorf("'frame review' = %+v, want [2,1] newest-first", got)
	}

	// Exact title wins outright over the other 'frame' session.
	if got := SearchByTitle(entries, "Frame PR 142 review"); len(got) != 1 || got[0].ID != "1" {
		t.Errorf("exact = %+v, want [1]", got)
	}

	// No match, and empty query, both yield nothing.
	if got := SearchByTitle(entries, "kubernetes"); got != nil {
		t.Errorf("no match = %+v, want nil", got)
	}
	if got := SearchByTitle(entries, "   "); got != nil {
		t.Errorf("empty query = %+v, want nil", got)
	}
}

func TestSearchByTitleDedupesByID(t *testing.T) {
	// The index can carry several lines for one session as it's updated; the
	// newest survives and it appears once.
	entries := []IndexEntry{
		{ID: "1", Title: "Frame PR 142 review", UpdatedAt: "2026-06-15T20:00:00Z"},
		{ID: "1", Title: "Frame PR 142 review", UpdatedAt: "2026-06-15T23:00:00Z"},
	}
	got := SearchByTitle(entries, "frame")
	if len(got) != 1 || got[0].UpdatedAt != "2026-06-15T23:00:00Z" {
		t.Errorf("dedup = %+v, want one entry with the newest timestamp", got)
	}
}
