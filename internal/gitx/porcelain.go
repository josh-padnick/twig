// Parser for `git worktree list --porcelain` output. Lives apart from the
// exec wrapper so it can be tested against canned strings covering format
// variations across git versions (locked/prunable annotations appeared in
// git 2.36; unknown lines must be ignored, not rejected).
package gitx

import "strings"

// Worktree is one entry from `git worktree list --porcelain`.
type Worktree struct {
	Path     string // absolute path of the worktree
	Head     string // commit sha, "" for a bare entry
	Branch   string // short branch name, "" when detached or bare
	Bare     bool
	Detached bool
	Locked   bool
	Prunable bool // git's own "directory is gone" flag (git 2.36+)
}

// ParseWorktrees parses porcelain output into entries. Blocks are separated
// by blank lines; attribute lines for unknown attributes are skipped so the
// parser tolerates format additions in newer git versions.
func ParseWorktrees(out string) []Worktree {
	var wts []Worktree
	var cur *Worktree
	flush := func() {
		if cur != nil {
			wts = append(wts, *cur)
			cur = nil
		}
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			flush()
			continue
		}
		if rest, ok := strings.CutPrefix(line, "worktree "); ok {
			flush()
			cur = &Worktree{Path: rest}
			continue
		}
		if cur == nil {
			continue // stray line before any worktree header
		}
		switch {
		case strings.HasPrefix(line, "HEAD "):
			cur.Head = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			cur.Branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
		case line == "bare":
			cur.Bare = true
		case line == "detached":
			cur.Detached = true
		case line == "locked" || strings.HasPrefix(line, "locked "):
			cur.Locked = true
		case line == "prunable" || strings.HasPrefix(line, "prunable "):
			cur.Prunable = true
		}
	}
	flush()
	return wts
}
