// POSIX shell quoting for commands twig composes and injects into
// terminals. Paths and commands are never interpolated into AppleScript
// source or shell lines unquoted.
package opener

import "strings"

// Quote returns s as a single POSIX shell word using single quotes, with
// embedded single quotes handled via the '\'' idiom.
func Quote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
