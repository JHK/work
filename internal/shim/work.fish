# The binary writes the worktree into the file this names, and the function
# changes into it.
function work
    set -l answer (mktemp -t work.XXXXXXXX)
    or return 1
    WORK_CD_FILE=$answer command work $argv
    set -l code $status
    if test $code -eq 0; and read -l dir <"$answer"
        cd $dir
        set code $status
    end
    rm -f "$answer"
    return $code
end
