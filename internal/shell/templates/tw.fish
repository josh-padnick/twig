# twig shell integration (fish): %CMD% resolves a worktree fragment, cd's
# into it in place, then runs the on-entry steps (trust gate + setup).
function %CMD%
    set -l dir (%TWIG% cd $argv); or return $status
    cd $dir; or return $status
    %TWIG% enter
end
