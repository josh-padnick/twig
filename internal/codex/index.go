package codex

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"sort"
	"strings"
)

// IndexEntry is one line of ~/.codex/session_index.jsonl: a session id, its
// human thread title, and when it was last touched.
type IndexEntry struct {
	ID        string `json:"id"`
	Title     string `json:"thread_name"`
	UpdatedAt string `json:"updated_at"`
}

// ReadIndex parses session_index.jsonl. A missing file yields no entries
// rather than an error — it just means no sessions have been recorded yet.
func ReadIndex(path string) ([]IndexEntry, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []IndexEntry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1*1024*1024)
	for sc.Scan() {
		var e IndexEntry
		if json.Unmarshal(sc.Bytes(), &e) != nil {
			continue
		}
		if e.ID != "" && e.Title != "" {
			entries = append(entries, e)
		}
	}
	return entries, sc.Err()
}

// match tiers for title search; lower is stronger.
const (
	titleExact  = iota // title equals the query
	titleSubstr        // query is a substring of the title
	titleTokens        // every query word appears in the title, any order
	titleNone
)

// SearchByTitle filters entries to those whose title matches query, keeps only
// the strongest match tier, orders survivors most-recent-first, and dedupes by
// id. All matching is case-insensitive.
func SearchByTitle(entries []IndexEntry, query string) []IndexEntry {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	tokens := strings.Fields(q)

	tierOf := func(title string) int {
		t := strings.ToLower(title)
		switch {
		case t == q:
			return titleExact
		case strings.Contains(t, q):
			return titleSubstr
		case containsAll(t, tokens):
			return titleTokens
		default:
			return titleNone
		}
	}

	best := titleNone
	tiers := make([]int, len(entries))
	for i, e := range entries {
		tiers[i] = tierOf(e.Title)
		if tiers[i] < best {
			best = tiers[i]
		}
	}
	if best == titleNone {
		return nil
	}

	var out []IndexEntry
	for i, e := range entries {
		if tiers[i] == best {
			out = append(out, e)
		}
	}
	// updated_at is ISO-8601, so a lexical descending sort is newest-first.
	sort.SliceStable(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })

	seen := map[string]bool{}
	var deduped []IndexEntry
	for _, e := range out {
		if seen[e.ID] {
			continue // newest per id already won via the sort above
		}
		seen[e.ID] = true
		deduped = append(deduped, e)
	}
	return deduped
}

func containsAll(s string, tokens []string) bool {
	for _, t := range tokens {
		if !strings.Contains(s, t) {
			return false
		}
	}
	return len(tokens) > 0
}
