#!/usr/bin/env bash
# Sourced by the hooks in this directory.
#
# Worktree tooling (and stale manual `git config`) can pin core.hooksPath to an
# ABSOLUTE path, which always resolves to the main checkout's .githooks — so
# every worktree ends up running the wrong branch's hooks. This repairs that on
# the fly:
#
#   1. Rewrites core.hooksPath back to the relative ".githooks". Git resolves a
#      relative hooksPath against each worktree's own root (githooks(5): it cd's
#      to the working-tree root before invoking a hook), so every worktree then
#      runs its own copy.
#   2. If this hook was still launched from the wrong copy, it hands off — via
#      exec — to this worktree's copy, so even this run uses the right checks.

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
