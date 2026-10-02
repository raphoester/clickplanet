#!/usr/bin/env bash

cp_normalize_hooks_path() {
    local want=.githooks

    if [ "$(git config --get core.hooksPath 2>/dev/null || true)" != "$want" ]; then
        git config core.hooksPath "$want" 2>/dev/null || true
        if [ "$(git config --bool --get extensions.worktreeConfig 2>/dev/null || true)" = "true" ]; then
            git config --worktree core.hooksPath "$want" 2>/dev/null || true
        fi
    fi

    local toplevel here correct
    toplevel=$(git rev-parse --show-toplevel 2>/dev/null || true)
    [ -n "$toplevel" ] || return 0
    here=$(cd "$(dirname "$0")" >/dev/null 2>&1 && pwd)/$(basename "$0")
    correct="$toplevel/$want/$(basename "$0")"
    if [ "$here" != "$correct" ] && [ -x "$correct" ]; then
        exec "$correct" "$@"
    fi
}

cp_normalize_hooks_path "$@"
