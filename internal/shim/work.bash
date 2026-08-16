# The binary writes the worktree into the file this names, and the function
# changes into it.
work() {
    # A completion hands back no worktree, so it needs no file to hand it in.
    if [[ $1 == __complete* ]]; then
        command work "$@"
        return
    fi
    local answer code dir
    answer=$(mktemp -t work.XXXXXXXX) || return 1
    WORK_CD_FILE=$answer command work "$@"
    code=$?
    if [ "$code" -eq 0 ] && IFS= read -r dir <"$answer"; then
        cd -- "$dir"
        code=$?
    fi
    rm -f "$answer"
    return "$code"
}
