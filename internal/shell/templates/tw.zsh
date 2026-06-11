# twig shell integration (zsh): %CMD% resolves a worktree fragment, cd's
# into it in place, then runs the on-entry steps (trust gate + setup).
%CMD%() {
  local dir
  dir="$(%TWIG% cd "$@")" || return $?
  cd "$dir" || return $?
  %TWIG% enter
}
