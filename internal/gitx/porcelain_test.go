package gitx

import (
	"reflect"
	"testing"
)

func TestParseWorktrees(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Worktree
	}{
		{
			name: "main plus linked branch worktree",
			in: "worktree /code/app\nHEAD 1111111111111111111111111111111111111111\nbranch refs/heads/main\n\n" +
				"worktree /code/app/.claude/worktrees/foo-1a2b3c\nHEAD 2222222222222222222222222222222222222222\nbranch refs/heads/claude/foo-1a2b3c\n",
			want: []Worktree{
				{Path: "/code/app", Head: "1111111111111111111111111111111111111111", Branch: "main"},
				{Path: "/code/app/.claude/worktrees/foo-1a2b3c", Head: "2222222222222222222222222222222222222222", Branch: "claude/foo-1a2b3c"},
			},
		},
		{
			name: "detached and bare entries",
			in:   "worktree /code/bare.git\nbare\n\nworktree /code/scratch\nHEAD 3333333333333333333333333333333333333333\ndetached\n",
			want: []Worktree{
				{Path: "/code/bare.git", Bare: true},
				{Path: "/code/scratch", Head: "3333333333333333333333333333333333333333", Detached: true},
			},
		},
		{
			name: "locked and prunable with and without reasons",
			in: "worktree /code/a\nHEAD 4444444444444444444444444444444444444444\nbranch refs/heads/a\nlocked\n\n" +
				"worktree /code/b\nHEAD 5555555555555555555555555555555555555555\nbranch refs/heads/b\nlocked reason with spaces\nprunable gitdir file points to non-existent location\n",
			want: []Worktree{
				{Path: "/code/a", Head: "4444444444444444444444444444444444444444", Branch: "a", Locked: true},
				{Path: "/code/b", Head: "5555555555555555555555555555555555555555", Branch: "b", Locked: true, Prunable: true},
			},
		},
		{
			name: "unknown attribute lines are ignored",
			in:   "worktree /code/c\nHEAD 6666666666666666666666666666666666666666\nbranch refs/heads/c\nfuture-attribute some value\n",
			want: []Worktree{
				{Path: "/code/c", Head: "6666666666666666666666666666666666666666", Branch: "c"},
			},
		},
		{
			name: "empty input",
			in:   "",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseWorktrees(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseWorktrees() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
