// Pull-request pickup: twig accepts a GitHub PR URL in place of a fragment and
// resolves it to the PR's remote branch when git can do that safely. It maps
// the URL back to the PR's head branch using git alone — `git ls-remote
// refs/pull/<n>/head` against the local checkout whose remote is that repo,
// then the matching unique non-default branch by SHA — so no GitHub API or
// token is involved. The head branch must live in the same repo (fork PRs
// aren't resolvable this way) and still exist; from there the normal
// fetch-and-worktree flow takes over.
package remote

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/josh-padnick/twig/internal/gitx"
	"github.com/josh-padnick/twig/internal/resolve"
)

// PullRequest identifies a pull request parsed from a URL.
type PullRequest struct {
	Host   string // e.g. github.com
	Owner  string // repo owner / org
	Repo   string // repo name
	Number int    // PR number
	URL    string // the original URL, for messages
}

// pullURLRe matches a GitHub-style pull request URL. The trailing path is
// left unanchored so a URL that points somewhere inside the PR
// (…/pull/140/files, …/pull/140/changes, …?diff=split) still matches.
var pullURLRe = regexp.MustCompile(`^https?://([^/]+)/([^/?#]+)/([^/?#]+)/pull/(\d+)`)

// ParsePullURL recognizes a GitHub-style pull request URL and extracts the
// repo and PR number. The URL need not point at the PR root.
func ParsePullURL(raw string) (PullRequest, bool) {
	m := pullURLRe.FindStringSubmatch(strings.TrimSpace(raw))
	if m == nil {
		return PullRequest{}, false
	}
	n, err := strconv.Atoi(m[4])
	if err != nil {
		return PullRequest{}, false
	}
	host := m[1]
	if colon := strings.Index(host, ":"); colon >= 0 {
		host = host[:colon] // strip a :port
	}
	return PullRequest{
		Host:   host,
		Owner:  m[2],
		Repo:   strings.TrimSuffix(m[3], ".git"),
		Number: n,
		URL:    strings.TrimSpace(raw),
	}, true
}

// PullError reports that a PR URL could not be turned into a local fetch.
type PullError struct {
	PR     PullRequest
	Reason string
}

func (e *PullError) Error() string {
	return fmt.Sprintf("could not resolve %s: %s", e.PR.URL, e.Reason)
}

// ResolvePullRequest finds the PR's head branch by asking the remotes of the
// local repos that point at the PR's repo. It only returns a match when the PR
// head commit maps to exactly one non-default branch; if the branch was deleted
// after a linear merge, refs/pull/<n>/head may equal the default branch tip, and
// resolving that would enter the wrong worktree. The caller fetches and creates
// the worktree as it would for any remote-branch pickup.
func ResolvePullRequest(pr PullRequest, repos []string) ([]Match, error) {
	foundRepo := false
	for _, repo := range repos {
		remotes, err := gitx.Remotes(repo)
		if err != nil {
			continue
		}
		for _, rem := range remotes {
			url, err := gitx.RemoteURL(repo, rem)
			if err != nil {
				continue
			}
			host, owner, name, ok := parseRepoURL(url)
			if !ok || !sameRepo(host, owner, name, pr) {
				continue
			}
			foundRepo = true
			// This remote is the PR's repo. Ask it where the PR head sits,
			// then map that commit back to a branch the remote still has.
			sha, err := gitx.LsRemotePullHead(repo, rem, pr.Number)
			if err != nil || sha == "" {
				continue
			}
			defaultBranch, err := gitx.LsRemoteDefaultBranch(repo, rem)
			if err != nil {
				continue
			}
			heads, err := gitx.LsRemoteHeadRefs(repo, rem)
			if err != nil {
				continue
			}
			var matches []Match
			for _, h := range heads {
				if h.SHA == sha && h.Branch != defaultBranch {
					matches = append(matches, Match{
						RepoDir: repo, Remote: rem, Branch: h.Branch, Tier: resolve.TierExactBranch,
					})
				}
			}
			if len(matches) == 1 {
				return matches, nil
			}
			if len(matches) > 1 {
				return nil, &PullError{PR: pr, Reason: "more than one remote branch points at the PR head commit — can't choose safely without the PR head branch name"}
			}
		}
	}
	if !foundRepo {
		return nil, &PullError{PR: pr, Reason: fmt.Sprintf(
			"no local checkout of %s/%s/%s found under your roots — twig needs the repo on disk to fetch from",
			pr.Host, pr.Owner, pr.Repo)}
	}
	return nil, &PullError{PR: pr, Reason: "couldn't map the PR to a branch on that remote — its head branch may live in a fork or have been deleted"}
}

// sameRepo reports whether a remote's parsed coordinates name the PR's repo.
// Owner and repo must match; host is only compared when both sides carry one
// (filesystem-path remotes have no host), so an owner/repo match is enough
// for a local mirror while github.com still disambiguates when present.
func sameRepo(host, owner, repo string, pr PullRequest) bool {
	if !strings.EqualFold(owner, pr.Owner) || !strings.EqualFold(repo, pr.Repo) {
		return false
	}
	if host != "" && pr.Host != "" && !strings.EqualFold(host, pr.Host) {
		return false
	}
	return true
}

// parseRepoURL extracts host/owner/repo from a git remote URL, covering the
// https, ssh, scp-like (git@host:owner/repo), and bare filesystem-path forms.
// host is "" when the URL carries no recognizable host (a filesystem path),
// in which case sameRepo matches on owner/repo alone.
func parseRepoURL(raw string) (host, owner, repo string, ok bool) {
	s := strings.TrimSpace(raw)
	s = strings.TrimSuffix(s, "/")
	s = strings.TrimSuffix(s, ".git")

	var path string
	switch {
	case strings.Contains(s, "://"):
		rest := s[strings.Index(s, "://")+3:]
		if at := strings.LastIndex(rest, "@"); at >= 0 {
			rest = rest[at+1:] // drop user[:pass]@
		}
		slash := strings.Index(rest, "/")
		if slash < 0 {
			return "", "", "", false
		}
		host, path = rest[:slash], rest[slash+1:]
	case strings.Contains(s, "@") && strings.Contains(s, ":") && !filepath.IsAbs(s):
		// scp-like: [user@]host:owner/repo
		hostpath := s[strings.LastIndex(s, "@")+1:]
		colon := strings.Index(hostpath, ":")
		host, path = hostpath[:colon], hostpath[colon+1:]
	default:
		path = s // filesystem path or anything else: match on the trailing segments
	}
	if colon := strings.Index(host, ":"); colon >= 0 {
		host = host[:colon] // strip a :port
	}
	parts := splitNonEmpty(path)
	if len(parts) < 2 {
		return "", "", "", false
	}
	return host, parts[len(parts)-2], parts[len(parts)-1], true
}

func splitNonEmpty(path string) []string {
	var out []string
	for _, p := range strings.Split(path, "/") {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
