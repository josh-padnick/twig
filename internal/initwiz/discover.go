// Package initwiz implements the `twig init` wizard's machinery: discover
// likely project roots on disk, detect installed editors, generate a
// commented config.toml, and offer shell-rc installation. The interactive
// prompting lives in the CLI layer; everything here is pure and testable.
package initwiz

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// RootCandidate is a directory whose immediate children include git repos —
// the shape twig's roots config expects.
type RootCandidate struct {
	Path        string
	RepoCount   int // children containing a .git entry
	ClaudeCount int // of those, repos with .claude/worktrees (a strong twig-user signal)
}

// wellKnownBases are the home-relative directories scanned for candidates.
var wellKnownBases = []string{
	"Code", "code", "Projects", "projects", "Developer", "src", "dev", "work", "repos",
}

// Discover scans well-known project locations under home and returns
// candidate roots ranked by Claude-worktree signal, then repo count.
// Both the base itself (~/Projects full of repos) and org dirs one level
// down (~/Code/fabricahq) are considered.
func Discover(home string) []RootCandidate {
	// Dedupe by file identity, not path string: on case-insensitive
	// filesystems (macOS) ~/Code and ~/code are the same directory, and
	// both spellings are in wellKnownBases.
	var seen []os.FileInfo
	var out []RootCandidate
	consider := func(dir string) {
		fi, err := os.Stat(dir)
		if err != nil || !fi.IsDir() {
			return
		}
		for _, prev := range seen {
			if os.SameFile(prev, fi) {
				return
			}
		}
		seen = append(seen, fi)
		if c, ok := examine(dir); ok {
			out = append(out, c)
		}
	}
	for _, base := range wellKnownBases {
		baseDir := filepath.Join(home, base)
		consider(baseDir)
		for _, sub := range childDirs(baseDir) {
			consider(sub)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ClaudeCount != out[j].ClaudeCount {
			return out[i].ClaudeCount > out[j].ClaudeCount
		}
		if out[i].RepoCount != out[j].RepoCount {
			return out[i].RepoCount > out[j].RepoCount
		}
		return out[i].Path < out[j].Path
	})
	return out
}

// examine counts the git repos directly under dir.
func examine(dir string) (RootCandidate, bool) {
	c := RootCandidate{Path: dir}
	for _, child := range childDirs(dir) {
		if _, err := os.Stat(filepath.Join(child, ".git")); err != nil {
			continue
		}
		c.RepoCount++
		if fi, err := os.Stat(filepath.Join(child, ".claude", "worktrees")); err == nil && fi.IsDir() {
			c.ClaudeCount++
		}
	}
	return c, c.RepoCount > 0
}

// childDirs lists immediate subdirectories, skipping dot-entries, capped so
// a giant directory can't stall the wizard.
func childDirs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	const cap = 500
	var out []string
	for _, e := range entries {
		if len(out) >= cap {
			break
		}
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		out = append(out, filepath.Join(dir, e.Name()))
	}
	return out
}

// Editor is an installed editor twig can offer as an opener.
type Editor struct {
	Name        string // opener key in the generated config (the binary name)
	DisplayName string // human name for wizard prompts
	Command     string // opener command template
}

// DetectEditors finds known editor CLIs on PATH.
func DetectEditors() []Editor {
	known := []struct{ bin, name string }{
		{"cursor", "Cursor"},
		{"code", "VS Code"},
		{"zed", "Zed"},
	}
	var out []Editor
	for _, k := range known {
		if _, err := exec.LookPath(k.bin); err == nil {
			out = append(out, Editor{Name: k.bin, DisplayName: k.name, Command: k.bin + " {{dir}}"})
		}
	}
	return out
}

// HasConductor reports whether Conductor's workspace directory exists,
// for the wizard's detected-tools summary.
func HasConductor(home string) bool {
	fi, err := os.Stat(filepath.Join(home, "conductor", "workspaces"))
	return err == nil && fi.IsDir()
}

// HasCodex reports whether the Codex CLI's home directory exists. Codex
// keeps no local worktrees — its sessions live as GitHub branches — so
// the wizard points at remote pickup instead of a scan location.
func HasCodex(home string) bool {
	fi, err := os.Stat(filepath.Join(home, ".codex"))
	return err == nil && fi.IsDir()
}
