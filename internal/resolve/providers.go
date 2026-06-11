// Providers are built-in scan locations for tools that create worktrees in
// known places. They activate with zero configuration when their directories
// exist on disk; users trim the set with the `providers` config key. Adding
// support for a new tool's layout is one entry in Builtin.
package resolve

import "path/filepath"

// Provider contributes scan-parent directories (directories whose immediate
// children are worktree candidates) for one known tool layout.
type Provider struct {
	Name string
	// Parents returns candidate scan parents; nonexistent paths are filtered
	// by the scanner, so implementations list locations unconditionally.
	Parents func(home string, roots []string) []string
}

// Builtin is every provider twig knows about. Codex has no local worktree
// layout (cloud sessions live as remote branches, served by remote pickup),
// so it deliberately has no provider entry.
var Builtin = []Provider{
	{
		// Conductor checks worktrees out under ~/conductor/workspaces/<project>/<name>.
		Name: "conductor",
		Parents: func(home string, roots []string) []string {
			return subdirs(filepath.Join(home, "conductor", "workspaces"))
		},
	},
	{
		// Claude Code desktop keeps worktrees inside the repo at
		// <repo>/.claude/worktrees/<slug>; apply that shape to each root
		// directly and to each repo one level under a root.
		Name: "claude-code",
		Parents: func(home string, roots []string) []string {
			var parents []string
			for _, root := range roots {
				parents = append(parents, filepath.Join(root, ".claude", "worktrees"))
				for _, sub := range subdirs(root) {
					parents = append(parents, filepath.Join(sub, ".claude", "worktrees"))
				}
			}
			return parents
		},
	},
}

// BuiltinNames returns the names of all builtin providers, in order.
func BuiltinNames() []string {
	return providerNames(Builtin)
}

// ByNames returns the builtin providers with the given names, preserving
// Builtin's order and silently skipping unknown names (config validation
// warns about those separately).
func ByNames(names []string) []Provider {
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}
	var out []Provider
	for _, p := range Builtin {
		if want[p.Name] {
			out = append(out, p)
		}
	}
	return out
}

func providerNames(ps []Provider) []string {
	var names []string
	for _, p := range ps {
		names = append(names, p.Name)
	}
	return names
}
